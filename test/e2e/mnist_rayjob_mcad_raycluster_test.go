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

package e2e

import (
	"encoding/base64"
	"net/url"
	"testing"

	. "github.com/onsi/gomega"
	mcadv1beta1 "github.com/project-codeflare/multi-cluster-app-dispatcher/pkg/apis/controller/v1beta1"
	rayv1alpha1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	routev1 "github.com/openshift/api/route/v1"

	. "github.com/project-codeflare/codeflare-operator/test/support"
)

// Trains the MNIST dataset as a RayJob, executed by a Ray cluster managed by MCAD,
// and asserts successful completion of the training job.
func TestMNISTRayJobMCADRayCluster(t *testing.T) {
	test := With(t)
	test.T().Parallel()

	// Create a namespace
	namespace := test.NewTestNamespace()

	// MNIST training script
	mnist := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mnist",
			Namespace: namespace.Name,
		},
		BinaryData: map[string][]byte{
			"mnist.py": ReadFile(test, "mnist.py"),
		},
		Immutable: Ptr(true),
	}
	mnist, err := test.Client().Core().CoreV1().ConfigMaps(namespace.Name).Create(test.Ctx(), mnist, metav1.CreateOptions{})
	test.Expect(err).NotTo(HaveOccurred())
	test.T().Logf("Created ConfigMap %s/%s successfully", mnist.Namespace, mnist.Name)

	// RayCluster
	rayCluster := &rayv1alpha1.RayCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rayv1alpha1.GroupVersion.String(),
			Kind:       "RayCluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "raycluster",
			Namespace: namespace.Name,
		},
		Spec: rayv1alpha1.RayClusterSpec{
			RayVersion: GetRayVersion(),
			HeadGroupSpec: rayv1alpha1.HeadGroupSpec{
				RayStartParams: map[string]string{
					"dashboard-host": "0.0.0.0",
				},
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "ray-head",
								Image: GetRayImage(),
								Ports: []corev1.ContainerPort{
									{
										ContainerPort: 6379,
										Name:          "gcs",
									},
									{
										ContainerPort: 8265,
										Name:          "dashboard",
									},
									{
										ContainerPort: 10001,
										Name:          "client",
									},
								},
								Lifecycle: &corev1.Lifecycle{
									PreStop: &corev1.LifecycleHandler{
										Exec: &corev1.ExecAction{
											Command: []string{"/bin/sh", "-c", "ray stop"},
										},
									},
								},
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("250m"),
										corev1.ResourceMemory: resource.MustParse("512Mi"),
									},
									Limits: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("1"),
										corev1.ResourceMemory: resource.MustParse("1G"),
									},
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "mnist",
										MountPath: "/home/ray/jobs",
									},
								},
							},
						},
						Volumes: []corev1.Volume{
							{
								Name: "mnist",
								VolumeSource: corev1.VolumeSource{
									ConfigMap: &corev1.ConfigMapVolumeSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: mnist.Name,
										},
									},
								},
							},
						},
					},
				},
			},
			WorkerGroupSpecs: []rayv1alpha1.WorkerGroupSpec{
				{
					Replicas:       Ptr(int32(1)),
					MinReplicas:    Ptr(int32(1)),
					MaxReplicas:    Ptr(int32(2)),
					GroupName:      "small-group",
					RayStartParams: map[string]string{},
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							InitContainers: []corev1.Container{
								{
									Name:    "init-myservice",
									Image:   "busybox:1.28",
									Command: []string{"sh", "-c", "until nslookup $RAY_IP.$(cat /var/run/secrets/kubernetes.io/serviceaccount/namespace).svc.cluster.local; do echo waiting for myservice; sleep 2; done"},
								},
							},
							Containers: []corev1.Container{
								{
									Name:  "ray-worker",
									Image: GetRayImage(),
									Lifecycle: &corev1.Lifecycle{
										PreStop: &corev1.LifecycleHandler{
											Exec: &corev1.ExecAction{
												Command: []string{"/bin/sh", "-c", "ray stop"},
											},
										},
									},
									Resources: corev1.ResourceRequirements{
										Requests: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("250m"),
											corev1.ResourceMemory: resource.MustParse("256Mi"),
										},
										Limits: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("1"),
											corev1.ResourceMemory: resource.MustParse("512Mi"),
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Create an AppWrapper resource
	aw := &mcadv1beta1.AppWrapper{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ray-cluster",
			Namespace: namespace.Name,
		},
		Spec: mcadv1beta1.AppWrapperSpec{
			AggrResources: mcadv1beta1.AppWrapperResourceList{
				GenericItems: []mcadv1beta1.AppWrapperGenericResource{
					{
						DesiredAvailable: 1,
						CustomPodResources: []mcadv1beta1.CustomPodResourceTemplate{
							{
								Replicas: 2,
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("250m"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("1G"),
								},
							},
						},
						GenericTemplate: Raw(test, rayCluster),
					},
				},
			},
		},
	}

	aw, err = test.Client().MCAD().McadV1beta1().AppWrappers(namespace.Name).Create(test.Ctx(), aw, metav1.CreateOptions{})
	test.Expect(err).NotTo(HaveOccurred())
	test.T().Logf("Created MCAD %s/%s successfully", aw.Namespace, aw.Name)

	test.T().Logf("Waiting for MCAD %s/%s to be running", aw.Namespace, aw.Name)
	test.Eventually(AppWrapper(test, namespace, aw.Name), TestTimeoutMedium).
		Should(WithTransform(AppWrapperState, Equal(mcadv1beta1.AppWrapperStateActive)))

	rayJob := &rayv1alpha1.RayJob{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rayv1alpha1.GroupVersion.String(),
			Kind:       "RayJob",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mnist",
			Namespace: namespace.Name,
		},
		Spec: rayv1alpha1.RayJobSpec{
			Entrypoint: "python /home/ray/jobs/mnist.py",
			RuntimeEnv: base64.StdEncoding.EncodeToString([]byte(`
{
  "pip": [
    "pytorch_lightning==1.5.10",
    "torchmetrics==0.9.1",
    "torchvision==0.12.0"
  ],
  "env_vars": {
  }
}
`)),
			ClusterSelector: map[string]string{
				RayJobDefaultClusterSelectorKey: rayCluster.Name,
			},
			ShutdownAfterJobFinishes: false,
		},
	}
	rayJob, err = test.Client().Ray().RayV1alpha1().RayJobs(namespace.Name).Create(test.Ctx(), rayJob, metav1.CreateOptions{})
	test.Expect(err).NotTo(HaveOccurred())
	test.T().Logf("Created RayJob %s/%s successfully", rayJob.Namespace, rayJob.Name)

	var rayDashboardURL url.URL
	if IsOpenShift(test) {
		// Create a route to expose the Ray cluster API
		route := &routev1.Route{
			TypeMeta: metav1.TypeMeta{
				APIVersion: routev1.GroupVersion.String(),
				Kind:       "Route",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace.Name,
				Name:      "ray-dashboard",
			},
			Spec: routev1.RouteSpec{
				To: routev1.RouteTargetReference{
					Name: "raycluster-head-svc",
				},
				Port: &routev1.RoutePort{
					TargetPort: intstr.FromString("dashboard"),
				},
			},
		}

		_, err := test.Client().Route().RouteV1().Routes(namespace.Name).Create(test.Ctx(), route, metav1.CreateOptions{})
		test.Expect(err).NotTo(HaveOccurred())
		test.T().Logf("Created Route %s/%s successfully", route.Namespace, route.Name)

		test.T().Logf("Waiting for Route %s/%s to be admitted", route.Namespace, route.Name)
		test.Eventually(Route(test, route.Namespace, route.Name), TestTimeoutMedium).
			Should(WithTransform(ConditionStatus(routev1.RouteAdmitted), Equal(corev1.ConditionTrue)))

		route = GetRoute(test, route.Namespace, route.Name)

		rayDashboardURL = url.URL{
			Scheme: "http",
			Host:   route.Status.Ingress[0].Host,
		}
	} else {
		ingress := &networkingv1.Ingress{
			TypeMeta: metav1.TypeMeta{
				APIVersion: networkingv1.SchemeGroupVersion.String(),
				Kind:       "Ingress",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace.Name,
				Name:      "ray-dashboard",
				Annotations: map[string]string{
					"nginx.ingress.kubernetes.io/use-regex":      "true",
					"nginx.ingress.kubernetes.io/rewrite-target": "/$2",
				},
			},
			Spec: networkingv1.IngressSpec{
				Rules: []networkingv1.IngressRule{
					{
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{
									{
										Path:     "/ray-dashboard(/|$)(.*)",
										PathType: Ptr(networkingv1.PathTypePrefix),
										Backend: networkingv1.IngressBackend{
											Service: &networkingv1.IngressServiceBackend{
												Name: "raycluster-head-svc",
												Port: networkingv1.ServiceBackendPort{
													Name: "dashboard",
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}

		_, err := test.Client().Core().NetworkingV1().Ingresses(ingress.Namespace).Create(test.Ctx(), ingress, metav1.CreateOptions{})
		test.Expect(err).NotTo(HaveOccurred())
		test.T().Logf("Created Ingress %s/%s successfully", ingress.Namespace, ingress.Name)

		test.T().Logf("Waiting for Ingress %s/%s to be admitted", ingress.Namespace, ingress.Name)
		test.Eventually(Ingress(test, ingress.Namespace, ingress.Name), TestTimeoutMedium).
			Should(WithTransform(LoadBalancerIngresses, HaveLen(1)))

		ingress = GetIngress(test, ingress.Namespace, ingress.Name)

		rayDashboardURL = url.URL{
			Scheme: "http",
			Host:   ingress.Status.LoadBalancer.Ingress[0].IP,
			Path:   "ray-dashboard",
		}
	}

	test.T().Logf("Connecting to Ray cluster at: %s", rayDashboardURL.String())
	rayClient := NewRayClusterClient(rayDashboardURL)

	// Retrieving the job logs once it has completed or timed out
	defer WriteRayJobAPILogs(test, rayClient, GetRayJobId(test, rayJob.Namespace, rayJob.Name))

	test.T().Logf("Waiting for RayJob %s/%s to complete", rayJob.Namespace, rayJob.Name)
	test.Eventually(RayJob(test, rayJob.Namespace, rayJob.Name), TestTimeoutLong).
		Should(WithTransform(RayJobStatus, Satisfy(rayv1alpha1.IsJobTerminal)))

	// Assert the Ray job has completed successfully
	test.Expect(GetRayJob(test, rayJob.Namespace, rayJob.Name)).
		To(WithTransform(RayJobStatus, Equal(rayv1alpha1.JobStatusSucceeded)))
}
