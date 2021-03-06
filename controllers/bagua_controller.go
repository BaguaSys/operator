/*
Copyright (c) 2021 Kuaishou AI Platform & DS3 Lab

Permission is hereby granted, free of charge, to any person obtaining
a copy of this software and associated documentation files (the
"Software"), to deal in the Software without restriction, including
without limitation the rights to use, copy, modify, merge, publish,
distribute, sublicense, and/or sell copies of the Software, and to
permit persons to whom the Software is furnished to do so, subject to
the following conditions:

The above copyright notice and this permission notice shall be
included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/kubeflow/common/pkg/controller.v1/common"
	"github.com/kubeflow/common/pkg/controller.v1/control"
	"github.com/kubeflow/common/pkg/controller.v1/expectation"
	"github.com/kubeflow/common/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	api "github.com/BaguaSys/operator/api/v1alpha1"
	config "github.com/BaguaSys/operator/conf"
)

// BaguaReconciler reconciles a Bagua object
type BaguaReconciler struct {
	common.JobController
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	Config *config.OperatorConfig
}

//+kubebuilder:rbac:groups=bagua.kuaishou.com,resources=baguas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=bagua.kuaishou.com,resources=baguas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=bagua.kuaishou.com,resources=baguas/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *BaguaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	bagua := &api.Bagua{}
	err := r.Get(ctx, req.NamespacedName, bagua)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		// requeue
		return ctrl.Result{}, err
	}
	job := bagua.DeepCopy()
	log := util.LoggerForJob(bagua)
	log.Infof(" >>> reconcile start")
	defer func() {
		log.Infof(" <<< reconcile end")
	}()

	if job.DeletionTimestamp != nil {
		log.Info("reconcile skipped, job has been deleted.")
		return ctrl.Result{}, nil
	}
	if IsInvalidDefinition(job.Status) {
		return ctrl.Result{}, nil
	}

	// Set default priorities for job
	if updated, err := r.setDefault(bagua, job); err != nil {
		return ctrl.Result{}, err
	} else if updated {
		return ctrl.Result{}, nil
	}
	// Validate definition
	if err := r.validate(job); err != nil {
		// do not need requeue
		log.Errorf("failed to validate: %v", err)
		return ctrl.Result{}, nil
	}

	needSync := r.SatisfiedExpectations(job)
	if !needSync {
		log.Info("reconcile skipped, expectation is not satisfied")
		return ctrl.Result{}, nil
	}

	// Use common to reconcile the job related pod and service
	err = r.JobController.ReconcileJobs(job, job.Spec.ReplicaSpecs, job.Status.JobStatus, &job.Spec.RunPolicy)
	if err != nil {
		log.Errorf("reconcile error %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BaguaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Pod{}, api.JobOwnerKey, func(rawObj client.Object) []string {
		pod := rawObj.(*corev1.Pod)
		owner := metav1.GetControllerOf(pod)
		if owner == nil {
			return nil
		}
		// Make sure owner is Bugua Controller.
		if owner.APIVersion != r.GetAPIGroupVersion().Version || owner.Kind != r.GetAPIGroupVersionKind().Kind {
			return nil
		}
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	r.Client = mgr.GetClient()
	r.Scheme = mgr.GetScheme()

	// Create k8s clients to list pods and service objects
	kubeClient := kubernetes.NewForConfigOrDie(mgr.GetConfig())
	recorder := mgr.GetEventRecorderFor(r.ControllerName())

	r.JobController = common.JobController{
		Controller:     r,
		Config:         common.JobControllerConfiguration{EnableGangScheduling: false},
		Expectations:   expectation.NewControllerExpectations(),
		WorkQueue:      workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), r.ControllerName()),
		Recorder:       recorder,
		KubeClientSet:  kubeClient,
		PodControl:     control.RealPodControl{KubeClient: kubeClient, Recorder: recorder},
		ServiceControl: control.RealServiceControl{KubeClient: kubeClient, Recorder: recorder},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&api.Bagua{}).
		Owns(&corev1.Pod{}, builder.WithPredicates(predicate.Funcs{
			CreateFunc: r.AddObject,
			DeleteFunc: r.DeleteObject,
		})).
		Owns(&corev1.Service{}, builder.WithPredicates(predicate.Funcs{
			CreateFunc: r.AddObject,
			DeleteFunc: r.DeleteObject,
		})).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 10,
		}).
		Complete(r)
}

func (r *BaguaReconciler) ControllerName() string {
	return "bugua-controller"
}

func (r *BaguaReconciler) GetAPIGroupVersionKind() schema.GroupVersionKind {
	return api.GroupVersionKind
}

func (r *BaguaReconciler) GetAPIGroupVersion() schema.GroupVersion {
	return api.GroupVersion
}

func (r *BaguaReconciler) GetGroupNameLabelValue() string {
	return api.GroupVersion.Group
}
