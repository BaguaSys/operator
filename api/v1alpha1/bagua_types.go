/*
Copyright 2021 Bagua-Operator Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
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
	RdzvEndpoint  string                                         `json:"rdzvEndpoint,omitempty"`

	// +kubebuilder:validation:Minimum=1
	MinReplicas *int32 `json:"minReplicas,omitempty"`
	MaxReplicas *int32 `json:"maxReplicas,omitempty"`
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
