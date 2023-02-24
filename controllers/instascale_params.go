package controllers

import (
	mf "github.com/manifestival/manifestival"
	instascalev1alpha1 "github.com/project-codeflare/codeflare-operator/api/v1alpha1"
)

type InstaScaleParams struct {
	Name               string
	Namespace          string
	Owner              mf.Owner
	EnableMonitoring   bool
	MaxScaleoutAllowed int
}

func (p *InstaScaleParams) ExtractParams(instascale *instascalev1alpha1.InstaScale) error {
	p.Name = instascale.Name
	p.Namespace = instascale.Namespace
	p.Owner = instascale
	p.EnableMonitoring = instascale.Spec.EnableMonitoring
	p.MaxScaleoutAllowed = instascale.Spec.MaxScaleoutAllowed

	return nil
}
