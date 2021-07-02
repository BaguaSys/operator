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
	key := obj.GetNamespace() + "/" + owner.Name
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
	key := obj.GetNamespace() + "/" + owner.Name
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
