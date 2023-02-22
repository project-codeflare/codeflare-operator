package controllers

import (
	mf "github.com/manifestival/manifestival"
	mcadv1alpha1 "github.com/project-codeflare/codeflare-operator/api/v1alpha1"
)

type MCADParams struct {
	Name               string
	Namespace          string
	Owner              mf.Owner
	EnableMonitoring   bool
	MultiCluster       bool
	DispatcherMode     bool
	PreemptionEnabled  bool
	AgentConfigs       string
	QuotaRestURL       string
	PodCreationTimeout int
}

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

	return nil
}
