package controllers

import (
	"context"
	"fmt"
	"reflect"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	api "github.com/BaguaSys/operator/api/v1alpha1"
)

const (
	FailedDeleteJobReason     = "FailedDeleteJob"
	SuccessfulDeleteJobReason = "SuccessfulDeleteJob"
	JobPhaseTransitionReason  = "JobPhaseTransition"
)

func (r *BaguaReconciler) GetJobFromInformerCache(namespace, name string) (metav1.Object, error) {
	namespacedName := types.NamespacedName{Namespace: namespace, Name: name}
	log := r.Log.WithValues("bagua", namespacedName)

	job := &api.Bagua{}
	err := r.Get(context.Background(), namespacedName, job)
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "failed to get Bugua from informer cache", "namespace", namespace, "name", name)
		}
		return nil, err
	}

	return job, nil
}

// GetJobFromAPIClient
func (r *BaguaReconciler) GetJobFromAPIClient(namespace, name string) (metav1.Object, error) {
	return r.GetJobFromInformerCache(namespace, name)
}

func (r *BaguaReconciler) DeleteJob(job interface{}) error {
	bagua, ok := job.(*api.Bagua)
	if !ok {
		return fmt.Errorf("%+v is not a type of Bugua", job)
	}

	log := util.LoggerForJob(bagua)
	if err := r.Delete(context.Background(), bagua); err != nil {
		r.JobController.Recorder.Eventf(bagua, corev1.EventTypeWarning, FailedDeleteJobReason, "Error deleting: %v", err)
		log.Errorf("failed to delete job %s/%s, %v", bagua.Namespace, bagua.Name, err)
		return err
	}

	r.JobController.Recorder.Eventf(bagua, corev1.EventTypeNormal, SuccessfulDeleteJobReason, "Deleted job: %v", bagua.Name)
	log.Infof("job %s/%s has been deleted", bagua.Namespace, bagua.Name)
	return nil
}

// UpdateJobStatus: update job status
func (r *BaguaReconciler) UpdateJobStatus(j interface{}, replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec, jobStatus *commonv1.JobStatus) error {
	job, ok := j.(*api.Bagua)
	if !ok {
		return fmt.Errorf("%+v is not a type of Bugua", j)
	}
	log := util.LoggerForJob(job)
	phase := getJobPhase(job, replicas, jobStatus)
	if phase != job.Status.Phase {
		msg := fmt.Sprintf("Phase of Bagua %s has changed: %s -> %s", job.Name, job.Status.Phase, phase)
		r.Recorder.Event(job, corev1.EventTypeNormal, JobPhaseTransitionReason, msg)
		log.Info(msg)
		now := metav1.Now()
		if IsFinished(phase) && jobStatus.CompletionTime == nil {
			jobStatus.CompletionTime = &now
		}
		msg = fmt.Sprintf("Bagua %s is %s.", job.Name, phase)
		util.UpdateJobConditions(jobStatus, phase, string(phase), msg)
		job.Status.Phase = phase
	}
	return nil
}

func getJobPhase(job *api.Bagua, replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec, jobStatus *commonv1.JobStatus) commonv1.JobConditionType {
	succeeded := true
	for rtype, status := range jobStatus.ReplicaStatuses {
		spec := replicas[rtype]
		if status.Failed > 0 {
			return commonv1.JobFailed
		}
		if status.Succeeded < *spec.Replicas && rtype != api.ReplicaEtcd {
			succeeded = false
		}
	}
	if succeeded {
		return commonv1.JobSucceeded
	}
	return commonv1.JobRunning
}

func (r *BaguaReconciler) UpdateJobStatusInApiServer(job interface{}, jobStatus *commonv1.JobStatus) error {
	bagua, ok := job.(*api.Bagua)
	if !ok {
		return fmt.Errorf("%+v is not a type of Bugua", job)
	}

	// Job status passed in differs with status in job, update in basis of the passed in one.
	if !reflect.DeepEqual(&bagua.Status.JobStatus, jobStatus) {
		bagua = bagua.DeepCopy()
		bagua.Status.JobStatus = *jobStatus
	}

	err := r.Status().Update(context.Background(), bagua)
	if err != nil {
		util.LoggerForJob(bagua).Error(err, "failed to update Bugua conditions in the API server")
		return err
	}
	return nil
}
