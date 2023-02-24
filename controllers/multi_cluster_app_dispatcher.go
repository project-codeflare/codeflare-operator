package controllers

import (
	"context"
	codeflarev1alpha1 "github.com/project-codeflare/codeflare-operator/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

var multiClusterAppDispatcherTemplates = []string{
	"mcad/configmap.yaml.tmpl",
}

func (r *MCADReconciler) ReconcileMCAD(ctx context.Context, mcad *codeflarev1alpha1.MCAD, req ctrl.Request, params *MCADParams) error {

	for _, template := range multiClusterAppDispatcherTemplates {
		err := r.Apply(mcad, params, template)
		if err != nil {
			return err
		}
	}

	r.Log.Info("Finished applying MultiClusterAppDispatcher Resources")
	return nil
}
