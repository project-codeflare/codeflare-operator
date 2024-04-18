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

const (
	oauthProxyContainerName = "oauth-proxy"
	oauthProxyVolumeName    = "proxy-tls-secret"
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

// +kubebuilder:webhook:path=/mutate-ray-io-v1-raycluster,mutating=true,failurePolicy=fail,sideEffects=None,groups=ray.io,resources=rayclusters,verbs=create,versions=v1,name=mraycluster.ray.openshift.ai,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/validate-ray-io-v1-raycluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=ray.io,resources=rayclusters,verbs=create;update,versions=v1,name=vraycluster.ray.openshift.ai,admissionReviewVersions=v1

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

	rayclusterlog.V(2).Info("Adding OAuth sidecar container")

	rayCluster.Spec.HeadGroupSpec.Template.Spec.Containers = upsert(rayCluster.Spec.HeadGroupSpec.Template.Spec.Containers, oauthProxyContainer(rayCluster), withContainerName(oauthProxyContainerName))

	rayCluster.Spec.HeadGroupSpec.Template.Spec.Volumes = upsert(rayCluster.Spec.HeadGroupSpec.Template.Spec.Volumes, oauthProxyTLSSecretVolume(rayCluster), withVolumeName(oauthProxyVolumeName))

	rayCluster.Spec.HeadGroupSpec.Template.Spec.ServiceAccountName = rayCluster.Name + "-oauth-proxy"

	return nil
}

func (w *rayClusterWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	rayCluster := obj.(*rayv1.RayCluster)

	var warnings admission.Warnings
	var allErrors field.ErrorList

	allErrors = append(allErrors, validateIngress(rayCluster)...)

	return warnings, allErrors.ToAggregate()
}

func (w *rayClusterWebhook) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	rayCluster := newObj.(*rayv1.RayCluster)

	var warnings admission.Warnings
	var allErrors field.ErrorList

	if !rayCluster.DeletionTimestamp.IsZero() {
		// Object is being deleted, skip validations
		return nil, nil
	}

	allErrors = append(allErrors, validateIngress(rayCluster)...)
	allErrors = append(allErrors, validateOAuthProxyContainer(rayCluster)...)
	allErrors = append(allErrors, validateOAuthProxyVolume(rayCluster)...)
	allErrors = append(allErrors, validateHeadGroupServiceAccountName(rayCluster)...)

	return warnings, allErrors.ToAggregate()
}

func (w *rayClusterWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	// Optional: Add delete validation logic here
	return nil, nil
}

func validateOAuthProxyContainer(rayCluster *rayv1.RayCluster) field.ErrorList {
	var allErrors field.ErrorList

	if err := contains(rayCluster.Spec.HeadGroupSpec.Template.Spec.Containers, oauthProxyContainer(rayCluster), byContainerName,
		field.NewPath("spec", "headGroupSpec", "template", "spec", "containers"),
		"OAuth Proxy container is immutable"); err != nil {
		allErrors = append(allErrors, err)
	}

	return allErrors
}

func validateOAuthProxyVolume(rayCluster *rayv1.RayCluster) field.ErrorList {
	var allErrors field.ErrorList

	if err := contains(rayCluster.Spec.HeadGroupSpec.Template.Spec.Volumes, oauthProxyTLSSecretVolume(rayCluster), byVolumeName,
		field.NewPath("spec", "headGroupSpec", "template", "spec", "volumes"),
		"OAuth Proxy TLS Secret volume is immutable"); err != nil {
		allErrors = append(allErrors, err)
	}

	return allErrors
}

func validateIngress(rayCluster *rayv1.RayCluster) field.ErrorList {
	var allErrors field.ErrorList

	if pointer.BoolDeref(rayCluster.Spec.HeadGroupSpec.EnableIngress, false) {
		allErrors = append(allErrors, field.Invalid(
			field.NewPath("spec", "headGroupSpec", "enableIngress"),
			rayCluster.Spec.HeadGroupSpec.EnableIngress,
			"RayCluster resources with EnableIngress set to true or unspecified is not allowed"))
	}

	return allErrors
}

func validateHeadGroupServiceAccountName(rayCluster *rayv1.RayCluster) field.ErrorList {
	var allErrors field.ErrorList

	if rayCluster.Spec.HeadGroupSpec.Template.Spec.ServiceAccountName != rayCluster.Name+"-oauth-proxy" {
		allErrors = append(allErrors, field.Invalid(
			field.NewPath("spec", "headGroupSpec", "template", "spec", "serviceAccountName"),
			rayCluster.Spec.HeadGroupSpec.Template.Spec.ServiceAccountName,
			"RayCluster head group service account is immutable"))
	}

	return allErrors
}

func oauthProxyContainer(rayCluster *rayv1.RayCluster) corev1.Container {
	return corev1.Container{
		Name:  oauthProxyContainerName,
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
				Name:      oauthProxyVolumeName,
				MountPath: "/etc/tls/private",
				ReadOnly:  true,
			},
		},
	}
}

func oauthProxyTLSSecretVolume(rayCluster *rayv1.RayCluster) corev1.Volume {
	return corev1.Volume{
		Name: oauthProxyVolumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: rayCluster.Name + "-proxy-tls-secret",
			},
		},
	}
}
