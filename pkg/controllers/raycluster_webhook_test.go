/*
Copyright 2024.

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
	"testing"

	. "github.com/onsi/gomega"
	"github.com/project-codeflare/codeflare-common/support"
	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/project-codeflare/codeflare-operator/pkg/config"
)

var (
	namespace      = "test-namespace"
	rayClusterName = "test-raycluster"

	rcWebhook = &rayClusterWebhook{
		Config: &config.KubeRayConfiguration{},
	}
)

func TestRayClusterWebhookDefault(t *testing.T) {
	test := support.NewTest(t)

	validRayCluster := &rayv1.RayCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rayClusterName,
			Namespace: namespace,
		},
		Spec: rayv1.RayClusterSpec{
			HeadGroupSpec: rayv1.HeadGroupSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{},
					},
				},
				RayStartParams: map[string]string{},
			},
			WorkerGroupSpecs: []rayv1.WorkerGroupSpec{
				{
					GroupName: "worker-group-1",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name: "worker-container-1",
								},
							},
						},
					},
					RayStartParams: map[string]string{},
				},
				{
					GroupName: "worker-group-2",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name: "worker-container-2",
								},
							},
						},
					},
					RayStartParams: map[string]string{},
				},
			},
		},
	}

	// Create the RayClusters
	if _, err := test.Client().Ray().RayV1().RayClusters(namespace).Create(test.Ctx(), validRayCluster, metav1.CreateOptions{}); err != nil {
		test.T().Fatalf("Failed to create RayCluster: %v", err)
	}

	// Call to default function is made
	err := rcWebhook.Default(test.Ctx(), runtime.Object(validRayCluster))
	t.Run("Expected no errors on call to Default function", func(t *testing.T) {
		test.Expect(err).ShouldNot(HaveOccurred(), "Expected no errors on call to Default function")
	})

	t.Run("Expected required OAuth proxy container for the head group", func(t *testing.T) {
		test.Expect(validRayCluster.Spec.HeadGroupSpec.Template.Spec.Containers).
			To(
				And(
					HaveLen(1),
					ContainElement(WithTransform(ContainerName, Equal(oauthProxyContainerName))),
				),
				"Expected the OAuth proxy container to be present in the head group",
			)
	})

	t.Run("Expected required OAuth proxy TLS secret and CA volumes for the head group", func(t *testing.T) {
		test.Expect(validRayCluster.Spec.HeadGroupSpec.Template.Spec.Volumes).
			To(
				And(
					HaveLen(3),
					ContainElement(WithTransform(VolumeName, Equal(oauthProxyVolumeName))),
					ContainElement(WithTransform(VolumeName, Equal("ca-vol"))),
					ContainElement(WithTransform(VolumeName, Equal("server-cert"))),
				),
				"Expected the OAuth proxy TLS secret and CA volumes to be present in the head group",
			)
	})

	t.Run("Expected required service account name for the head group", func(t *testing.T) {
		test.Expect(validRayCluster.Spec.HeadGroupSpec.Template.Spec.ServiceAccountName).
			To(Equal(validRayCluster.Name+"-oauth-proxy"),
				"Expected the service account name to be set correctly")
	})

	t.Run("Expected required environment variables for the head group", func(t *testing.T) {
		test.Expect(validRayCluster.Spec.HeadGroupSpec.Template.Spec.Containers[0].Env).
			To(
				And(
					HaveLen(6),
					ContainElement(WithTransform(EnvVarName, Equal("COOKIE_SECRET"))),
					ContainElement(WithTransform(EnvVarName, Equal("MY_POD_IP"))),
					ContainElement(WithTransform(EnvVarName, Equal("RAY_USE_TLS"))),
					ContainElement(WithTransform(EnvVarName, Equal("RAY_TLS_SERVER_CERT"))),
					ContainElement(WithTransform(EnvVarName, Equal("RAY_TLS_SERVER_KEY"))),
					ContainElement(WithTransform(EnvVarName, Equal("RAY_TLS_CA_CERT"))),
				),
				"Expected the required environment variables to be present in the head group",
			)
	})

	t.Run("Expected required create-cert init container for the head group", func(t *testing.T) {
		test.Expect(validRayCluster.Spec.HeadGroupSpec.Template.Spec.InitContainers).
			To(
				And(
					HaveLen(1),
					ContainElement(WithTransform(ContainerName, Equal(initContainerName))),
				),
				"Expected the create-cert init container to be present in the head group",
			)
	})

	t.Run("Expected required OAuth proxy TLS secret and CA volume mounts for the head group", func(t *testing.T) {
		test.Expect(validRayCluster.Spec.HeadGroupSpec.Template.Spec.Containers[0].VolumeMounts).
			To(
				And(
					HaveLen(3),
					ContainElement(WithTransform(VolumeMountName, Equal(oauthProxyVolumeName))),
					ContainElement(WithTransform(VolumeMountName, Equal("ca-vol"))),
					ContainElement(WithTransform(VolumeMountName, Equal("server-cert"))),
				),
				"Expected the OAuth proxy TLS secret and CA volume mounts to be present in the head group",
			)
	})

	t.Run("Expected required environment variables for each worker group", func(t *testing.T) {
		for _, workerGroup := range validRayCluster.Spec.WorkerGroupSpecs {
			test.Expect(workerGroup.Template.Spec.Containers[0].Env).
				To(
					And(
						HaveLen(5),
						ContainElement(WithTransform(EnvVarName, Equal("MY_POD_IP"))),
						ContainElement(WithTransform(EnvVarName, Equal("RAY_USE_TLS"))),
						ContainElement(WithTransform(EnvVarName, Equal("RAY_TLS_SERVER_CERT"))),
						ContainElement(WithTransform(EnvVarName, Equal("RAY_TLS_SERVER_KEY"))),
						ContainElement(WithTransform(EnvVarName, Equal("RAY_TLS_CA_CERT"))),
					),
					"Expected the required environment variables to be present in each worker group",
				)
		}
	})

	t.Run("Expected required CA Volumes for each worker group", func(t *testing.T) {
		for _, workerGroup := range validRayCluster.Spec.WorkerGroupSpecs {
			test.Expect(workerGroup.Template.Spec.Volumes).
				To(
					And(
						HaveLen(2),
						ContainElement(WithTransform(VolumeName, Equal("ca-vol"))),
						ContainElement(WithTransform(VolumeName, Equal("server-cert"))),
					),
					"Expected the required CA volumes to be present in each worker group",
				)
		}
	})

	t.Run("Expected required certificate volume mounts for each worker group", func(t *testing.T) {
		for _, workerSpec := range validRayCluster.Spec.WorkerGroupSpecs {
			test.Expect(workerSpec.Template.Spec.Containers[0].VolumeMounts).
				To(
					And(
						HaveLen(2),
						ContainElement(WithTransform(VolumeMountName, Equal("ca-vol"))),
						ContainElement(WithTransform(VolumeMountName, Equal("server-cert"))),
					),
					"Expected the required certificate volume mounts to be present in each worker group",
				)
		}
	})

	t.Run("Expected required init container for each worker group", func(t *testing.T) {
		for _, workerSpec := range validRayCluster.Spec.WorkerGroupSpecs {
			test.Expect(workerSpec.Template.Spec.InitContainers).
				To(
					And(
						HaveLen(1),
						ContainElement(WithTransform(ContainerName, Equal(initContainerName))),
					),
					"Expected the required init container to be present in each worker group",
				)
		}
	})

}

func TestValidateCreate(t *testing.T) {
	test := support.NewTest(t)

	validRayCluster := &rayv1.RayCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rayClusterName,
			Namespace: namespace,
		},
		Spec: rayv1.RayClusterSpec{
			HeadGroupSpec: rayv1.HeadGroupSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
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
													Name: rayClusterName + "-oauth-config",
												},
												Key: "cookie_secret",
											},
										},
									},
								},
								Args: []string{
									"--https-address=:8443",
									"--provider=openshift",
									"--openshift-service-account=" + rayClusterName + "-oauth-proxy",
									"--upstream=http://localhost:8265",
									"--tls-cert=/etc/tls/private/tls.crt",
									"--tls-key=/etc/tls/private/tls.key",
									"--cookie-secret=$(COOKIE_SECRET)",
									"--openshift-delegate-urls={\"/\":{\"resource\":\"pods\",\"namespace\":\"" + namespace + "\",\"verb\":\"get\"}}",
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      oauthProxyVolumeName,
										MountPath: "/etc/tls/private",
										ReadOnly:  true,
									},
								},
							},
						},
						Volumes: []corev1.Volume{
							{
								Name: oauthProxyVolumeName,
								VolumeSource: corev1.VolumeSource{
									Secret: &corev1.SecretVolumeSource{
										SecretName: rayClusterName + "-proxy-tls-secret",
									},
								},
							},
						},
						ServiceAccountName: rayClusterName + "-oauth-proxy",
					},
				},
				RayStartParams: map[string]string{},
			},
		},
	}

	// Create the RayClusters
	if _, err := test.Client().Ray().RayV1().RayClusters(namespace).Create(test.Ctx(), validRayCluster, metav1.CreateOptions{}); err != nil {
		test.T().Fatalf("Failed to create RayCluster: %v", err)
	}

	// Call to ValidateCreate function is made
	warnings, err := rcWebhook.ValidateCreate(test.Ctx(), runtime.Object(validRayCluster))
	t.Run("Expected no warnings or errors on call to ValidateCreate function", func(t *testing.T) {
		test.Expect(warnings).Should(BeNil(), "Expected no warnings on call to ValidateCreate function")
		test.Expect(err).ShouldNot(HaveOccurred(), "Expected no errors on call to ValidateCreate function")
	})

	// Negative Test: Invalid RayCluster with EnableIngress set to true
	trueBool := true
	invalidRayCluster := validRayCluster.DeepCopy()

	t.Run("Negative: Expected errors on call to ValidateCreate function due to EnableIngress set to True", func(t *testing.T) {
		invalidRayCluster.Spec.HeadGroupSpec.EnableIngress = &trueBool
		_, err := rcWebhook.ValidateCreate(test.Ctx(), runtime.Object(invalidRayCluster))
		test.Expect(err).Should(HaveOccurred(), "Expected errors on call to ValidateCreate function due to EnableIngress set to True")
	})

	t.Run("Negative: Expected errors on call to ValidateCreate function due to manipulated OAuth Proxy Container", func(t *testing.T) {
		for i, headContainer := range invalidRayCluster.Spec.HeadGroupSpec.Template.Spec.Containers {
			if headContainer.Name == oauthProxyContainerName {
				invalidRayCluster.Spec.HeadGroupSpec.Template.Spec.Containers[i].Args = []string{"--invalid-arg"}
				break
			}
		}
		_, err = rcWebhook.ValidateCreate(test.Ctx(), runtime.Object(invalidRayCluster))
		test.Expect(err).Should(HaveOccurred(), "Expected errors on call to ValidateCreate function due to manipulated OAuth Proxy Container")
	})

	t.Run("Negative: Expected errors on call to ValidateCreate function due to manipulated OAuth Proxy Volume", func(t *testing.T) {
		for i, headVolume := range invalidRayCluster.Spec.HeadGroupSpec.Template.Spec.Volumes {
			if headVolume.Name == oauthProxyVolumeName {
				invalidRayCluster.Spec.HeadGroupSpec.Template.Spec.Volumes[i].Secret.SecretName = "invalid-secret-name"
				break
			}
		}
		_, err = rcWebhook.ValidateCreate(test.Ctx(), runtime.Object(invalidRayCluster))
		test.Expect(err).Should(HaveOccurred(), "Expected errors on call to ValidateCreate function due to manipulated OAuth Proxy Volume")
	})

	t.Run("Negative: Expected errors on call to ValidateCreate function due to manipulated head group service account name", func(t *testing.T) {
		invalidRayCluster.Spec.HeadGroupSpec.Template.Spec.ServiceAccountName = "invalid-service-account-name"
		_, err = rcWebhook.ValidateCreate(test.Ctx(), runtime.Object(invalidRayCluster))
		test.Expect(err).Should(HaveOccurred(), "Expected errors on call to ValidateCreate function due to manipulated head group service account name")
	})

}

func TestValidateUpdate(t *testing.T) {
	test := support.NewTest(t)

	validRayCluster := &rayv1.RayCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rayClusterName,
			Namespace: namespace,
		},
		Spec: rayv1.RayClusterSpec{
			HeadGroupSpec: rayv1.HeadGroupSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
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
													Name: rayClusterName + "-oauth-config",
												},
												Key: "cookie_secret",
											},
										},
									},
									{
										Name: "MY_POD_IP",
										ValueFrom: &corev1.EnvVarSource{
											FieldRef: &corev1.ObjectFieldSelector{
												FieldPath: "status.podIP",
											},
										},
									},
									{Name: "RAY_USE_TLS", Value: "1"},
									{Name: "RAY_TLS_SERVER_CERT", Value: "/home/ray/workspace/tls/server.crt"},
									{Name: "RAY_TLS_SERVER_KEY", Value: "/home/ray/workspace/tls/server.key"},
									{Name: "RAY_TLS_CA_CERT", Value: "/home/ray/workspace/tls/ca.crt"},
								},
								Args: []string{
									"--https-address=:8443",
									"--provider=openshift",
									"--openshift-service-account=" + rayClusterName + "-oauth-proxy",
									"--upstream=http://localhost:8265",
									"--tls-cert=/etc/tls/private/tls.crt",
									"--tls-key=/etc/tls/private/tls.key",
									"--cookie-secret=$(COOKIE_SECRET)",
									"--openshift-delegate-urls={\"/\":{\"resource\":\"pods\",\"namespace\":\"" + namespace + "\",\"verb\":\"get\"}}",
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      oauthProxyVolumeName,
										MountPath: "/etc/tls/private",
										ReadOnly:  true,
									},
								},
							},
						},
						InitContainers: []corev1.Container{
							{
								Name:  "create-cert",
								Image: "",
								Command: []string{
									"sh",
									"-c",
									`cd /home/ray/workspace/tls && openssl req -nodes -newkey rsa:2048 -keyout server.key -out server.csr -subj '/CN=ray-head' && printf "authorityKeyIdentifier=keyid,issuer\nbasicConstraints=CA:FALSE\nsubjectAltName = @alt_names\n[alt_names]\nDNS.1 = 127.0.0.1\nDNS.2 = localhost\nDNS.3 = ${FQ_RAY_IP}\nDNS.4 = $(awk 'END{print $1}' /etc/hosts)\nDNS.5 = rayclient-` + rayClusterName + `-` + namespace + `.\nDNS.6 = ` + rayClusterName + `-head-svc.` + namespace + `.svc` + `">./domain.ext && cp /home/ray/workspace/ca/* . && openssl x509 -req -CA ca.crt -CAkey ca.key -in server.csr -out server.crt -days 365 -CAcreateserial -extfile domain.ext`,
								},
								VolumeMounts: []corev1.VolumeMount{
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
								},
							},
						},
						Volumes: []corev1.Volume{
							{
								Name: oauthProxyVolumeName,
								VolumeSource: corev1.VolumeSource{
									Secret: &corev1.SecretVolumeSource{
										SecretName: rayClusterName + "-proxy-tls-secret",
									},
								},
							},
							{
								Name: "ca-vol",
								VolumeSource: corev1.VolumeSource{
									Secret: &corev1.SecretVolumeSource{
										SecretName: `ca-secret-` + rayClusterName,
									},
								},
							},
							{
								Name: "server-cert",
								VolumeSource: corev1.VolumeSource{
									EmptyDir: &corev1.EmptyDirVolumeSource{},
								},
							},
						},
						ServiceAccountName: rayClusterName + "-oauth-proxy",
					},
				},
				RayStartParams: map[string]string{},
			},
			WorkerGroupSpecs: []rayv1.WorkerGroupSpec{
				{
					GroupName: "worker-group-1",
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name: "worker-container-1",
									Env: []corev1.EnvVar{
										{
											Name: "MY_POD_IP",
											ValueFrom: &corev1.EnvVarSource{
												FieldRef: &corev1.ObjectFieldSelector{
													FieldPath: "status.podIP",
												},
											},
										},
										{Name: "RAY_USE_TLS", Value: "1"},
										{Name: "RAY_TLS_SERVER_CERT", Value: "/home/ray/workspace/tls/server.crt"},
										{Name: "RAY_TLS_SERVER_KEY", Value: "/home/ray/workspace/tls/server.key"},
										{Name: "RAY_TLS_CA_CERT", Value: "/home/ray/workspace/tls/ca.crt"},
									},
								},
							},
							InitContainers: []corev1.Container{
								{
									Name:  "create-cert",
									Image: "",
									Command: []string{
										"sh",
										"-c",
										`cd /home/ray/workspace/tls && openssl req -nodes -newkey rsa:2048 -keyout server.key -out server.csr -subj '/CN=ray-head' && printf "authorityKeyIdentifier=keyid,issuer\nbasicConstraints=CA:FALSE\nsubjectAltName = @alt_names\n[alt_names]\nDNS.1 = 127.0.0.1\nDNS.2 = localhost\nDNS.3 = ${FQ_RAY_IP}\nDNS.4 = $(awk 'END{print $1}' /etc/hosts)">./domain.ext && cp /home/ray/workspace/ca/* . && openssl x509 -req -CA ca.crt -CAkey ca.key -in server.csr -out server.crt -days 365 -CAcreateserial -extfile domain.ext`,
									},
									VolumeMounts: certVolumeMounts(),
								},
							},
							Volumes: []corev1.Volume{
								{
									Name: "ca-vol",
									VolumeSource: corev1.VolumeSource{
										Secret: &corev1.SecretVolumeSource{
											SecretName: `ca-secret-` + rayClusterName,
										},
									},
								},
								{
									Name: "server-cert",
									VolumeSource: corev1.VolumeSource{
										EmptyDir: &corev1.EmptyDirVolumeSource{},
									},
								},
							},
						},
					},
					RayStartParams: map[string]string{},
				},
			},
		},
	}

	// Create the RayClusters
	if _, err := test.Client().Ray().RayV1().RayClusters(namespace).Create(test.Ctx(), validRayCluster, metav1.CreateOptions{}); err != nil {
		test.T().Fatalf("Failed to create RayCluster: %v", err)
	}

	// Positive Test Case: Valid RayCluster with immutable fields
	t.Run("Expected no warnings or errors on call to ValidateUpdate function", func(t *testing.T) {
		warnings, err := rcWebhook.ValidateUpdate(test.Ctx(), runtime.Object(validRayCluster), runtime.Object(validRayCluster))
		test.Expect(warnings).Should(BeNil(), "Expected no warnings on call to ValidateUpdate function")
		test.Expect(err).ShouldNot(HaveOccurred(), "Expected no errors on call to ValidateUpdate function")
	})

	// Negative Test Cases
	trueBool := true
	invalidRayCluster := validRayCluster.DeepCopy()

	t.Run("Negative: Expected errors on call to ValidateUpdate function due to EnableIngress set to True", func(t *testing.T) {
		invalidRayCluster.Spec.HeadGroupSpec.EnableIngress = &trueBool
		_, err := rcWebhook.ValidateUpdate(test.Ctx(), runtime.Object(validRayCluster), runtime.Object(invalidRayCluster))
		test.Expect(err).Should(HaveOccurred(), "Expected errors on call to ValidateUpdate function due to EnableIngress set to True")
	})

	t.Run("Negative: Expected errors on call to ValidateUpdate function due to manipulated OAuth Proxy Container", func(t *testing.T) {
		for i, headContainer := range invalidRayCluster.Spec.HeadGroupSpec.Template.Spec.Containers {
			if headContainer.Name == oauthProxyContainerName {
				invalidRayCluster.Spec.HeadGroupSpec.Template.Spec.Containers[i].Args = []string{"--invalid-arg"}
				break
			}
		}
		_, err := rcWebhook.ValidateUpdate(test.Ctx(), runtime.Object(validRayCluster), runtime.Object(invalidRayCluster))
		test.Expect(err).Should(HaveOccurred(), "Expected errors on call to ValidateUpdate function due to manipulated OAuth Proxy Container")
	})

	t.Run("Negative: Expected errors on call to ValidateUpdate function due to manipulated OAuth Proxy Volume", func(t *testing.T) {
		for i, headVolume := range invalidRayCluster.Spec.HeadGroupSpec.Template.Spec.Volumes {
			if headVolume.Name == oauthProxyVolumeName {
				invalidRayCluster.Spec.HeadGroupSpec.Template.Spec.Volumes[i].Secret.SecretName = "invalid-secret-name"
				break
			}
		}
		_, err := rcWebhook.ValidateUpdate(test.Ctx(), runtime.Object(validRayCluster), runtime.Object(invalidRayCluster))
		test.Expect(err).Should(HaveOccurred(), "Expected errors on call to ValidateUpdate function due to manipulated OAuth Proxy Volume")
	})

	t.Run("Negative: Expected errors on call to ValidateUpdate function due to manipulated head group service account name", func(t *testing.T) {
		invalidRayCluster.Spec.HeadGroupSpec.Template.Spec.ServiceAccountName = "invalid-service-account-name"
		_, err := rcWebhook.ValidateUpdate(test.Ctx(), runtime.Object(validRayCluster), runtime.Object(invalidRayCluster))
		test.Expect(err).Should(HaveOccurred(), "Expected errors on call to ValidateUpdate function due to manipulated head group service account name")
	})

	t.Run("Negative: Expected errors on call to ValidateUpdate function due to manipulated Init Container in the head group", func(t *testing.T) {
		for i, headInitContainer := range invalidRayCluster.Spec.HeadGroupSpec.Template.Spec.InitContainers {
			if headInitContainer.Name == "create-cert" {
				invalidRayCluster.Spec.HeadGroupSpec.Template.Spec.InitContainers[i].Command = []string{"manipulated command"}
				break
			}
		}
		_, err := rcWebhook.ValidateUpdate(test.Ctx(), runtime.Object(validRayCluster), runtime.Object(invalidRayCluster))
		test.Expect(err).Should(HaveOccurred(), "Expected errors on call to ValidateUpdate function due to manipulated Init Container in the head group")
	})

	t.Run("Negative: Expected errors on call to ValidateUpdate function due to manipulated Init Container in the worker group", func(t *testing.T) {
		for _, workerGroup := range invalidRayCluster.Spec.WorkerGroupSpecs {
			for i, workerInitContainer := range workerGroup.Template.Spec.InitContainers {
				if workerInitContainer.Name == "create-cert" {
					workerGroup.Template.Spec.InitContainers[i].Command = []string{"manipulated command"}
					break
				}
			}
		}
		_, err := rcWebhook.ValidateUpdate(test.Ctx(), runtime.Object(validRayCluster), runtime.Object(invalidRayCluster))
		test.Expect(err).Should(HaveOccurred(), "Expected errors on call to ValidateUpdate function due to manipulated Init Container in the worker group")
	})

	t.Run("Negative: Expected errors on call to ValidateUpdate function due to manipulated Volume in the head group", func(t *testing.T) {
		for i, headVolume := range invalidRayCluster.Spec.HeadGroupSpec.Template.Spec.Volumes {
			if headVolume.Name == "ca-vol" {
				invalidRayCluster.Spec.HeadGroupSpec.Template.Spec.Volumes[i].Secret.SecretName = "invalid-secret-name"
				break
			}
		}
		_, err := rcWebhook.ValidateUpdate(test.Ctx(), runtime.Object(validRayCluster), runtime.Object(invalidRayCluster))
		test.Expect(err).Should(HaveOccurred(), "Expected errors on call to ValidateUpdate function due to manipulated Volume in the head group")
	})

	t.Run("Negative: Expected errors on call to ValidateUpdate function due to manipulated Volume in the worker group", func(t *testing.T) {
		for _, workerGroup := range invalidRayCluster.Spec.WorkerGroupSpecs {
			for i, workerVolume := range workerGroup.Template.Spec.Volumes {
				if workerVolume.Name == "ca-vol" {
					workerGroup.Template.Spec.Volumes[i].Secret.SecretName = "invalid-secret-name"
					break
				}
			}
		}
		_, err := rcWebhook.ValidateUpdate(test.Ctx(), runtime.Object(validRayCluster), runtime.Object(invalidRayCluster))
		test.Expect(err).Should(HaveOccurred(), "Expected errors on call to ValidateUpdate function due to manipulated Volume in the worker group")
	})

	t.Run("Negative: Expected errors on call to ValidateUpdate function due to manipulated env vars in the head group", func(t *testing.T) {
		for i, headEnvVar := range invalidRayCluster.Spec.HeadGroupSpec.Template.Spec.Containers[0].Env {
			if headEnvVar.Name == "RAY_USE_TLS" {
				invalidRayCluster.Spec.HeadGroupSpec.Template.Spec.Containers[0].Env[i].Value = "invalid-value"
				break
			}
		}
		_, err := rcWebhook.ValidateUpdate(test.Ctx(), runtime.Object(validRayCluster), runtime.Object(invalidRayCluster))
		test.Expect(err).Should(HaveOccurred(), "Expected errors on call to ValidateUpdate function due to manipulated env vars in the head group")
	})

	t.Run("Negative: Expected errors on call to ValidateUpdate function due to manipulated env vars in the worker group", func(t *testing.T) {
		for _, workerGroup := range invalidRayCluster.Spec.WorkerGroupSpecs {
			for i, workerEnvVar := range workerGroup.Template.Spec.Containers[0].Env {
				if workerEnvVar.Name == "RAY_USE_TLS" {
					workerGroup.Template.Spec.Containers[0].Env[i].Value = "invalid-value"
					break
				}
			}
		}
		_, err := rcWebhook.ValidateUpdate(test.Ctx(), runtime.Object(validRayCluster), runtime.Object(invalidRayCluster))
		test.Expect(err).Should(HaveOccurred(), "Expected errors on call to ValidateUpdate function due to manipulated env vars in the worker group")
	})
}

func ContainerName(container corev1.Container) string {
	return container.Name
}

func VolumeName(volume corev1.Volume) string {
	return volume.Name
}

func ServiceAccountName(serviceAccount corev1.ServiceAccount) string {
	return serviceAccount.Name
}

func VolumeMountName(volumeMount corev1.VolumeMount) string {
	return volumeMount.Name
}

func EnvVarName(envVar corev1.EnvVar) string {
	return envVar.Name
}
