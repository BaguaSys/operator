package controllers

import (
	"fmt"

	v1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/controller.v1/common"
	"github.com/kubeflow/common/pkg/controller.v1/expectation"
	"github.com/kubeflow/common/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"

	api "github.com/BaguaSys/operator/api/v1alpha1"
)

func (r *BaguaReconciler) ExpectationsFuncApply(job *api.Bagua, f func(key string) error) error {
	key, err := common.KeyFunc(job)
	if err != nil {
		runtime.HandleError(fmt.Errorf("couldn't get key for job object %v: %v", job, err))
		return err
	}
	for rtype := range job.Spec.ReplicaSpecs {
		expectationPodsKey := expectation.GenExpectationPodsKey(key, string(rtype))
		if err := f(expectationPodsKey); err != nil {
			return err
		}

		expectationServicesKey := expectation.GenExpectationServicesKey(key, string(rtype))
		if err := f(expectationServicesKey); err != nil {
			return err
		}
	}
	return nil
}

func (r *BaguaReconciler) ResetExpectations(job *api.Bagua) error {
	return r.ExpectationsFuncApply(job, func(key string) error {
		// Clear the expectations
		return r.JobController.Expectations.SetExpectations(key, 0, 0)
	})
}

func (r *BaguaReconciler) SatisfiedExpectations(job *api.Bagua) bool {
	err := r.ExpectationsFuncApply(job, func(key string) error {
		// Check the expectations
		if satisfied := r.JobController.Expectations.SatisfiedExpectations(key); !satisfied {
			return fmt.Errorf("expectation is not satisfied: %s", key)
		}
		return nil
	})
	return err == nil
}

func (r *BaguaReconciler) DeleteExpectations(job *api.Bagua) {
	r.ExpectationsFuncApply(job, func(key string) error {
		// Delete the expectations
		r.JobController.Expectations.DeleteExpectations(key)
		return nil
	})
}

func (r *BaguaReconciler) AddObject(e event.CreateEvent) bool {
	obj := e.Object
	if value := obj.GetLabels()[v1.GroupNameLabel]; value != r.GetAPIGroupVersion().Group {
		return false
	}
	rtype := obj.GetLabels()[v1.ReplicaTypeLabel]
	if len(rtype) == 0 {
		return false
	}
	owner := metav1.GetControllerOf(obj)
	if owner == nil {
		return false
	}
	key := obj.GetNamespace()+"/"+owner.Name
	var expectKey string
	if _, ok := obj.(*corev1.Pod); ok {
		expectKey = expectation.GenExpectationPodsKey(key, rtype)
	}
	if _, ok := obj.(*corev1.Service); ok {
		expectKey = expectation.GenExpectationServicesKey(key, rtype)
	}
	util.LoggerForKey(key).Infof("creation observed: name=%s, key=%s", obj.GetName(), expectKey)
	r.JobController.Expectations.CreationObserved(expectKey)
	return true
}

func (r *BaguaReconciler) DeleteObject(e event.DeleteEvent) bool {
	obj := e.Object
	if value := obj.GetLabels()[v1.GroupNameLabel]; value != r.GetAPIGroupVersion().Group {
		return false
	}
	rtype := obj.GetLabels()[v1.ReplicaTypeLabel]
	if len(rtype) == 0 {
		return false
	}
	owner := metav1.GetControllerOf(obj)
	if owner == nil {
		return false
	}
	key := obj.GetNamespace()+"/"+owner.Name
	var expectKey string
	if _, ok := obj.(*corev1.Pod); ok {
		expectKey = expectation.GenExpectationPodsKey(key, rtype)
	}
	if _, ok := obj.(*corev1.Service); ok {
		expectKey = expectation.GenExpectationServicesKey(key, rtype)
	}
	util.LoggerForKey(key).Infof("deletion observed: name=%s, key=%s", obj.GetName(), expectKey)
	r.JobController.Expectations.DeletionObserved(expectKey)
	return true
}
