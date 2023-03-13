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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	authv1 "k8s.io/api/rbac/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/go-logr/logr"
	mf "github.com/manifestival/manifestival"
	codeflarev1alpha1 "github.com/project-codeflare/codeflare-operator/api/v1alpha1"
	"github.com/project-codeflare/codeflare-operator/controllers/config"
	"github.com/project-codeflare/codeflare-operator/controllers/util"
)

// InstaScaleReconciler reconciles a InstaScale object
type InstaScaleReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	Log           logr.Logger
	TemplatesPath string
}

var instascaleClusterScopedTemplates = []string{
	"instascale/clusterrole.yaml.tmpl",
	"instascale/clusterrolebinding.yaml.tmpl",
}

func (r *InstaScaleReconciler) Apply(owner mf.Owner, params *InstaScaleParams, template string, fns ...mf.Transformer) error {

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

	if err = tmplManifest.Apply(); err != nil {
		return err
	}
	return nil
}

// TODO: Review node permissions, instascale should only require read

//+kubebuilder:rbac:groups=codeflare.codeflare.dev,resources=instascales,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=codeflare.codeflare.dev,resources=instascales/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=codeflare.codeflare.dev,resources=instascales/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=*,resources=deployments;services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=secrets;configmaps;nodes;services;serviceaccounts;persistentvolumes;persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=persistentvolumes;persistentvolumeclaims,verbs=*
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups=machine.openshift.io,resources=*,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=mcad.ibm.com,resources=appwrappers;queuejobs;schedulingspecs,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the InstaScale object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *InstaScaleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("namespace", req.Namespace)

	log.V(1).Info("InstaScale reconciler called.")

	params := &InstaScaleParams{}
	instascaleCustomResource := &codeflarev1alpha1.InstaScale{}

	err := r.Get(ctx, req.NamespacedName, instascaleCustomResource)
	if err != nil && apierrs.IsNotFound(err) {
		log.Info("Stop InstaScale reconciliation")
		return ctrl.Result{}, nil
	} else if err != nil {
		log.Error(err, "Unable to fetch the InstaScale custom resource")
		return ctrl.Result{}, err
	}

	// FixMe: Hack for stubbing gvk during tests as these are not populated by test suite
	// Refer to https://github.com/operator-framework/operator-sdk/issues/727#issuecomment-581169171
	// In production we expect these to be populated
	if instascaleCustomResource.Kind == "" {
		instascaleCustomResource = instascaleCustomResource.DeepCopy()
		gvk := codeflarev1alpha1.GroupVersion.WithKind("InstaScale")
		instascaleCustomResource.APIVersion, instascaleCustomResource.Kind = gvk.Version, gvk.Kind
	}

	err = params.ExtractParams(instascaleCustomResource)
	if err != nil {
		log.Error(err, "Unable to parse InstaScale custom resource")
		return ctrl.Result{}, err
	}

	if instascaleCustomResource.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(instascaleCustomResource, finalizerName) {
			controllerutil.AddFinalizer(instascaleCustomResource, finalizerName)
			if err := r.Update(ctx, instascaleCustomResource); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(instascaleCustomResource, finalizerName) {
			if err := r.cleanUpClusterResources(ctx, req, instascaleCustomResource, params); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(instascaleCustomResource, finalizerName)
			if err := r.Update(ctx, instascaleCustomResource); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	log.V(1).Info("ReconcileInstaScale called.")
	err = r.ReconcileInstaScale(ctx, instascaleCustomResource, req, params)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = updateInstascaleReadyStatus(ctx, r, req, instascaleCustomResource)
	if err != nil {
		return ctrl.Result{}, err
	}
	err = r.Client.Status().Update(context.Background(), instascaleCustomResource)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func updateInstascaleReadyStatus(ctx context.Context, r *InstaScaleReconciler, req ctrl.Request, instascaleCustomResource *codeflarev1alpha1.InstaScale) error {
	deployment := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("instascale-%s", req.Name), Namespace: req.Namespace}, deployment)
	if err != nil {
		return err
	}
	r.Log.Info("Checking if deployment is ready.")
	instascaleCustomResource.Status.Ready = util.IsDeploymentReady(deployment)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *InstaScaleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&codeflarev1alpha1.InstaScale{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&authv1.ClusterRole{}).
		Owns(&authv1.ClusterRoleBinding{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

func (r *InstaScaleReconciler) DeleteResource(params *InstaScaleParams, template string, fns ...mf.Transformer) error {
	tmplManifest, err := config.Manifest(r.Client, r.TemplatesPath+template, params, template, r.Log)
	if err != nil {
		return fmt.Errorf("error loading template yaml: %w", err)
	}

	tmplManifest, err = tmplManifest.Transform(fns...)
	if err != nil {
		return err
	}

	if err = tmplManifest.Delete(); err != nil {
		return err
	}
	return nil
}

// cleanUpClusterResources will be responsible for deleting objects that do not have owner references set
func (r *InstaScaleReconciler) cleanUpClusterResources(ctx context.Context, req ctrl.Request, instascale *codeflarev1alpha1.InstaScale, params *InstaScaleParams) error {
	for _, template := range instascaleClusterScopedTemplates {
		err := r.DeleteResource(params, template)
		if err != nil {
			return err
		}
	}
	return nil
}
