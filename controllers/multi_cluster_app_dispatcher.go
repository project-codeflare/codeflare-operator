package controllers

import (
	"context"
	codeflarev1alpha1 "github.com/project-codeflare/codeflare-operator/api/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

var multiClusterAppDispatcherTemplates = []string{
	"mcad/configmap.yaml.tmpl",
	"mcad/service.yaml.tmpl",
	"mcad/serviceaccount.yaml.tmpl",
}
var ownerLessmultiClusterAppDispatcherTemplates = []string{
	"mcad/rolebinding_custom-metrics-auth-reader.yaml.tmpl",
}

func (r *MCADReconciler) ReconcileMCAD(ctx context.Context, mcad *codeflarev1alpha1.MCAD, req ctrl.Request, params *MCADParams) error {

	for _, template := range multiClusterAppDispatcherTemplates {
		r.Log.Info("Applying " + template)
		err := r.Apply(mcad, params, template)
		if err != nil {
			return err
		}
	}

	for _, template := range ownerLessmultiClusterAppDispatcherTemplates {
		r.Log.Info("Applying " + template)
		err := r.ApplyWithoutOwner(params, template)
		if err != nil {
			return err
		}
	}

	r.Log.Info("Finished applying MultiClusterAppDispatcher Resources")
	return nil
}

func (r *MCADReconciler) deleteOwnerLessObjects(params *MCADParams) error {
	for _, template := range ownerLessmultiClusterAppDispatcherTemplates {
		r.Log.Info("Deleting Ownerless object: " + template)
		err := r.DeleteResource(params, template)
		if err != nil {
			return err
		}
	}
	return nil
}
