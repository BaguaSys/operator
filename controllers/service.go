package controllers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetServicesForJob returns the services managed by the job. This can be achieved by selecting services using label key "job-name"
// i.e. all services created by the job will come with label "job-name" = <this_job_name>
func (r *BaguaReconciler) GetServicesForJob(job interface{}) ([]*corev1.Service, error) {
	obj, err := meta.Accessor(job)
	if err != nil {
		return nil, fmt.Errorf("%+v is not a type of AresJob", job)
	}
	// List all pods to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.
	serviceList := &corev1.ServiceList{}
	err = r.List(context.Background(), serviceList, client.MatchingLabels(r.JobController.GenLabels(obj.GetName())))
	if err != nil {
		return nil, err
	}

	ret := convertServiceList(serviceList.Items)

	return ret, nil
}

// convertServiceList convert service list to service point list
func convertServiceList(list []corev1.Service) []*corev1.Service {
	if list == nil {
		return nil
	}
	ret := make([]*corev1.Service, 0, len(list))
	for i := range list {
		ret = append(ret, &list[i])
	}
	return ret
}
