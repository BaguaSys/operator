package controllers

import (
	"context"
	api "github.com/BaguaSys/operator/api/v1alpha1"
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetPodsForJob returns the pods managed by the job. This can be achieved by selecting pods using label key "job-name"
// i.e. all pods created by the job will come with label "job-name" = <this_job_name>
func (r *BaguaReconciler) GetPodsForJob(job interface{}) ([]*corev1.Pod, error) {
	obj, err := meta.Accessor(job)
	if err != nil {
		return nil, err
	}
	// List all pods to include those that don't match the selector anymore
	// but have a ControllerRef pointing to this controller.
	podlist := &corev1.PodList{}
	err = r.List(context.Background(), podlist, client.MatchingLabels(r.JobController.GenLabels(obj.GetName())))
	if err != nil {
		return nil, err
	}

	return convertPodList(podlist.Items), nil
}

// convertPodList convert pod list to pod point list
func convertPodList(list []corev1.Pod) []*corev1.Pod {
	if list == nil {
		return nil
	}
	ret := make([]*corev1.Pod, 0, len(list))
	for i := range list {
		ret = append(ret, &list[i])
	}
	return ret
}

func (r *BaguaReconciler) GetDefaultContainerName() string {
	return api.DefaultContainerName
}

func (r *BaguaReconciler) GetDefaultContainerPortName() string {
	return ""
}

func (r *BaguaReconciler) IsMasterRole(replicas map[commonv1.ReplicaType]*commonv1.ReplicaSpec,
	rtype commonv1.ReplicaType, index int) bool {
	return false
}

// SetClusterSpec sets the cluster spec for the pod
func (r *BaguaReconciler) SetClusterSpec(job interface{}, podTemplate *corev1.PodTemplateSpec, rtype, index string) error {
	return nil
}
