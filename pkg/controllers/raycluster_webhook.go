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
	"strconv"

	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/project-codeflare/codeflare-operator/pkg/config"
)

const (
	oauthProxyContainerName = "oauth-proxy"
	oauthProxyVolumeName    = "proxy-tls-secret"
	initContainerName       = "create-cert"
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

	if ptr.Deref(w.Config.RayDashboardOAuthEnabled, true) {
		rayclusterlog.V(2).Info("Adding OAuth sidecar container")
		rayCluster.Spec.HeadGroupSpec.Template.Spec.Containers = upsert(rayCluster.Spec.HeadGroupSpec.Template.Spec.Containers, oauthProxyContainer(rayCluster), withContainerName(oauthProxyContainerName))

		rayCluster.Spec.HeadGroupSpec.Template.Spec.Volumes = upsert(rayCluster.Spec.HeadGroupSpec.Template.Spec.Volumes, oauthProxyTLSSecretVolume(rayCluster), withVolumeName(oauthProxyVolumeName))

		rayCluster.Spec.HeadGroupSpec.Template.Spec.ServiceAccountName = rayCluster.Name + "-oauth-proxy"
	}

	if ptr.Deref(w.Config.MTLSEnabled, true) {
		rayclusterlog.V(2).Info("Adding create-cert Init Containers")
		// HeadGroupSpec

		// Append the list of environment variables for the ray-head container
		for _, envVar := range envVarList() {
			rayCluster.Spec.HeadGroupSpec.Template.Spec.Containers[0].Env = upsert(rayCluster.Spec.HeadGroupSpec.Template.Spec.Containers[0].Env, envVar, withEnvVarName(envVar.Name))
		}

		// Append the create-cert Init Container
		rayCluster.Spec.HeadGroupSpec.Template.Spec.InitContainers = upsert(rayCluster.Spec.HeadGroupSpec.Template.Spec.InitContainers, rayHeadInitContainer(rayCluster, w.Config), withContainerName(initContainerName))

		// Append the CA volumes
		for _, caVol := range caVolumes(rayCluster) {
			rayCluster.Spec.HeadGroupSpec.Template.Spec.Volumes = upsert(rayCluster.Spec.HeadGroupSpec.Template.Spec.Volumes, caVol, withVolumeName(caVol.Name))
		}

		// Append the certificate volume mounts
		for _, mount := range certVolumeMounts() {
			rayCluster.Spec.HeadGroupSpec.Template.Spec.Containers[0].VolumeMounts = upsert(rayCluster.Spec.HeadGroupSpec.Template.Spec.Containers[0].VolumeMounts, mount, byVolumeMountName)
		}

		// WorkerGroupSpec
		for i := range rayCluster.Spec.WorkerGroupSpecs {
			workerSpec := &rayCluster.Spec.WorkerGroupSpecs[i]

			// Append the list of environment variables for the worker container
			for _, envVar := range envVarList() {
				workerSpec.Template.Spec.Containers[0].Env = upsert(workerSpec.Template.Spec.Containers[0].Env, envVar, withEnvVarName(envVar.Name))
			}

			// Append the CA volumes
			for _, caVol := range caVolumes(rayCluster) {
				workerSpec.Template.Spec.Volumes = upsert(workerSpec.Template.Spec.Volumes, caVol, withVolumeName(caVol.Name))
			}

			// Append the certificate volume mounts
			for _, mount := range certVolumeMounts() {
				workerSpec.Template.Spec.Containers[0].VolumeMounts = upsert(workerSpec.Template.Spec.Containers[0].VolumeMounts, mount, byVolumeMountName)
			}

			// Append the create-cert Init Container
			workerSpec.Template.Spec.InitContainers = upsert(workerSpec.Template.Spec.InitContainers, rayWorkerInitContainer(w.Config), withContainerName(initContainerName))
		}
	}

	return nil
}

func (w *rayClusterWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	rayCluster := obj.(*rayv1.RayCluster)

	var warnings admission.Warnings
	var allErrors field.ErrorList

	allErrors = append(allErrors, validateIngress(rayCluster)...)

	if ptr.Deref(w.Config.RayDashboardOAuthEnabled, true) {
		allErrors = append(allErrors, validateOAuthProxyContainer(rayCluster)...)
		allErrors = append(allErrors, validateOAuthProxyVolume(rayCluster)...)
		allErrors = append(allErrors, validateHeadGroupServiceAccountName(rayCluster)...)
	}

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

	if ptr.Deref(w.Config.RayDashboardOAuthEnabled, true) {
		allErrors = append(allErrors, validateOAuthProxyContainer(rayCluster)...)
		allErrors = append(allErrors, validateOAuthProxyVolume(rayCluster)...)
		allErrors = append(allErrors, validateHeadGroupServiceAccountName(rayCluster)...)
	}

	// Init Container related errors
	if ptr.Deref(w.Config.MTLSEnabled, true) {
		allErrors = append(allErrors, validateHeadInitContainer(rayCluster, w.Config)...)
		allErrors = append(allErrors, validateWorkerInitContainer(rayCluster, w.Config)...)
		allErrors = append(allErrors, validateHeadEnvVars(rayCluster)...)
		allErrors = append(allErrors, validateWorkerEnvVars(rayCluster)...)
		allErrors = append(allErrors, validateCaVolumes(rayCluster)...)
	}
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

	if ptr.Deref(rayCluster.Spec.HeadGroupSpec.EnableIngress, false) {
		allErrors = append(allErrors, field.Invalid(
			field.NewPath("spec", "headGroupSpec", "enableIngress"),
			rayCluster.Spec.HeadGroupSpec.EnableIngress,
			"RayCluster with enableIngress set to true is not allowed"))
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
			"--openshift-delegate-urls={\"/\":{\"resource\":\"pods\",\"namespace\":\"" + rayCluster.Namespace + "\",\"verb\":\"get\"}}",
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

func certVolumeMounts() []corev1.VolumeMount {
	return []corev1.VolumeMount{
		{
			Name:      "ca-vol",
			MountPath: "/home/ray/workspace/ca",
			ReadOnly:  true,
		},
		{
			Name:      "server-cert",
			MountPath: "/home/ray/workspace/tls",
			ReadOnly:  false,
		},
	}
}

func envVarList() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name: "MY_POD_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "status.podIP",
				},
			},
		},
		{
			Name:  "RAY_USE_TLS",
			Value: "1",
		},
		{
			Name:  "RAY_TLS_SERVER_CERT",
			Value: "/home/ray/workspace/tls/server.crt",
		},
		{
			Name:  "RAY_TLS_SERVER_KEY",
			Value: "/home/ray/workspace/tls/server.key",
		},
		{
			Name:  "RAY_TLS_CA_CERT",
			Value: "/home/ray/workspace/tls/ca.crt",
		},
	}
}

func caVolumes(rayCluster *rayv1.RayCluster) []corev1.Volume {
	return []corev1.Volume{
		{
			Name: "ca-vol",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: `ca-secret-` + rayCluster.Name,
				},
			},
		},
		{
			Name: "server-cert",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}
}

func rayHeadInitContainer(rayCluster *rayv1.RayCluster, config *config.KubeRayConfiguration) corev1.Container {
	rayClientRoute := "rayclient-" + rayCluster.Name + "-" + rayCluster.Namespace + "." + config.IngressDomain
	// Service name for basic interactive
	svcDomain := rayCluster.Name + "-head-svc." + rayCluster.Namespace + ".svc"

	initContainerHead := corev1.Container{
		Name:  "create-cert",
		Image: config.CertGeneratorImage,
		Command: []string{
			"sh",
			"-c",
			`cd /home/ray/workspace/tls && openssl req -nodes -newkey rsa:2048 -keyout server.key -out server.csr -subj '/CN=ray-head' && printf "authorityKeyIdentifier=keyid,issuer\nbasicConstraints=CA:FALSE\nsubjectAltName = @alt_names\n[alt_names]\nDNS.1 = 127.0.0.1\nDNS.2 = localhost\nDNS.3 = ${FQ_RAY_IP}\nDNS.4 = $(awk 'END{print $1}' /etc/hosts)\nDNS.5 = ` + rayClientRoute + `\nDNS.6 = ` + svcDomain + `">./domain.ext && cp /home/ray/workspace/ca/* . && openssl x509 -req -CA ca.crt -CAkey ca.key -in server.csr -out server.crt -days 365 -CAcreateserial -extfile domain.ext`,
		},
		VolumeMounts: certVolumeMounts(),
	}
	return initContainerHead
}

func rayWorkerInitContainer(config *config.KubeRayConfiguration) corev1.Container {
	initContainerWorker := corev1.Container{
		Name:  "create-cert",
		Image: config.CertGeneratorImage,
		Command: []string{
			"sh",
			"-c",
			`cd /home/ray/workspace/tls && openssl req -nodes -newkey rsa:2048 -keyout server.key -out server.csr -subj '/CN=ray-head' && printf "authorityKeyIdentifier=keyid,issuer\nbasicConstraints=CA:FALSE\nsubjectAltName = @alt_names\n[alt_names]\nDNS.1 = 127.0.0.1\nDNS.2 = localhost\nDNS.3 = ${FQ_RAY_IP}\nDNS.4 = $(awk 'END{print $1}' /etc/hosts)">./domain.ext && cp /home/ray/workspace/ca/* . && openssl x509 -req -CA ca.crt -CAkey ca.key -in server.csr -out server.crt -days 365 -CAcreateserial -extfile domain.ext`,
		},
		VolumeMounts: certVolumeMounts(),
	}
	return initContainerWorker
}

func validateHeadInitContainer(rayCluster *rayv1.RayCluster, config *config.KubeRayConfiguration) field.ErrorList {
	var allErrors field.ErrorList

	if err := contains(rayCluster.Spec.HeadGroupSpec.Template.Spec.InitContainers, rayHeadInitContainer(rayCluster, config), byContainerName,
		field.NewPath("spec", "headGroupSpec", "template", "spec", "initContainers"),
		"create-cert Init Container is immutable"); err != nil {
		allErrors = append(allErrors, err)
	}

	return allErrors
}

func validateWorkerInitContainer(rayCluster *rayv1.RayCluster, config *config.KubeRayConfiguration) field.ErrorList {
	var allErrors field.ErrorList

	for i := range rayCluster.Spec.WorkerGroupSpecs {
		workerSpec := &rayCluster.Spec.WorkerGroupSpecs[i]
		if err := contains(workerSpec.Template.Spec.InitContainers, rayWorkerInitContainer(config), byContainerName,
			field.NewPath("spec", "workerGroupSpecs", strconv.Itoa(i), "template", "spec", "initContainers"),
			"create-cert Init Container is immutable"); err != nil {
			allErrors = append(allErrors, err)
		}
	}

	return allErrors
}

func validateCaVolumes(rayCluster *rayv1.RayCluster) field.ErrorList {
	var allErrors field.ErrorList

	for _, caVol := range caVolumes(rayCluster) {
		if err := contains(rayCluster.Spec.HeadGroupSpec.Template.Spec.Volumes, caVol, byVolumeName,
			field.NewPath("spec", "headGroupSpec", "template", "spec", "volumes"),
			"ca-vol and server-cert Secret volumes are immutable"); err != nil {
			allErrors = append(allErrors, err)
		}
		for i := range rayCluster.Spec.WorkerGroupSpecs {
			workerSpec := &rayCluster.Spec.WorkerGroupSpecs[i]
			if err := contains(workerSpec.Template.Spec.Volumes, caVol, byVolumeName,
				field.NewPath("spec", "workerGroupSpecs", strconv.Itoa(i), "template", "spec", "volumes"),
				"ca-vol and server-cert Secret volumes are immutable"); err != nil {
				allErrors = append(allErrors, err)
			}
		}
	}

	return allErrors
}

func validateHeadEnvVars(rayCluster *rayv1.RayCluster) field.ErrorList {
	var allErrors field.ErrorList

	for _, envVar := range envVarList() {
		if err := contains(rayCluster.Spec.HeadGroupSpec.Template.Spec.Containers[0].Env, envVar, byEnvVarName,
			field.NewPath("spec", "headGroupSpec", "template", "spec", "containers", strconv.Itoa(0), "env"),
			"RAY_TLS related environment variables are immutable"); err != nil {
			allErrors = append(allErrors, err)
		}
	}

	return allErrors
}

func validateWorkerEnvVars(rayCluster *rayv1.RayCluster) field.ErrorList {
	var allErrors field.ErrorList

	for i := range rayCluster.Spec.WorkerGroupSpecs {
		workerSpec := &rayCluster.Spec.WorkerGroupSpecs[i]
		for _, envVar := range envVarList() {
			if err := contains(workerSpec.Template.Spec.Containers[0].Env, envVar, byEnvVarName,
				field.NewPath("spec", "workerGroupSpecs", strconv.Itoa(i), "template", "spec", "containers", strconv.Itoa(0), "env"),
				"RAY_TLS related environment variables are immutable"); err != nil {
				allErrors = append(allErrors, err)
			}
		}
	}

	return allErrors
}
