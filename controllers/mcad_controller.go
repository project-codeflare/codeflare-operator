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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/project-codeflare/codeflare-operator/api/codeflare/v1alpha1"
	"github.com/project-codeflare/codeflare-operator/controllers/config"
	"github.com/project-codeflare/codeflare-operator/controllers/util"
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
	tmplManifest, err := config.Manifest(r.Client, r.TemplatesPath+template, params, template, r.Log)
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

	return tmplManifest.Apply()
}

func (r *MCADReconciler) ApplyWithoutOwner(params *MCADParams, template string, fns ...mf.Transformer) error {
	tmplManifest, err := config.Manifest(r.Client, r.TemplatesPath+template, params, template, r.Log)
	if err != nil {
		return fmt.Errorf("error loading template yaml: %w", err)
	}

	tmplManifest, err = tmplManifest.Transform(fns...)
	if err != nil {
		return err
	}

	return tmplManifest.Apply()
}

func (r *MCADReconciler) DeleteResource(params *MCADParams, template string, fns ...mf.Transformer) error {
	tmplManifest, err := config.Manifest(r.Client, r.TemplatesPath+template, params, template, r.Log)
	if err != nil {
		return fmt.Errorf("error loading template yaml: %w", err)
	}

	tmplManifest, err = tmplManifest.Transform(fns...)
	if err != nil {
		return err
	}

	return tmplManifest.Delete()
}

// +kubebuilder:rbac:groups=codeflare.codeflare.dev,resources=mcads,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=codeflare.codeflare.dev,resources=mcads/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=codeflare.codeflare.dev,resources=mcads/finalizers,verbs=update
// +kubebuilder:rbac:groups=mcad.ibm.com;ibm.com,resources=xqueuejobs;queuejobs;schedulingspecs;appwrappers;appwrappers/finalizers;appwrappers/status;quotasubtrees,verbs=get;list;watch;create;update;patch;delete;deletecollection
// +kubebuilder:rbac:groups=core,resources=pods;lists;namespaces,verbs=get;list;watch;create;update;patch;delete;deletecollection
// +kubebuilder:rbac:groups=core,resources=bindings;pods/binding,verbs=create
// +kubebuilder:rbac:groups=core,resources=kube-scheduler,verbs=get;update
// +kubebuilder:rbac:groups=core,resources=endpoints;kube-scheduler,verbs=create;get;update
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch;update
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=patch;update
// +kubebuilder:rbac:groups=core,resources=replicationcontrollers,verbs=get;list;watch
// +kubebuilder:rbac:groups=scheduling.sigs.k8s.io,resources=podgroups,verbs=get;list;watch;create;update;patch;delete;deletecollection
// +kubebuilder:rbac:groups=apps,resources=deployments;replicasets;statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=*,resources=deployments;services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets;configmaps;services;serviceaccounts;persistentvolumes;persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=persistentvolumes;persistentvolumeclaims,verbs=*
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=custom.metrics.k8s.io,resources=*,verbs=*
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases;kube-scheduler,verbs=create;update;get
// +kubebuilder:rbac:groups=events.k8s.io,resources=events;kube-scheduler,verbs=create;update;patch
// +kubebuilder:rbac:groups=extensions,resources=replicasets,verbs=get;list;watch
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch
// +kubebuilder:rbac:groups=storage.k8s.io,resources=csidrivers;csinodes;csistoragecapacities,verbs=get;list;watch
// +kubebuilder:rbac:groups=authentication.k8s.io,resources=tokenreviews,verbs=create
// +kubebuilder:rbac:groups=authorization.k8s.io,resources=subjectaccessreviews,verbs=create

func (r *MCADReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("namespace", req.Namespace)

	log.V(1).Info("MCAD reconciler called.")

	params := &MCADParams{}
	mcadCustomResource := &v1alpha1.MCAD{}

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
		gvk := v1alpha1.SchemeGroupVersion.WithKind("MCAD")
		mcadCustomResource.APIVersion, mcadCustomResource.Kind = gvk.Version, gvk.Kind
	}

	params.ExtractParams(mcadCustomResource)

	if mcadCustomResource.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(mcadCustomResource, finalizerName) {
			controllerutil.AddFinalizer(mcadCustomResource, finalizerName)
			if err := r.Update(ctx, mcadCustomResource); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(mcadCustomResource, finalizerName) {
			if err := r.cleanUpOwnerLessResources(params); err != nil {
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
	err = r.ReconcileMCAD(mcadCustomResource, params)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = updateMCADReadyStatus(ctx, r, req, mcadCustomResource)
	if err != nil {
		return ctrl.Result{}, err
	}
	err = r.Client.Status().Update(ctx, mcadCustomResource)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func updateMCADReadyStatus(ctx context.Context, r *MCADReconciler, req ctrl.Request, mcadCustomResource *v1alpha1.MCAD) error {
	deployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("mcad-controller-%s", req.Name), Namespace: req.Namespace}, deployment)
	if err != nil {
		return err
	}
	r.Log.Info("Checking if MCAD deployment is ready.")
	mcadCustomResource.Status.Ready = util.IsDeploymentReady(deployment)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MCADReconciler) SetupWithManager(mgr ctrl.Manager) error {
	crFromLabels := handler.EnqueueRequestsFromMapFunc(func(o client.Object) []reconcile.Request {
		labels := o.GetLabels()
		if labels["app.kubernetes.io/managed-by"] == "MCAD" {
			crName := labels["codeflare.codeflare.dev/cr-name"]
			crNamespace := labels["codeflare.codeflare.dev/cr-namespace"]
			if crName != "" {
				return []reconcile.Request{
					{NamespacedName: types.NamespacedName{
						Name:      crName,
						Namespace: crNamespace,
					}},
				}
			}
		}
		return nil
	})
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.MCAD{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&rbacv1.RoleBinding{}).
		Owns(&appsv1.Deployment{}).
		Watches(
			&source.Kind{Type: &rbacv1.ClusterRole{}},
			crFromLabels,
		).
		Watches(
			&source.Kind{Type: &rbacv1.ClusterRoleBinding{}},
			crFromLabels,
		).
		Complete(r)
}

// cleanUpClusterResources will be responsible for deleting objects that do not have owner references set
func (r *MCADReconciler) cleanUpOwnerLessResources(params *MCADParams) error {
	return r.deleteOwnerLessObjects(params)
}
