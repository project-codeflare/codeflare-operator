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
	"context"
	"fmt"
	"github.com/go-logr/logr"
	mf "github.com/manifestival/manifestival"
	"github.com/project-codeflare/codeflare-operator/controllers/config"

	codeflarev1alpha1 "github.com/project-codeflare/codeflare-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const finalizerName = "codeflare.codeflare.dev/finalizer"

// MCADReconciler reconciles a MCAD object
type MCADReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	Log           logr.Logger
	TemplatesPath string
}

func (r *MCADReconciler) Apply(owner mf.Owner, params *MCADParams, template string, fns ...mf.Transformer) error {

	tmplManifest, err := config.Manifest(r.Client, r.TemplatesPath+template, params, template)
	if err != nil {
		return fmt.Errorf("error loading template yaml: %w", err)
	}
	tmplManifest, err = tmplManifest.Transform(
		mf.InjectOwner(owner),
	)
	if err != nil {
		return err
	}

	tmplManifest, err = tmplManifest.Transform(fns...)
	if err != nil {
		return err
	}

	if err = tmplManifest.Apply(); err != nil {
		return err
	}
	return nil
}

//+kubebuilder:rbac:groups=codeflare.codeflare.dev,resources=mcads,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=codeflare.codeflare.dev,resources=mcads/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=codeflare.codeflare.dev,resources=mcads/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=*,resources=deployments;services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets;configmaps;services;serviceaccounts;persistentvolumes;persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=persistentvolumes;persistentvolumeclaims,verbs=*
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings,verbs=get;list;watch;create;update;delete

func (r *MCADReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("namespace", req.Namespace)

	log.V(1).Info("MCAD reconciler called.")

	params := &MCADParams{}
	mcadCustomResource := &codeflarev1alpha1.MCAD{}

	err := r.Get(ctx, req.NamespacedName, mcadCustomResource)
	if err != nil && apierrs.IsNotFound(err) {
		log.Info("Stop MCAD reconciliation")
		return ctrl.Result{}, nil
	} else if err != nil {
		log.Error(err, "Unable to fetch the MCAD custom resource")
		return ctrl.Result{}, err
	}

	// FixMe: Hack for stubbing gvk during tests as these are not populated by test suite
	// Refer to https://github.com/operator-framework/operator-sdk/issues/727#issuecomment-581169171
	// In production we expect these to be populated
	if mcadCustomResource.Kind == "" {
		mcadCustomResource = mcadCustomResource.DeepCopy()
		gvk := codeflarev1alpha1.GroupVersion.WithKind("MCAD")
		mcadCustomResource.APIVersion, mcadCustomResource.Kind = gvk.Version, gvk.Kind
	}

	err = params.ExtractParams(mcadCustomResource)
	if err != nil {
		log.Error(err, "Unable to parse MCAD custom resource")
		return ctrl.Result{}, err
	}

	if mcadCustomResource.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(mcadCustomResource, finalizerName) {
			controllerutil.AddFinalizer(mcadCustomResource, finalizerName)
			if err := r.Update(ctx, mcadCustomResource); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(mcadCustomResource, finalizerName) {
			if err := r.cleanUpClusterResources(ctx, req, mcadCustomResource, params); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(mcadCustomResource, finalizerName)
			if err := r.Update(ctx, mcadCustomResource); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	log.V(1).Info("ReconcileMCAD called.")
	err = r.ReconcileMCAD(ctx, mcadCustomResource, req, params)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MCADReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&codeflarev1alpha1.MCAD{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&rbacv1.RoleBinding{}).
		Complete(r)
}

// cleanUpClusterResources will be responsible for deleting objects that do not have owner references set
func (r *MCADReconciler) cleanUpClusterResources(ctx context.Context, req ctrl.Request, mcad *codeflarev1alpha1.MCAD, params *MCADParams) error {

	return nil
}
