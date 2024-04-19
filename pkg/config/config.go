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

package config

import (
	configv1alpha1 "k8s.io/component-base/config/v1alpha1"
)

type CodeFlareOperatorConfiguration struct {
	// ClientConnection provides additional configuration options for Kubernetes
	// API server client.
	ClientConnection *ClientConnection `json:"clientConnection,omitempty"`

	// ControllerManager returns the configurations for controllers
	ControllerManager `json:",inline"`

	KubeRay *KubeRayConfiguration `json:"kuberay,omitempty"`
}

type KubeRayConfiguration struct {
	RayDashboardOAuthEnabled *bool `json:"rayDashboardOAuthEnabled,omitempty"`

	IngressDomain string `json:"ingressDomain"`

	MTLSEnabled *bool `json:"mTLSEnabled,omitempty"`
}

type ControllerManager struct {
	// Metrics contains the controller metrics configuration
	// +optional
	Metrics MetricsConfiguration `json:"metrics,omitempty"`

	// Health contains the controller health configuration
	// +optional
	Health HealthConfiguration `json:"health,omitempty"`

	// LeaderElection is the LeaderElection config to be used when configuring
	// the manager.Manager leader election
	LeaderElection *configv1alpha1.LeaderElectionConfiguration `json:"leaderElection,omitempty"`
}

type ClientConnection struct {
	// QPS controls the number of queries per second allowed before client-side throttling
	// connection to the API server.
	QPS *float32 `json:"qps,omitempty"`

	// Burst allows extra queries to accumulate when a client is exceeding its rate.
	Burst *int32 `json:"burst,omitempty"`
}

// MetricsConfiguration defines the metrics configuration.
type MetricsConfiguration struct {
	// BindAddress is the TCP address that the controller should bind to
	// for serving Prometheus metrics.
	// It can be set to "0" to disable the metrics serving.
	// +optional
	BindAddress string `json:"bindAddress,omitempty"`
}

// HealthConfiguration defines the health configuration.
type HealthConfiguration struct {
	// BindAddress is the TCP address that the controller should bind to
	// for serving health probes.
	// It can be set to "0" or "" to disable serving the health probe.
	// +optional
	BindAddress string `json:"bindAddress,omitempty"`

	// ReadinessEndpointName, defaults to "readyz"
	// +optional
	ReadinessEndpointName string `json:"readinessEndpointName,omitempty"`

	// LivenessEndpointName, defaults to "healthz"
	// +optional
	LivenessEndpointName string `json:"livenessEndpointName,omitempty"`
}
