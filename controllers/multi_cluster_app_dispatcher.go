package controllers

import (
	codeflarev1alpha1 "github.com/project-codeflare/codeflare-operator/api/v1alpha1"
)

var multiClusterAppDispatcherTemplates = []string{
	"mcad/configmap.yaml.tmpl",
	"mcad/service.yaml.tmpl",
	"mcad/serviceaccount.yaml.tmpl",
	"mcad/deployment.yaml.tmpl",
}
var ownerLessmultiClusterAppDispatcherTemplates = []string{
	"mcad/rolebinding_custom-metrics-auth-reader.yaml.tmpl",
	"mcad/clusterrole_custom-metrics-server-admin.yaml.tmpl",
	"mcad/clusterrole_mcad-controller.yaml.tmpl",
	"mcad/clusterrole_metrics-resource-reader.yaml.tmpl",
	"mcad/clusterrolebinding_hpa-controller-custom-metrics.yaml.tmpl",
	"mcad/clusterrolebinding_mcad-controller.yaml.tmpl",
	"mcad/clusterrolebinding_mcad-controller-kube-scheduler.yaml.tmpl",
	"mcad/clusterrolebinding_mcad-edit.yaml.tmpl",
	"mcad/clusterrolebinding_mcad-system-auth-delegator.yaml.tmpl",
	"mcad/clusterrolebinding_metrics-resource-reader.yaml.tmpl",
}

func (r *MCADReconciler) ReconcileMCAD(mcad *codeflarev1alpha1.MCAD, params *MCADParams) error {

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
