/*
Copyright 2023.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InstaScaleSpec defines the desired state of InstaScale
type InstaScaleSpec struct {

	// enableMonitoring determines if monitoring artifacts are deployed for the InstaScale instance.
	// +kubebuilder:default=true
	EnableMonitoring bool `json:"enableMonitoring,omitempty"`

	// maxScaleoutAllowed determines the max number of machines that can be scaled up by InstaScale
	// +kubebuilder:default=15
	MaxScaleoutAllowed int `json:"maxScaleoutAllowed,omitempty"`
}

// InstaScaleStatus defines the observed state of InstaScale
type InstaScaleStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// InstaScale is the Schema for the instascales API
// +operator-sdk:csv:customresourcedefinitions:displayName="InstaScale"
type InstaScale struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstaScaleSpec   `json:"spec,omitempty"`
	Status InstaScaleStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// InstaScaleList contains a list of InstaScale
type InstaScaleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InstaScale `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InstaScale{}, &InstaScaleList{})
}
