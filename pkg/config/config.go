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
	instascale "github.com/project-codeflare/instascale/pkg/config"
	mcad "github.com/project-codeflare/multi-cluster-app-dispatcher/pkg/config"

	configv1alpha1 "k8s.io/component-base/config/v1alpha1"
)

type CodeFlareOperatorConfiguration struct {
	// LeaderElection is the LeaderElection config to be used when configuring
	// the manager.Manager leader election
	LeaderElection *configv1alpha1.LeaderElectionConfiguration `json:"leaderElection,omitempty"`

	// The MCAD controller configuration
	MCAD *mcad.MCADConfiguration `json:"mcad,omitempty"`

	// The InstaScale controller configuration
	InstaScale *InstaScaleConfiguration `json:"instascale,omitempty"`
}

type InstaScaleConfiguration struct {
	// enabled controls whether the InstaScale controller is started.
	// It may default to true on platforms that InstaScale supports.
	// Otherwise, it defaults to false.
	Enabled *bool `json:"enabled,omitempty"`

	// The InstaScale controller configuration
	instascale.InstaScaleConfiguration `json:",inline,omitempty"`
}
