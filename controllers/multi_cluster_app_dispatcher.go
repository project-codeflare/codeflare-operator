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
	codeflarev1alpha1 "github.com/project-codeflare/codeflare-operator/api/codeflare/v1alpha1"
)

var multiClusterAppDispatcherTemplates = []string{
	"mcad/configmap.yaml.tmpl",
	"mcad/serviceaccount.yaml.tmpl",
	"mcad/deployment.yaml.tmpl",
}
var ownerLessmultiClusterAppDispatcherTemplates = []string{
	"mcad/clusterrole_mcad-controller.yaml.tmpl",
	"mcad/clusterrolebinding_mcad-controller.yaml.tmpl",
	"mcad/clusterrolebinding_mcad-controller-kube-scheduler.yaml.tmpl",
	"mcad/clusterrolebinding_mcad-edit.yaml.tmpl",
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
