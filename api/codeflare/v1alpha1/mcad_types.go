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

// MCADSpec defines the desired state of MCAD
type MCADSpec struct {
	// EnableMonitoring determines if monitoring artifacts are deployed for the MCAD instance.
	// +kubebuilder:default=true
	EnableMonitoring bool `json:"enableMonitoring,omitempty"`

	// MultiCluster determines if MCAD will be routing traffic to multiple clusters.
	// +kubebuilder:default=false
	MultiCluster bool `json:"multiCluster,omitempty"`

	// DispatcherMode determines whether the MCAD Controller should be launched in Dispatcher mode.
	// +kubebuilder:default=false
	DispatcherMode bool `json:"dispatcherMode,omitempty"`

	// PreemptionEnabled determines if scheduled jobs can be preempted for others
	// +kubebuilder:default=false
	PreemptionEnabled bool `json:"preemptionEnabled,omitempty"`

	// AgentConfigs determine paths to agent config file:deploymentName separated by commas(,).
	AgentConfigs string `json:"agentConfigs,omitempty"`

	// QuotaRestURL determines URL for Rest quota management.
	QuotaRestURL string `json:"quotaRestURL,omitempty"`

	// PodCreationTimeout determines timeout in milliseconds for pods to be created after dispatching job.
	// +kubebuilder:default=-1
	PodCreationTimeout int `json:"podCreationTimeout,omitempty"`
	// podCreationTimeout: //int (default blank)

	// ControllerResources defines the cpu and memory resource requirements for the MCAD Controller
	// +kubebuilder:default={}
	ControllerResources v1.ResourceRequirements `json:"controllerResources,omitempty" protobuf:"bytes,8,opt"`
}

// MCADStatus defines the observed state of MCAD
type MCADStatus struct {
	// Important: Run "make" to regenerate code after modifying this file

	// Ready indicates whether the application is ready to serve requests
	Ready bool `json:"ready"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// MCAD is the Schema for the mcads API
type MCAD struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MCADSpec   `json:"spec,omitempty"`
	Status MCADStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MCADList contains a list of MCAD
type MCADList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MCAD `json:"items"`
}
