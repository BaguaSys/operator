package controllers

import (
	"context"
	"fmt"
	"reflect"

	api "github.com/BaguaSys/operator/api/v1alpha1"
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/util"
	"k8s.io/client-go/kubernetes/scheme"
)

func (r *BaguaReconciler) setDefault(old, job *api.Bagua) (bool, error) {
	scheme.Scheme.Default(job)
	// set default properties
	if job.Status.ReplicaStatuses == nil {
		job.Status.ReplicaStatuses = map[commonv1.ReplicaType]*commonv1.ReplicaStatus{}
	}
	for rtype, spec := range job.Spec.ReplicaSpecs {
		one := int32(1)
		if spec != nil && spec.Replicas == nil {
			spec.Replicas = &one
		}
		if job.Status.ReplicaStatuses[rtype] == nil {
			job.Status.ReplicaStatuses[rtype] = &commonv1.ReplicaStatus{}
		}
	}

	if job.Spec.RunPolicy.CleanPodPolicy == nil {
		none := commonv1.CleanPodPolicyNone
		job.Spec.RunPolicy.CleanPodPolicy = &none
	}

	if len(job.Status.Phase) == 0 {
		job.Status.Phase = commonv1.JobCreated
	}

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

func (r *BaguaReconciler) validate(job *api.Bagua) (err error) {
	defer func() {
		if err != nil {
			r.setInvalidDefinition(job, fmt.Sprintf("validation failed: %v", err))
		}
	}()
	// validate specs not nil
	for rt, spec := range job.Spec.ReplicaSpecs {
		if spec == nil {
			return fmt.Errorf("replica %v spec is nil", rt)
		}
	}
	// validate container name
	for rt, spec := range job.Spec.ReplicaSpecs {
		if rt != api.ReplicaEtcd && rt != api.ReplicaMaster && rt != api.ReplicaWorker {
			continue
		}
		containers := spec.Template.Spec.Containers
		if len(containers) == 0 {
			return fmt.Errorf("must define a container for %v at least", rt)
		}
		hasDefaultContainer := false
		for _, c := range containers {
			if c.Name == r.GetDefaultContainerName() {
				hasDefaultContainer = true
			}
		}
		if !hasDefaultContainer {
			return fmt.Errorf("must have a container named %v for %v at least", r.GetDefaultContainerName(), rt)
		}
	}

	if !job.Spec.EnableElastic {
		if err = job.Spec.CheckReplicasExist([]commonv1.ReplicaType{api.ReplicaMaster}); err != nil {
			return err
		}
		// validate ports
		ports, err := r.GetPortsFromJob(job.Spec.ReplicaSpecs[api.ReplicaMaster])
		if err != nil {
			return err
		}
		if len(ports) < 1 {
			return fmt.Errorf("must assign a container port for master at least")
		}
		// validate replicas
		for rt, spec := range job.Spec.ReplicaSpecs {
			if *spec.Replicas < 1 {
				return fmt.Errorf("invalid replicas %v for %v", *spec.Replicas, rt)
			}
			if rt == api.ReplicaMaster && *spec.Replicas != 1 {
				return fmt.Errorf("master replicas must be 1")
			}
		}
	} else {
		if err = job.Spec.CheckReplicasExist([]commonv1.ReplicaType{api.ReplicaEtcd, api.ReplicaWorker}); err != nil {
			return err
		}
		// validate ports
		ports, err := r.GetPortsFromJob(job.Spec.ReplicaSpecs[api.ReplicaEtcd])
		if err != nil {
			return err
		}
		if len(ports) < 1 {
			return fmt.Errorf("must assign 1 container port for etcd at least")
		}
		// validate replicas
		for rt, spec := range job.Spec.ReplicaSpecs {
			if *spec.Replicas < 1 {
				return fmt.Errorf("invalid replicas %v for %v", *spec.Replicas, rt)
			}
		}
		// validate bagua spec
		workerReplicas := job.Spec.GetReplicas(api.ReplicaWorker)
		if job.Spec.MinReplicas != nil && *job.Spec.MinReplicas > workerReplicas {
			return fmt.Errorf("worker replicas should in the interval [minReplicas, maxReplicas]")
		}
		if job.Spec.MaxReplicas != nil && *job.Spec.MaxReplicas < workerReplicas {
			return fmt.Errorf("worker replicas should in the interval [minReplicas, maxReplicas]")
		}
	}

	return nil
}

func (r *BaguaReconciler) setInvalidDefinition(job *api.Bagua, msg string) error {
	util.UpdateJobConditions(&job.Status.JobStatus, commonv1.JobFailed, api.ErrInvalidDefinition, msg)
	job.Status.Phase = commonv1.JobFailed
	return r.Status().Update(context.Background(), job)
}
