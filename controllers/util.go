package controllers

import (
	"fmt"

	api "github.com/BaguaSys/operator/api/v1alpha1"
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	"github.com/kubeflow/common/pkg/controller.v1/common"
)

// IsInvalidDefinition
func IsInvalidDefinition(status api.BaguaStatus) bool {
	for _, condition := range status.Conditions {
		if condition.Type != commonv1.JobFailed {
			continue
		}
		return condition.Reason == api.ErrInvalidDefinition
	}
	return false
}

// IsFinished
func IsFinished(phase commonv1.JobConditionType) bool {
	return phase == commonv1.JobSucceeded || phase == commonv1.JobFailed
}

// GetPodDomainName
func GetPodDomainName(jobName, namespace string, rtype, index string) string {
	n := common.GenGeneralName(jobName, rtype, index)
	return fmt.Sprintf("%v.%v.%v.svc.cluster.local", n, n, namespace)
}
