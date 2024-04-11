package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	routev1 "github.com/openshift/api/route/v1"
	rayv1api "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	NameConsoleLink      string = "console"
	NamespaceConsoleLink string = "openshift-console"
)

func createPatch(rayCluster *rayv1api.RayCluster) ([]patchOperation, error) {
	oauthExists := false
	initHeadExists := false
	workerHeadExists := false
	fmt.Println("creating json patch for RayCluster")

	var patches []patchOperation
	rayclusterName := rayCluster.Name

	// Check if the OAuth sidecar container already exists
	for _, container := range rayCluster.Spec.HeadGroupSpec.Template.Spec.Containers {
		if container.Name == "oauth-proxy" {
			fmt.Println("OAuth sidecar already exists, no patch needed")
			oauthExists = true
		}
	}

	// Check for the create-cert Init Containers
	for _, container := range rayCluster.Spec.HeadGroupSpec.Template.Spec.InitContainers {
		if container.Name == "create-cert" {
			fmt.Println("Head Init Containers already exist, no patch needed")
			//return patches, nil
			initHeadExists = true
		}
	}
	// Check fot the create-cert Init Container WorkerGroupSpec
	for _, container := range rayCluster.Spec.WorkerGroupSpecs[0].Template.Spec.InitContainers {
		if container.Name == "create-cert" {
			fmt.Println("Worker Init Containers already exist, no patch needed")
			//return patches, nil
			workerHeadExists = true
		}
	}

	if !oauthExists {
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

		tlsSecretVolume := corev1.Volume{
			Name: "proxy-tls-secret",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: rayclusterName + "-proxy-tls-secret",
				},
			},
		}

		// Patch to add new volume
		patches = append(patches, patchOperation{
			Op:    "add",
			Path:  "/spec/headGroupSpec/template/spec/volumes/-",
			Value: tlsSecretVolume,
		})
	}

	patches, err := mtlsPatch(rayCluster, patches, initHeadExists, workerHeadExists)
	if err != nil {
		fmt.Errorf(err.Error())
		return patches, err
	}

	return patches, nil
}

func mtlsPatch(rayCluster *rayv1api.RayCluster, patches []patchOperation, initHeadExists bool, workerHeadExists bool) ([]patchOperation, error) {
	fmt.Printf("creating json patch for RayCluster initContainers")
	isLocalInteractive := annotationBoolVal(rayCluster, "sdk.codeflare.dev/local_interactive", false)
	initContainerHead := corev1.Container{}

	key_volumes := []corev1.VolumeMount{
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
	svcDomain := rayCluster.Name + "-head-svc." + rayCluster.Namespace + ".svc"
	secretName := `ca-secret-` + rayCluster.Name

	if !initHeadExists {
		if isLocalInteractive {
			domain, err := getDomainName()
			if err != nil {
				return nil, err
			}
			rayClientRoute := "rayclient-" + rayCluster.Name + "-" + rayCluster.Namespace + "." + domain
			initContainerHead = corev1.Container{
				Name:  "create-cert",
				Image: "quay.io/project-codeflare/ray:latest-py39-cu118",
				Command: []string{
					"sh",
					"-c",
					`cd /home/ray/workspace/tls && openssl req -nodes -newkey rsa:2048 -keyout server.key -out server.csr -subj '/CN=ray-head' && printf "authorityKeyIdentifier=keyid,issuer\nbasicConstraints=CA:FALSE\nsubjectAltName = @alt_names\n[alt_names]\nDNS.1 = 127.0.0.1\nDNS.2 = localhost\nDNS.3 = ${FQ_RAY_IP}\nDNS.4 = $(awk 'END{print $1}' /etc/hosts)\nDNS.5 = ` + rayClientRoute + `">./domain.ext && cp /home/ray/workspace/ca/* . && openssl x509 -req -CA ca.crt -CAkey ca.key -in server.csr -out server.crt -days 365 -CAcreateserial -extfile domain.ext`,
				},
				VolumeMounts: key_volumes,
			}
		} else {
			initContainerHead = corev1.Container{
				Name:  "create-cert",
				Image: "quay.io/project-codeflare/ray:latest-py39-cu118",
				Command: []string{
					"sh",
					"-c",
					`cd /home/ray/workspace/tls && openssl req -nodes -newkey rsa:2048 -keyout server.key -out server.csr -subj '/CN=ray-head' && printf "authorityKeyIdentifier=keyid,issuer\nbasicConstraints=CA:FALSE\nsubjectAltName = @alt_names\n[alt_names]\nDNS.1 = 127.0.0.1\nDNS.2 = localhost\nDNS.3 = ${FQ_RAY_IP}\nDNS.4 = $(awk 'END{print $1}' /etc/hosts)\nDNS.5 = ` + svcDomain + `">./domain.ext && cp /home/ray/workspace/ca/* . && openssl x509 -req -CA ca.crt -CAkey ca.key -in server.csr -out server.crt -days 365 -CAcreateserial -extfile domain.ext`,
				},
				VolumeMounts: key_volumes,
			}
		}

		headVolume := corev1.Volume{
			Name: "ca-vol",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secretName,
				},
			},
		}

		headVolume2 := corev1.Volume{
			Name: "server-cert",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}

		patches = append(patches, patchOperation{
			Op:   "add",
			Path: "/spec/headGroupSpec/template/spec/containers/0/env/-",
			Value: map[string]interface{}{
				"name": "MY_POD_IP",
				"valueFrom": map[string]interface{}{
					"fieldRef": map[string]interface{}{
						"fieldPath": "status.podIP",
					},
				},
			},
		}, patchOperation{
			Op:    "add",
			Path:  "/spec/headGroupSpec/template/spec/containers/0/env/-",
			Value: map[string]interface{}{"name": "RAY_USE_TLS", "value": "1"},
		}, patchOperation{
			Op:    "add",
			Path:  "/spec/headGroupSpec/template/spec/containers/0/env/-",
			Value: map[string]interface{}{"name": "RAY_TLS_SERVER_CERT", "value": "/home/ray/workspace/tls/server.crt"},
		}, patchOperation{
			Op:    "add",
			Path:  "/spec/headGroupSpec/template/spec/containers/0/env/-",
			Value: map[string]interface{}{"name": "RAY_TLS_SERVER_KEY", "value": "/home/ray/workspace/tls/server.key"},
		}, patchOperation{
			Op:    "add",
			Path:  "/spec/headGroupSpec/template/spec/containers/0/env/-",
			Value: map[string]interface{}{"name": "RAY_TLS_CA_CERT", "value": "/home/ray/workspace/tls/ca.crt"},
		})

		patches = append(patches, patchOperation{
			Op:    "add",
			Path:  "/spec/headGroupSpec/template/spec/volumes/-",
			Value: headVolume,
		})

		patches = append(patches, patchOperation{
			Op:    "add",
			Path:  "/spec/headGroupSpec/template/spec/volumes/-",
			Value: headVolume2,
		})
		// Append HeadGroup init container
		patches = append(patches, patchOperation{
			Op:    "replace",
			Path:  "/spec/headGroupSpec/template/spec/initContainers/0",
			Value: initContainerHead,
		})
	}

	if !workerHeadExists {
		initContainerWorker := corev1.Container{
			Name:  "create-cert",
			Image: "quay.io/project-codeflare/ray:latest-py39-cu118",
			Command: []string{
				"sh",
				"-c",
				`cd /home/ray/workspace/tls && openssl req -nodes -newkey rsa:2048 -keyout server.key -out server.csr -subj '/CN=ray-head' && printf "authorityKeyIdentifier=keyid,issuer\nbasicConstraints=CA:FALSE\nsubjectAltName = @alt_names\n[alt_names]\nDNS.1 = 127.0.0.1\nDNS.2 = localhost\nDNS.3 = ${FQ_RAY_IP}\nDNS.4 = $(awk 'END{print $1}' /etc/hosts)">./domain.ext && cp /home/ray/workspace/ca/* . && openssl x509 -req -CA ca.crt -CAkey ca.key -in server.csr -out server.crt -days 365 -CAcreateserial -extfile domain.ext`,
			},
			VolumeMounts: key_volumes,
		}
		// Append WorkerGroup init container
		patches = append(patches, patchOperation{
			Op:    "replace",
			Path:  "/spec/workerGroupSpecs/0/template/spec/initContainers/0",
			Value: initContainerWorker,
		})

		workerCaVolume := corev1.Volume{
			Name: "ca-vol",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: secretName,
				},
			},
		}

		workerServerVolume := corev1.Volume{
			Name: "server-cert",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}

		patches = append(patches, patchOperation{
			Op:   "add",
			Path: "/spec/workerGroupSpecs/0/template/spec/containers/0/env/-",
			Value: map[string]interface{}{
				"name": "MY_POD_IP",
				"valueFrom": map[string]interface{}{
					"fieldRef": map[string]interface{}{
						"fieldPath": "status.podIP",
					},
				},
			},
		}, patchOperation{
			Op:    "add",
			Path:  "/spec/workerGroupSpecs/0/template/spec/containers/0/env/-",
			Value: map[string]interface{}{"name": "RAY_USE_TLS", "value": "1"},
		}, patchOperation{
			Op:    "add",
			Path:  "/spec/workerGroupSpecs/0/template/spec/containers/0/env/-",
			Value: map[string]interface{}{"name": "RAY_TLS_SERVER_CERT", "value": "/home/ray/workspace/tls/server.crt"},
		}, patchOperation{
			Op:    "add",
			Path:  "/spec/workerGroupSpecs/0/template/spec/containers/0/env/-",
			Value: map[string]interface{}{"name": "RAY_TLS_SERVER_KEY", "value": "/home/ray/workspace/tls/server.key"},
		}, patchOperation{
			Op:    "add",
			Path:  "/spec/workerGroupSpecs/0/template/spec/containers/0/env/-",
			Value: map[string]interface{}{"name": "RAY_TLS_CA_CERT", "value": "/home/ray/workspace/tls/ca.crt"},
		})

		patches = append(patches, patchOperation{
			Op:    "add",
			Path:  "/spec/workerGroupSpecs/0/template/spec/volumes/-",
			Value: workerCaVolume,
		})

		patches = append(patches, patchOperation{
			Op:    "add",
			Path:  "/spec/workerGroupSpecs/0/template/spec/volumes/-",
			Value: workerServerVolume,
		})
	}

	return patches, nil
}

func annotationBoolVal(cluster *rayv1api.RayCluster, annotation string, defaultValue bool) bool {
	val, exists := cluster.ObjectMeta.Annotations[annotation]
	if !exists || val == "" {
		return defaultValue
	}
	boolVal, err := strconv.ParseBool(val)
	if err != nil {
		fmt.Errorf("Could not convert annotation value to bool", "annotation", annotation, "value", val)
		return defaultValue
	}
	return boolVal
}

func getDomainName() (string, error) {
	consoleRoute := &routev1.Route{}

	if err := k8Client.Get(context.TODO(), types.NamespacedName{Name: NameConsoleLink, Namespace: NamespaceConsoleLink}, consoleRoute); err != nil {
		return "error getting console route URL", err
	}
	domainIndex := strings.Index(consoleRoute.Spec.Host, ".")
	consoleLinkDomain := consoleRoute.Spec.Host[domainIndex+1:]
	return consoleLinkDomain, nil
}
