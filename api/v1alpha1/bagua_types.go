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

package v1alpha1

import (
	"encoding/json"
	"fmt"

	commonv1 "github.com/kubeflow/common/pkg/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

/*********
 * Spec
 *********/
// BaguaSpec defines the desired state of Bagua
type BaguaSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	RunPolicy     commonv1.RunPolicy                             `json:"runPolicy,omitempty"`
	ReplicaSpecs  map[commonv1.ReplicaType]*commonv1.ReplicaSpec `json:"replicaSpecs,omitempty"`
	EnableElastic bool                                           `json:"enableElastic,omitempty"`

	// +kubebuilder:validation:Minimum=1
	MinReplicas *int32 `json:"minReplicas,omitempty"`
	MaxReplicas *int32 `json:"maxReplicas,omitempty"`
}

func (spec BaguaSpec) CheckReplicasExist(replicaTypes []commonv1.ReplicaType) error {
	for _, rt := range replicaTypes {
		_, ok := spec.ReplicaSpecs[rt]
		if !ok {
			return fmt.Errorf("roles(%v) not exist", rt)
		}
	}

	return nil
}

func (spec BaguaSpec) GetReplicas(rt commonv1.ReplicaType) int32 {
	if spec.ReplicaSpecs[rt] == nil || spec.ReplicaSpecs[rt].Replicas == nil {
		return 0
	}
	return *spec.ReplicaSpecs[rt].Replicas
}

/*********
 * Status
 *********/
// BaguaStatus defines the observed state of Bagua
type BaguaStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	commonv1.JobStatus `json:",inline"`
	Phase              commonv1.JobConditionType `json:"phase,omitempty"`
}

/*********
 * Bagua
 *********/
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=bg
//+kubebuilder:printcolumn:name="Namespace",type=string,JSONPath=`.metadata.namespace`
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Bagua is the Schema for the baguas API
type Bagua struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BaguaSpec   `json:"spec,omitempty"`
	Status BaguaStatus `json:"status,omitempty"`
}

func (job Bagua) Json() string {
	if content, err := json.Marshal(job); err == nil {
		return string(content)
	}
	return fmt.Sprintf("Bagua<%s.%s>", job.Namespace, job.Name)
}

//+kubebuilder:object:root=true

// BaguaList contains a list of Bagua
type BaguaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Bagua `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Bagua{}, &BaguaList{})
}
