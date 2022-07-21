/*
Copyright 2022.

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
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SampleKindSpec defines the desired state of SampleKind
type SampleKindSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Sie of the deployment(Required)
	//+kubebuilder:validation:Minimum=0
	Size int32 `json:"size"`

	// Optional label for the pod(Optional)
	Label string `json:"label,omitempty"`
}

// SampleKindStatus defines the observed state of SampleKind
type SampleKindStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Name of the pods
	Status     string                   `json:"status,omitempty"`
	LastUpdate metav1.Time              `json:"lastUpdate,omitempty"`
	Reason     string                   `json:"reason,omitempty"`
	Pods       map[string]core.PodPhase `json:"pods,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// SampleKind is the Schema for the samplekinds API
type SampleKind struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SampleKindSpec   `json:"spec,omitempty"`
	Status SampleKindStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SampleKindList contains a list of SampleKind
type SampleKindList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SampleKind `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SampleKind{}, &SampleKindList{})
}
