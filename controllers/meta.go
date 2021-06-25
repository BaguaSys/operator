package controllers

import (
	"context"
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"reflect"

	api "github.com/BaguaSys/operator/api/v1alpha1"
	"github.com/kubeflow/common/pkg/util"
	"k8s.io/client-go/kubernetes/scheme"
)

func (r *BaguaReconciler) setDefault(old, job *api.Bagua) (bool, error) {
	scheme.Scheme.Default(job)
	// set default properties
	if job.Status.ReplicaStatuses == nil {
		job.Status.ReplicaStatuses = map[commonv1.ReplicaType]*commonv1.ReplicaStatus{}
	}
	for rtype := range job.Spec.ReplicaSpecs {
		if job.Status.ReplicaStatuses[rtype] == nil {
			job.Status.ReplicaStatuses[rtype] = &commonv1.ReplicaStatus{}
		}
	}
	if len(job.Status.Phase) == 0 {
		job.Status.Phase = commonv1.JobCreated
	}
	// TODO

	// update
	if reflect.DeepEqual(old.Spec, job.Spec) {
		return false, nil
	}
	log := util.LoggerForJob(job)
	if err := r.Update(context.Background(), job); err != nil {
		log.Errorf("failed to update spec in API server: %v", err)
		return false, err
	}
	log.Infof("succeeded to update default properties in API server: %s", job.Json())
	return true, nil
}

func (r *BaguaReconciler) validate(job *api.Bagua) error {
	/* TODO
	if err != nil {
		r.setInvalidDefinition(job, fmt.Sprintf("validation failed: %v", err))
		return err
	}
	*/
	return nil
}

func (r *BaguaReconciler) setInvalidDefinition(job *api.Bagua, msg string) error {
	util.UpdateJobConditions(&job.Status.JobStatus, commonv1.JobFailed, api.ErrInvalidDefinition, msg)
	job.Status.Phase = commonv1.JobFailed
	return r.Status().Update(context.Background(), job)
}
