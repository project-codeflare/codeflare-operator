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

	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/project-codeflare/codeflare-operator/pkg/config"
)

// log is for logging in this package.
var rayclusterlog = logf.Log.WithName("raycluster-resource")

func SetupRayClusterWebhookWithManager(mgr ctrl.Manager, cfg *config.KubeRayConfiguration) error {
	rayClusterWebhookInstance := &rayClusterWebhook{
		Config: cfg,
	}
	return ctrl.NewWebhookManagedBy(mgr).
		For(&rayv1.RayCluster{}).
		WithDefaulter(rayClusterWebhookInstance).
		WithValidator(rayClusterWebhookInstance).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-ray-io-v1-raycluster,mutating=true,failurePolicy=fail,sideEffects=None,groups=ray.io,resources=rayclusters,verbs=create;update,versions=v1,name=mraycluster.kb.io,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/validate-ray-io-v1-raycluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=ray.io,resources=rayclusters,verbs=create;update,versions=v1,name=vraycluster.kb.io,admissionReviewVersions=v1

type rayClusterWebhook struct {
	Config *config.KubeRayConfiguration
}

var _ webhook.CustomDefaulter = &rayClusterWebhook{}
var _ webhook.CustomValidator = &rayClusterWebhook{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (w *rayClusterWebhook) Default(ctx context.Context, obj runtime.Object) error {
	rayCluster := obj.(*rayv1.RayCluster)

	if !pointer.BoolDeref(w.Config.RayDashboardOAuthEnabled, true) {
		return nil
	}

	// Check and add OAuth proxy if it does not exist
	for _, container := range rayCluster.Spec.HeadGroupSpec.Template.Spec.Containers {
		if container.Name == "oauth-proxy" {
			rayclusterlog.V(2).Info("OAuth sidecar already exists, no patch needed")
			return nil
		}
	}

	rayclusterlog.V(2).Info("Adding OAuth sidecar container")

	newOAuthSidecar := corev1.Container{
		Name:  "oauth-proxy",
		Image: "registry.redhat.io/openshift4/ose-oauth-proxy@sha256:1ea6a01bf3e63cdcf125c6064cbd4a4a270deaf0f157b3eabb78f60556840366",
		Ports: []corev1.ContainerPort{
			{ContainerPort: 8443, Name: "oauth-proxy"},
		},
		Env: []corev1.EnvVar{
			{
				Name: "COOKIE_SECRET",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: rayCluster.Name + "-oauth-config",
						},
						Key: "cookie_secret",
					},
				},
			},
		},
		Args: []string{
			"--https-address=:8443",
			"--provider=openshift",
			"--openshift-service-account=" + rayCluster.Name + "-oauth-proxy",
			"--upstream=http://localhost:8265",
			"--tls-cert=/etc/tls/private/tls.crt",
			"--tls-key=/etc/tls/private/tls.key",
			"--cookie-secret=$(COOKIE_SECRET)",
			"--openshift-delegate-urls={\"/\":{\"resource\":\"pods\",\"namespace\":\"default\",\"verb\":\"get\"}}",
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "proxy-tls-secret",
				MountPath: "/etc/tls/private",
				ReadOnly:  true,
			},
		},
	}

	rayCluster.Spec.HeadGroupSpec.Template.Spec.Containers = append(rayCluster.Spec.HeadGroupSpec.Template.Spec.Containers, newOAuthSidecar)

	tlsSecretVolume := corev1.Volume{
		Name: "proxy-tls-secret",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: rayCluster.Name + "-proxy-tls-secret",
			},
		},
	}

	rayCluster.Spec.HeadGroupSpec.Template.Spec.Volumes = append(rayCluster.Spec.HeadGroupSpec.Template.Spec.Volumes, tlsSecretVolume)

	// Ensure the service account is set
	if rayCluster.Spec.HeadGroupSpec.Template.Spec.ServiceAccountName == "" {
		rayCluster.Spec.HeadGroupSpec.Template.Spec.ServiceAccountName = rayCluster.Name + "-oauth-proxy"
	}

	return nil
}

func (w *rayClusterWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	raycluster := obj.(*rayv1.RayCluster)
	var warnings admission.Warnings
	var allErrors field.ErrorList
	specPath := field.NewPath("spec")

	if pointer.BoolDeref(raycluster.Spec.HeadGroupSpec.EnableIngress, false) {
		rayclusterlog.Info("Creating RayCluster resources with EnableIngress set to true or unspecified is not allowed")
		allErrors = append(allErrors, field.Invalid(specPath.Child("headGroupSpec").Child("enableIngress"), raycluster.Spec.HeadGroupSpec.EnableIngress, "creating RayCluster resources with EnableIngress set to true or unspecified is not allowed"))
	}

	return warnings, allErrors.ToAggregate()
}

func (w *rayClusterWebhook) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	newRayCluster := newObj.(*rayv1.RayCluster)
	if !newRayCluster.DeletionTimestamp.IsZero() {
		// Object is being deleted, skip validations
		return nil, nil
	}
	warnings, err := w.ValidateCreate(ctx, newRayCluster)
	return warnings, err
}

func (w *rayClusterWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	// Optional: Add delete validation logic here
	return nil, nil
}
