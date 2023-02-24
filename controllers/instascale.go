package controllers

import (
	"context"

	codeflarev1alpha1 "github.com/project-codeflare/codeflare-operator/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

var instascaleTemplates = []string{
	"instascale/configmap.yaml.tmpl",
	"instascale/sa.yaml.tmpl",
	"instascale/clusterrole.yaml.tmpl",
	"instascale/clusterrolebinding.yaml.tmpl",
	"instascale/deployment.yaml.tmpl",
}

func (r *InstaScaleReconciler) ReconcileInstaScale(ctx context.Context, instascale *codeflarev1alpha1.InstaScale, req ctrl.Request, params *InstaScaleParams) error {

	for _, template := range instascaleTemplates {
		err := r.Apply(instascale, params, template)
		if err != nil {
			return err
		}
	}

	r.Log.Info("Finished applying InstaScale Resources")
	return nil
}
