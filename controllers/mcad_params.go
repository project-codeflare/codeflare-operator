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

package controllers

import (
	mf "github.com/manifestival/manifestival"
	mcadv1alpha1 "github.com/project-codeflare/codeflare-operator/api/codeflare/v1alpha1"
)

type MCADParams struct {
	Name                string
	Namespace           string
	Owner               mf.Owner
	EnableMonitoring    bool
	MultiCluster        bool
	DispatcherMode      bool
	PreemptionEnabled   bool
	AgentConfigs        string
	QuotaRestURL        string
	PodCreationTimeout  int
	ControllerResources ControllerResources
}

// type ControllerResources struct {
// 	v1.ResourceRequirements
// }

// ExtractParams is currently a straight-up copy. We can add in more complex validation at a later date
func (p *MCADParams) ExtractParams(mcad *mcadv1alpha1.MCAD) error {
	p.Name = mcad.Name
	p.Namespace = mcad.Namespace
	p.Owner = mcad
	p.EnableMonitoring = mcad.Spec.EnableMonitoring
	p.MultiCluster = mcad.Spec.MultiCluster
	p.DispatcherMode = mcad.Spec.DispatcherMode
	p.PreemptionEnabled = mcad.Spec.PreemptionEnabled
	p.AgentConfigs = mcad.Spec.AgentConfigs
	p.QuotaRestURL = mcad.Spec.QuotaRestURL
	p.PodCreationTimeout = mcad.Spec.PodCreationTimeout
	p.ControllerResources = ControllerResources{mcad.Spec.ControllerResources}

	return nil
}
