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

// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
)

// MCADSpecApplyConfiguration represents an declarative configuration of the MCADSpec type for use
// with apply.
type MCADSpecApplyConfiguration struct {
	EnableMonitoring    *bool                    `json:"enableMonitoring,omitempty"`
	MultiCluster        *bool                    `json:"multiCluster,omitempty"`
	DispatcherMode      *bool                    `json:"dispatcherMode,omitempty"`
	PreemptionEnabled   *bool                    `json:"preemptionEnabled,omitempty"`
	AgentConfigs        *string                  `json:"agentConfigs,omitempty"`
	QuotaRestURL        *string                  `json:"quotaRestURL,omitempty"`
	PodCreationTimeout  *int                     `json:"podCreationTimeout,omitempty"`
	ControllerResources *v1.ResourceRequirements `json:"controllerResources,omitempty"`
}

// MCADSpecApplyConfiguration constructs an declarative configuration of the MCADSpec type for use with
// apply.
func MCADSpec() *MCADSpecApplyConfiguration {
	return &MCADSpecApplyConfiguration{}
}

// WithEnableMonitoring sets the EnableMonitoring field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the EnableMonitoring field is set to the value of the last call.
func (b *MCADSpecApplyConfiguration) WithEnableMonitoring(value bool) *MCADSpecApplyConfiguration {
	b.EnableMonitoring = &value
	return b
}

// WithMultiCluster sets the MultiCluster field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the MultiCluster field is set to the value of the last call.
func (b *MCADSpecApplyConfiguration) WithMultiCluster(value bool) *MCADSpecApplyConfiguration {
	b.MultiCluster = &value
	return b
}

// WithDispatcherMode sets the DispatcherMode field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the DispatcherMode field is set to the value of the last call.
func (b *MCADSpecApplyConfiguration) WithDispatcherMode(value bool) *MCADSpecApplyConfiguration {
	b.DispatcherMode = &value
	return b
}

// WithPreemptionEnabled sets the PreemptionEnabled field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the PreemptionEnabled field is set to the value of the last call.
func (b *MCADSpecApplyConfiguration) WithPreemptionEnabled(value bool) *MCADSpecApplyConfiguration {
	b.PreemptionEnabled = &value
	return b
}

// WithAgentConfigs sets the AgentConfigs field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the AgentConfigs field is set to the value of the last call.
func (b *MCADSpecApplyConfiguration) WithAgentConfigs(value string) *MCADSpecApplyConfiguration {
	b.AgentConfigs = &value
	return b
}

// WithQuotaRestURL sets the QuotaRestURL field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the QuotaRestURL field is set to the value of the last call.
func (b *MCADSpecApplyConfiguration) WithQuotaRestURL(value string) *MCADSpecApplyConfiguration {
	b.QuotaRestURL = &value
	return b
}

// WithPodCreationTimeout sets the PodCreationTimeout field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the PodCreationTimeout field is set to the value of the last call.
func (b *MCADSpecApplyConfiguration) WithPodCreationTimeout(value int) *MCADSpecApplyConfiguration {
	b.PodCreationTimeout = &value
	return b
}

// WithControllerResources sets the ControllerResources field in the declarative configuration to the given value
// and returns the receiver, so that objects can be built by chaining "With" function invocations.
// If called multiple times, the ControllerResources field is set to the value of the last call.
func (b *MCADSpecApplyConfiguration) WithControllerResources(value v1.ResourceRequirements) *MCADSpecApplyConfiguration {
	b.ControllerResources = &value
	return b
}
