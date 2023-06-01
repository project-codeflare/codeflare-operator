package controllers

import (
	"encoding/json"

	"github.com/manifestival/manifestival"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	instascalev1alpha1 "github.com/project-codeflare/codeflare-operator/api/v1alpha1"
)

type InstaScaleParams struct {
	Name                string
	Namespace           string
	Owner               manifestival.Owner
	EnableMonitoring    bool
	MaxScaleoutAllowed  int
	UseMachinePools     bool
	ControllerResources ControllerResources
	ControllerImage     string
}

type ControllerResources struct {
	v1.ResourceRequirements
}

func (c *ControllerResources) String() string {
	raw, err := json.Marshal(c)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func (p *InstaScaleParams) ExtractParams(instascale *instascalev1alpha1.InstaScale) {
	p.Name = instascale.Name
	p.Namespace = instascale.Namespace
	p.ControllerImage = instascale.Spec.ControllerImage
	if p.ControllerImage == "" {
		p.ControllerImage = InstaScaleImage
	}
	p.Owner = instascale
	p.EnableMonitoring = instascale.Spec.EnableMonitoring
	p.MaxScaleoutAllowed = instascale.Spec.MaxScaleoutAllowed
	p.UseMachinePools = instascale.Spec.UseMachinePools
	if instascale.Spec.ControllerResources == nil {
		p.ControllerResources = ControllerResources{
			v1.ResourceRequirements{
				Limits: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("2"),
					v1.ResourceMemory: resource.MustParse("2G")},
				Requests: v1.ResourceList{
					v1.ResourceCPU:    resource.MustParse("200m"),
					v1.ResourceMemory: resource.MustParse("200M")},
			}}
	} else {
		p.ControllerResources = ControllerResources{*instascale.Spec.ControllerResources}
	}
}
