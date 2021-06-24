package controllers

import (
	api "github.com/BaguaSys/operator/api/v1alpha1"
	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
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
