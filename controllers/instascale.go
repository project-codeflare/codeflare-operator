package controllers

import (
	codeflarev1alpha1 "github.com/project-codeflare/codeflare-operator/api/v1alpha1"
)

var instascaleTemplates = []string{
	"instascale/configmap.yaml.tmpl",
	"instascale/sa.yaml.tmpl",
	"instascale/clusterrole.yaml.tmpl",
	"instascale/clusterrolebinding.yaml.tmpl",
	"instascale/deployment.yaml.tmpl",
}

func (r *InstaScaleReconciler) ReconcileInstaScale(instascale *codeflarev1alpha1.InstaScale, params *InstaScaleParams) error {

	for _, template := range instascaleTemplates {
		err := r.Apply(instascale, params, template)
		if err != nil {
			return err
		}
	}

	r.Log.Info("Finished applying InstaScale Resources")
	return nil
}
