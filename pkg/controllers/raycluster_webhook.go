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
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var rayclusterlog = logf.Log.WithName("raycluster-resource")

func (r *RayClusterDefaulter) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&rayv1.RayCluster{}).
		WithDefaulter(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-ray-io-v1-raycluster,mutating=true,failurePolicy=fail,sideEffects=None,groups=ray.io,resources=rayclusters,verbs=create;update,versions=v1,name=mraycluster.kb.io,admissionReviewVersions=v1

type RayClusterDefaulter struct{}

var _ webhook.CustomDefaulter = &RayClusterDefaulter{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *RayClusterDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	raycluster := obj.(*rayv1.RayCluster)

	rayclusterlog.Info("default", "name", raycluster.Name)
	// Check and add OAuth proxy if it does not exist.
	alreadyExists := false
	for _, container := range raycluster.Spec.HeadGroupSpec.Template.Spec.Containers {
		if container.Name == "oauth-proxy" {
			rayclusterlog.Info("OAuth sidecar already exists, no patch needed")
			alreadyExists = true
			break // exits the for loop
		}
	}

	if !alreadyExists {
		rayclusterlog.Info("Adding OAuth sidecar container")
		// definition of the new container
		newOAuthSidecar := corev1.Container{
			Name:  "oauth-proxy",
			Image: "registry.redhat.io/openshift4/ose-oauth-proxy@sha256:1ea6a01bf3e63cdcf125c6064cbd4a4a270deaf0f157b3eabb78f60556840366",
			Ports: []corev1.ContainerPort{
				{ContainerPort: 8443, Name: "oauth-proxy"},
			},
			Args: []string{
				"--https-address=:8443",
				"--provider=openshift",
				"--openshift-service-account=" + raycluster.Name + "-oauth-proxy",
				"--upstream=http://localhost:8265",
				"--tls-cert=/etc/tls/private/tls.crt",
				"--tls-key=/etc/tls/private/tls.key",
				"--cookie-secret=$(COOKIE_SECRET)",
				"--openshift-delegate-urls={\"/\":{\"resource\":\"pods\",\"namespace\":\"default\",\"verb\":\"get\"}}",
			},
			Env: []corev1.EnvVar{
				{
					Name: "COOKIE_SECRET",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: raycluster.Name + "-oauth-config",
							},
							Key: "cookie_secret",
						},
					},
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "proxy-tls-secret",
					MountPath: "/etc/tls/private",
					ReadOnly:  true,
				},
			},
		}

		// Adding the new OAuth sidecar container
		raycluster.Spec.HeadGroupSpec.Template.Spec.Containers = append(raycluster.Spec.HeadGroupSpec.Template.Spec.Containers, newOAuthSidecar)

		tlsSecretVolume := corev1.Volume{
			Name: "proxy-tls-secret",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: raycluster.Name + "-proxy-tls-secret",
				},
			},
		}

		raycluster.Spec.HeadGroupSpec.Template.Spec.Volumes = append(raycluster.Spec.HeadGroupSpec.Template.Spec.Volumes, tlsSecretVolume)

		// Ensure the service account is set
		if raycluster.Spec.HeadGroupSpec.Template.Spec.ServiceAccountName == "" {
			raycluster.Spec.HeadGroupSpec.Template.Spec.ServiceAccountName = raycluster.Name + "-oauth-proxy"
		}
	}
	return nil
}
