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
	"crypto/tls"
	"net/http"
	"net/url"
	"testing"

	. "github.com/onsi/gomega"
	. "github.com/project-codeflare/codeflare-common/support"
	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Trains the MNIST dataset as a RayJob, executed by a Ray cluster
// directly managed by Kueue, and asserts successful completion of the training job.
func TestMNISTRayJobRayCluster(t *testing.T) {
	test := With(t)

	// Create a namespace and localqueue in that namespace
	namespace := test.NewTestNamespace()
	localQueue := CreateKueueLocalQueue(test, namespace.Name, "e2e-cluster-queue")

	// Create MNIST training script
	mnist := constructMNISTConfigMap(test, namespace)
	mnist, err := test.Client().Core().CoreV1().ConfigMaps(namespace.Name).Create(test.Ctx(), mnist, metav1.CreateOptions{})
	test.Expect(err).NotTo(HaveOccurred())
	test.T().Logf("Created ConfigMap %s/%s successfully", mnist.Namespace, mnist.Name)

	// Create RayCluster and assign it to the localqueue
	rayCluster := constructRayCluster(test, namespace, mnist)
	AssignToLocalQueue(rayCluster, localQueue)
	rayCluster, err = test.Client().Ray().RayV1().RayClusters(namespace.Name).Create(test.Ctx(), rayCluster, metav1.CreateOptions{})
	test.Expect(err).NotTo(HaveOccurred())
	test.T().Logf("Created RayCluster %s/%s successfully", rayCluster.Namespace, rayCluster.Name)

	test.T().Logf("Waiting for RayCluster %s/%s to be running", rayCluster.Namespace, rayCluster.Name)
	test.Eventually(RayCluster(test, namespace.Name, rayCluster.Name), TestTimeoutMedium).
		Should(WithTransform(RayClusterState, Equal(rayv1.Ready)))

	// Create RayJob
	rayJob := constructRayJob(test, namespace, rayCluster)
	rayJob, err = test.Client().Ray().RayV1().RayJobs(namespace.Name).Create(test.Ctx(), rayJob, metav1.CreateOptions{})
	test.Expect(err).NotTo(HaveOccurred())
	test.T().Logf("Created RayJob %s/%s successfully", rayJob.Namespace, rayJob.Name)

	rayDashboardURL := getRayDashboardURL(test, rayCluster.Namespace, rayCluster.Name)

	test.T().Logf("Connecting to Ray cluster at: %s", rayDashboardURL.String())
	rayClient := NewRayClusterClient(rayDashboardURL)

	// Wait for Ray job id to be available, this value is needed for writing logs in defer
	test.Eventually(RayJob(test, rayJob.Namespace, rayJob.Name), TestTimeoutShort).
		Should(WithTransform(RayJobId, Not(BeEmpty())))

	// Retrieving the job logs once it has completed or timed out
	defer WriteRayJobAPILogs(test, rayClient, GetRayJobId(test, rayJob.Namespace, rayJob.Name))

	test.T().Logf("Waiting for RayJob %s/%s to complete", rayJob.Namespace, rayJob.Name)
	test.Eventually(RayJob(test, rayJob.Namespace, rayJob.Name), TestTimeoutLong).
		Should(WithTransform(RayJobStatus, Satisfy(rayv1.IsJobTerminal)))

	// Assert the Ray job has completed successfully
	test.Expect(GetRayJob(test, rayJob.Namespace, rayJob.Name)).
		To(WithTransform(RayJobStatus, Equal(rayv1.JobStatusSucceeded)))
}

func constructMNISTConfigMap(test Test, namespace *corev1.Namespace) *corev1.ConfigMap {
	return &corev1.ConfigMap{
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
}

func constructRayCluster(_ Test, namespace *corev1.Namespace, mnist *corev1.ConfigMap) *rayv1.RayCluster {
	return &rayv1.RayCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rayv1.GroupVersion.String(),
			Kind:       "RayCluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "raycluster",
			Namespace: namespace.Name,
		},
		Spec: rayv1.RayClusterSpec{
			RayVersion: GetRayVersion(),
			HeadGroupSpec: rayv1.HeadGroupSpec{
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
										corev1.ResourceMemory: resource.MustParse("2G"),
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
			WorkerGroupSpecs: []rayv1.WorkerGroupSpec{
				{
					Replicas:       Ptr(int32(1)),
					MinReplicas:    Ptr(int32(1)),
					MaxReplicas:    Ptr(int32(2)),
					GroupName:      "small-group",
					RayStartParams: map[string]string{},
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
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
											corev1.ResourceMemory: resource.MustParse("2G"),
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
}

func constructRayJob(_ Test, namespace *corev1.Namespace, rayCluster *rayv1.RayCluster) *rayv1.RayJob {
	return &rayv1.RayJob{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rayv1.GroupVersion.String(),
			Kind:       "RayJob",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mnist",
			Namespace: namespace.Name,
		},
		Spec: rayv1.RayJobSpec{
			Entrypoint: "python /home/ray/jobs/mnist.py",
			RuntimeEnvYAML: `
  pip:
    - pytorch_lightning==1.5.10
    - torchmetrics==0.9.1
    - torchvision==0.12.0
  env_vars:
    MNIST_DATASET_URL: "` + GetMnistDatasetURL() + `"
    PIP_INDEX_URL: "` + GetPipIndexURL() + `"
    PIP_TRUSTED_HOST: "` + GetPipTrustedHost() + `"
`,
			ClusterSelector: map[string]string{
				RayJobDefaultClusterSelectorKey: rayCluster.Name,
			},
			ShutdownAfterJobFinishes: false,
			SubmitterPodTemplate: &corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Image: GetRayImage(),
							Name:  "rayjob-submitter-pod",
						},
					},
				},
			},
		},
	}
}

func getRayDashboardURL(test Test, namespace, rayClusterName string) url.URL {
	dashboardName := "ray-dashboard-" + rayClusterName

	if IsOpenShift(test) {
		route := GetRoute(test, namespace, dashboardName)
		hostname := route.Status.Ingress[0].Host

		// Wait for expected HTTP code
		test.T().Logf("Waiting for Route %s/%s to be available", route.Namespace, route.Name)
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := &http.Client{Transport: tr}

		test.Eventually(func() (int, error) {
			resp, err := client.Get("https://" + hostname)
			if err != nil {
				return -1, err
			}
			return resp.StatusCode, nil
		}, TestTimeoutShort).Should(Not(Equal(503)))

		return url.URL{
			Scheme: "https",
			Host:   hostname,
		}
	}

	ingress := GetIngress(test, namespace, dashboardName)

	test.T().Logf("Waiting for Ingress %s/%s to be admitted", ingress.Namespace, ingress.Name)
	test.Eventually(Ingress(test, ingress.Namespace, ingress.Name), TestTimeoutShort).
		Should(WithTransform(LoadBalancerIngresses, HaveLen(1)))

	return url.URL{
		Scheme: "http",
		Host:   ingress.Spec.Rules[0].Host,
	}
}
