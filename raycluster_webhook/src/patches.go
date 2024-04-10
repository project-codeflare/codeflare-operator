package main

import (
	"fmt"

	rayv1api "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"
	corev1 "k8s.io/api/core/v1"
)

func createPatch(rayCluster *rayv1api.RayCluster) ([]patchOperation, error) {
	fmt.Printf("creating json patch for RayCluster")

	var patches []patchOperation
	rayclusterName := rayCluster.Name

	// Check if the OAuth sidecar container already exists
	for _, container := range rayCluster.Spec.HeadGroupSpec.Template.Spec.Containers {
		if container.Name == "oauth-proxy" {
			fmt.Println("OAuth sidecar already exists, no patch needed")
			return patches, nil
		}
	}

	// New OAuth sidecar configuration based on provided example
	newOAuthSidecar := corev1.Container{
		Name:  "oauth-proxy",
		Image: "registry.redhat.io/openshift4/ose-oauth-proxy@sha256:1ea6a01bf3e63cdcf125c6064cbd4a4a270deaf0f157b3eabb78f60556840366",
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: 8443,
				Name:          "oauth-proxy",
			},
		},
		Args: []string{
			"--https-address=:8443",
			"--provider=openshift",
			"--openshift-service-account=" + rayclusterName + "-oauth-proxy",
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
							Name: rayclusterName + "-oauth-config",
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

	// Check if the service account is set, and if not, set it
	if rayCluster.Spec.HeadGroupSpec.Template.Spec.ServiceAccountName == "" {
		patches = append(patches, patchOperation{
			Op:    "add",
			Path:  "/spec/headGroupSpec/template/spec/serviceAccountName",
			Value: rayclusterName + "-oauth-proxy",
		})
	}

	// Patch to add new OAuth sidecar
	patches = append(patches, patchOperation{
		Op:    "add",
		Path:  "/spec/headGroupSpec/template/spec/containers/-",
		Value: newOAuthSidecar,
	})

	return patches, nil
}
