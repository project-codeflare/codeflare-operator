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
	v1 "k8s.io/api/core/v1"
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

	// useMachinePools determines whether InstaScale should use MachineSets or MachinePools for scaling
	// +kubebuilder:default=false
	UseMachinePools bool `json:"useMachinePools,omitempty"`

	// controllerResources determines the container resources for the InstaScale controller deployment
	ControllerResources *v1.ResourceRequirements `json:"controllerResources,omitempty"`

	// The container image for the InstaScale controller deployment.
	// If specified, the provided container image must be compatible with the running CodeFlare operator.
	// Using an incompatible, or unrelated container image, will result in an undefined behavior.
	// A CodeFlare operator upgrade will not upgrade the InstaScale controller, that'll keep running this
	// specified container image.
	// If not specified, the latest version compatible with the running CodeFlare operator is used.
	// A CodeFlare operator upgrade may upgrade the InstaScale controller to a newer container image.
	//
	// +optional
	ControllerImage string `json:"controllerImage,omitempty"`
}

// InstaScaleStatus defines the observed state of InstaScale
type InstaScaleStatus struct {
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:default=false
	Ready bool `json:"ready"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// InstaScale is the Schema for the instascales API
// +operator-sdk:csv:customresourcedefinitions:displayName="InstaScale"
type InstaScale struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstaScaleSpec   `json:"spec,omitempty"`
	Status InstaScaleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// InstaScaleList contains a list of InstaScale
type InstaScaleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InstaScale `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InstaScale{}, &InstaScaleList{})
}
