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
