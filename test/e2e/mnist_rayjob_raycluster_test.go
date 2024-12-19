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
	"fmt"
	"net/http"
	"net/url"
	"testing"

	. "github.com/onsi/gomega"
	mcadv1beta2 "github.com/project-codeflare/appwrapper/api/v1beta2"
	. "github.com/project-codeflare/codeflare-common/support"
	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/kueue/apis/kueue/v1beta1"
)

// Trains the MNIST dataset as a RayJob, executed by a Ray cluster
// directly managed by Kueue, and asserts successful completion of the training job.

func TestMnistRayJobRayClusterCpu(t *testing.T) {
	runMnistRayJobRayCluster(t, "cpu", 0)
}

func TestMnistRayJobRayClusterGpu(t *testing.T) {
	runMnistRayJobRayCluster(t, "gpu", 1)
}

func runMnistRayJobRayCluster(t *testing.T, accelerator string, numberOfGpus int) {
	test := With(t)

	// Create a namespace
	namespace := test.NewTestNamespace()

	// Create Kueue resources
	resourceFlavor := CreateKueueResourceFlavor(test, v1beta1.ResourceFlavorSpec{})
	defer func() {
		_ = test.Client().Kueue().KueueV1beta1().ResourceFlavors().Delete(test.Ctx(), resourceFlavor.Name, metav1.DeleteOptions{})
	}()
	clusterQueue := createClusterQueue(test, resourceFlavor, numberOfGpus)
	defer func() {
		_ = test.Client().Kueue().KueueV1beta1().ClusterQueues().Delete(test.Ctx(), clusterQueue.Name, metav1.DeleteOptions{})
	}()
	CreateKueueLocalQueue(test, namespace.Name, clusterQueue.Name, AsDefaultQueue)

	// Create MNIST training script
	mnist := constructMNISTConfigMap(test, namespace)
	mnist, err := test.Client().Core().CoreV1().ConfigMaps(namespace.Name).Create(test.Ctx(), mnist, metav1.CreateOptions{})
	test.Expect(err).NotTo(HaveOccurred())
	test.T().Logf("Created ConfigMap %s/%s successfully", mnist.Namespace, mnist.Name)

	// Create RayCluster and assign it to the localqueue
	rayCluster := constructRayCluster(test, namespace, mnist, numberOfGpus)
	rayCluster, err = test.Client().Ray().RayV1().RayClusters(namespace.Name).Create(test.Ctx(), rayCluster, metav1.CreateOptions{})
	test.Expect(err).NotTo(HaveOccurred())
	test.T().Logf("Created RayCluster %s/%s successfully", rayCluster.Namespace, rayCluster.Name)

	test.T().Logf("Waiting for RayCluster %s/%s to be running", rayCluster.Namespace, rayCluster.Name)
	test.Eventually(RayCluster(test, namespace.Name, rayCluster.Name), TestTimeoutMedium).
		Should(WithTransform(RayClusterState, Equal(rayv1.Ready)))

	// Create RayJob
	rayJob := constructRayJob(test, namespace, rayCluster, accelerator, numberOfGpus)
	rayJob, err = test.Client().Ray().RayV1().RayJobs(namespace.Name).Create(test.Ctx(), rayJob, metav1.CreateOptions{})
	test.Expect(err).NotTo(HaveOccurred())
	test.T().Logf("Created RayJob %s/%s successfully", rayJob.Namespace, rayJob.Name)

	rayDashboardURL := getRayDashboardURL(test, rayCluster.Namespace, rayCluster.Name)

	test.T().Logf("Connecting to Ray cluster at: %s", rayDashboardURL)
	rayClient := GetRayClusterClient(test, rayDashboardURL, test.Config().BearerToken)

	test.T().Logf("Waiting for RayJob %s/%s to complete", rayJob.Namespace, rayJob.Name)
	test.Eventually(RayJob(test, rayJob.Namespace, rayJob.Name), TestTimeoutLong).
		Should(WithTransform(RayJobStatus, Satisfy(rayv1.IsJobTerminal)))

	// Assert the Ray job has completed successfully
	test.Expect(GetRayJob(test, rayJob.Namespace, rayJob.Name)).
		To(WithTransform(RayJobStatus, Equal(rayv1.JobStatusSucceeded)))

	WriteRayJobAPILogs(test, rayClient, GetRayJobId(test, rayJob.Namespace, rayJob.Name))

	test.T().Logf("Deleting RayCluster %s/%s", rayCluster.Namespace, rayCluster.Name)
	err = test.Client().Ray().RayV1().RayClusters(namespace.Name).Delete(test.Ctx(), rayCluster.Name, metav1.DeleteOptions{})
	test.Expect(err).NotTo(HaveOccurred())

	test.T().Logf("Waiting for RayCluster %s/%s to be deleted", rayCluster.Namespace, rayCluster.Name)
	test.Eventually(RayClusters(test, namespace.Name), TestTimeoutShort).Should(BeEmpty())
}

func TestMnistRayJobRayClusterAppWrapperCpu(t *testing.T) {
	runMnistRayJobRayClusterAppWrapper(t, "cpu", 0)
}

func TestMnistRayJobRayClusterAppWrapperGpu(t *testing.T) {
	runMnistRayJobRayClusterAppWrapper(t, "gpu", 1)
}

// Same as TestMNISTRayJobRayCluster, except the RayCluster is wrapped in an AppWrapper
func runMnistRayJobRayClusterAppWrapper(t *testing.T, accelerator string, numberOfGpus int) {
	test := With(t)

	// Create a namespace
	namespace := test.NewTestNamespace()

	// Create Kueue resources
	resourceFlavor := CreateKueueResourceFlavor(test, v1beta1.ResourceFlavorSpec{})
	defer func() {
		_ = test.Client().Kueue().KueueV1beta1().ResourceFlavors().Delete(test.Ctx(), resourceFlavor.Name, metav1.DeleteOptions{})
	}()
	clusterQueue := createClusterQueue(test, resourceFlavor, numberOfGpus)
	defer func() {
		_ = test.Client().Kueue().KueueV1beta1().ClusterQueues().Delete(test.Ctx(), clusterQueue.Name, metav1.DeleteOptions{})
	}()
	localQueue := CreateKueueLocalQueue(test, namespace.Name, clusterQueue.Name, AsDefaultQueue)

	// Create MNIST training script
	mnist := constructMNISTConfigMap(test, namespace)
	mnist, err := test.Client().Core().CoreV1().ConfigMaps(namespace.Name).Create(test.Ctx(), mnist, metav1.CreateOptions{})
	test.Expect(err).NotTo(HaveOccurred())
	test.T().Logf("Created ConfigMap %s/%s successfully", mnist.Namespace, mnist.Name)

	// Create RayCluster, wrap in AppWrapper and assign to localqueue
	rayCluster := constructRayCluster(test, namespace, mnist, numberOfGpus)
	raw := Raw(test, rayCluster)
	raw = RemoveCreationTimestamp(test, raw)

	aw := &mcadv1beta2.AppWrapper{
		TypeMeta: metav1.TypeMeta{
			APIVersion: mcadv1beta2.GroupVersion.String(),
			Kind:       "AppWrapper",
		},
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: rayCluster.Name,
			Namespace:    namespace.Name,
			Labels:       map[string]string{"kueue.x-k8s.io/queue-name": localQueue.Name},
		},
		Spec: mcadv1beta2.AppWrapperSpec{
			Components: []mcadv1beta2.AppWrapperComponent{
				{
					Template: raw,
				},
			},
		},
	}
	appWrapperResource := mcadv1beta2.GroupVersion.WithResource("appwrappers")
	awMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(aw)
	test.Expect(err).NotTo(HaveOccurred())
	unstruct := unstructured.Unstructured{Object: awMap}
	unstructp, err := test.Client().Dynamic().Resource(appWrapperResource).Namespace(namespace.Name).Create(test.Ctx(), &unstruct, metav1.CreateOptions{})
	test.Expect(err).NotTo(HaveOccurred())
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructp.Object, aw)
	test.Expect(err).NotTo(HaveOccurred())
	test.T().Logf("Created AppWrapper %s/%s successfully", aw.Namespace, aw.Name)

	test.T().Logf("Waiting for AppWrapper %s/%s to be running", aw.Namespace, aw.Name)
	test.Eventually(AppWrappers(test, namespace), TestTimeoutMedium).
		Should(ContainElement(WithTransform(AppWrapperPhase, Equal(mcadv1beta2.AppWrapperRunning))))

	test.T().Logf("Waiting for RayCluster %s/%s to be running", rayCluster.Namespace, rayCluster.Name)
	test.Eventually(RayCluster(test, namespace.Name, rayCluster.Name), TestTimeoutMedium).
		Should(WithTransform(RayClusterState, Equal(rayv1.Ready)))

	// Create RayJob
	rayJob := constructRayJob(test, namespace, rayCluster, accelerator, numberOfGpus)
	rayJob, err = test.Client().Ray().RayV1().RayJobs(namespace.Name).Create(test.Ctx(), rayJob, metav1.CreateOptions{})
	test.Expect(err).NotTo(HaveOccurred())
	test.T().Logf("Created RayJob %s/%s successfully", rayJob.Namespace, rayJob.Name)

	rayDashboardURL := getRayDashboardURL(test, rayCluster.Namespace, rayCluster.Name)

	test.T().Logf("Connecting to Ray cluster at: %s", rayDashboardURL)
	rayClient := GetRayClusterClient(test, rayDashboardURL, test.Config().BearerToken)

	test.T().Logf("Waiting for RayJob %s/%s to complete", rayJob.Namespace, rayJob.Name)
	test.Eventually(RayJob(test, rayJob.Namespace, rayJob.Name), TestTimeoutLong).
		Should(WithTransform(RayJobStatus, Satisfy(rayv1.IsJobTerminal)))

	// Assert the Ray job has completed successfully
	test.Expect(GetRayJob(test, rayJob.Namespace, rayJob.Name)).
		To(WithTransform(RayJobStatus, Equal(rayv1.JobStatusSucceeded)))

	WriteRayJobAPILogs(test, rayClient, GetRayJobId(test, rayJob.Namespace, rayJob.Name))

	test.T().Logf("Deleting AppWrapper %s/%s", aw.Namespace, aw.Name)
	err = test.Client().Dynamic().Resource(appWrapperResource).Namespace(namespace.Name).Delete(test.Ctx(), aw.Name, metav1.DeleteOptions{})
	test.Expect(err).NotTo(HaveOccurred())

	test.T().Logf("Waiting for AppWrapper %s/%s to be deleted", aw.Namespace, aw.Name)
	test.Eventually(AppWrappers(test, namespace), TestTimeoutShort).Should(BeEmpty())
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

func constructRayCluster(_ Test, namespace *corev1.Namespace, mnist *corev1.ConfigMap, numberOfGpus int) *rayv1.RayCluster {
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
							Tolerations: []corev1.Toleration{
								{
									Key:      "nvidia.com/gpu",
									Operator: corev1.TolerationOpExists,
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
											corev1.ResourceMemory: resource.MustParse("1G"),
											"nvidia.com/gpu":      resource.MustParse(fmt.Sprint(numberOfGpus)),
										},
										Limits: corev1.ResourceList{
											corev1.ResourceCPU:    resource.MustParse("2"),
											corev1.ResourceMemory: resource.MustParse("4G"),
											"nvidia.com/gpu":      resource.MustParse(fmt.Sprint(numberOfGpus)),
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
			},
		},
	}
}

func constructRayJob(_ Test, namespace *corev1.Namespace, rayCluster *rayv1.RayCluster, accelerator string, numberOfGpus int) *rayv1.RayJob {
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
    - pytorch_lightning==2.4.0
    - torchmetrics==1.6.0
    - torchvision==0.20.1
  env_vars:
    MNIST_DATASET_URL: "` + GetMnistDatasetURL() + `"
    PIP_INDEX_URL: "` + GetPipIndexURL() + `"
    PIP_TRUSTED_HOST: "` + GetPipTrustedHost() + `"
    ACCELERATOR: "` + accelerator + `"
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
			EntrypointNumCpus: 2,
			// Using EntrypointNumGpus doesn't seem to work properly on KinD cluster with GPU, EntrypointNumCpus seems reliable
			EntrypointNumGpus: float32(numberOfGpus),
		},
	}
}

func getRayDashboardURL(test Test, namespace, rayClusterName string) string {
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

		dashboardUrl, _ := url.Parse("https://" + hostname)
		test.T().Logf("Ray-dashboard route : %s\n", dashboardUrl.String())

		return dashboardUrl.String()
	}

	ingress := GetIngress(test, namespace, dashboardName)

	test.T().Logf("Waiting for Ingress %s/%s to be admitted", ingress.Namespace, ingress.Name)
	test.Eventually(Ingress(test, ingress.Namespace, ingress.Name), TestTimeoutShort).
		Should(WithTransform(LoadBalancerIngresses, HaveLen(1)))

	hostname := ingress.Spec.Rules[0].Host
	dashboardUrl, _ := url.Parse("http://" + hostname)
	test.T().Logf("Ray-dashboard route : %s\n", dashboardUrl.String())

	return dashboardUrl.String()
}

// Create ClusterQueue
func createClusterQueue(test Test, resourceFlavor *v1beta1.ResourceFlavor, numberOfGpus int) *v1beta1.ClusterQueue {
	cqSpec := v1beta1.ClusterQueueSpec{
		NamespaceSelector: &metav1.LabelSelector{},
		ResourceGroups: []v1beta1.ResourceGroup{
			{
				CoveredResources: []corev1.ResourceName{corev1.ResourceName("cpu"), corev1.ResourceName("memory"), corev1.ResourceName("nvidia.com/gpu")},
				Flavors: []v1beta1.FlavorQuotas{
					{
						Name: v1beta1.ResourceFlavorReference(resourceFlavor.Name),
						Resources: []v1beta1.ResourceQuota{
							{
								Name:         corev1.ResourceCPU,
								NominalQuota: resource.MustParse("8"),
							},
							{
								Name:         corev1.ResourceMemory,
								NominalQuota: resource.MustParse("12Gi"),
							},
							{
								Name:         corev1.ResourceName("nvidia.com/gpu"),
								NominalQuota: resource.MustParse(fmt.Sprint(numberOfGpus)),
							},
						},
					},
				},
			},
		},
	}
	return CreateKueueClusterQueue(test, cqSpec)
}
