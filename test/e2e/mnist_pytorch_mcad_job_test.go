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
	"testing"

	. "github.com/onsi/gomega"
	mcadv1beta2 "github.com/project-codeflare/appwrapper/api/v1beta2"
	. "github.com/project-codeflare/codeflare-common/support"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// Trains the MNIST dataset as a batch Job managed by MCAD, and asserts successful completion of the training job.
func TestMNISTPyTorchMCAD(t *testing.T) {
	test := With(t)
	test.T().Parallel()

	// Create a namespace
	namespace := test.NewTestNamespace()

	// Test configuration
	config := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mnist-mcad",
			Namespace: namespace.Name,
		},
		BinaryData: map[string][]byte{
			// pip requirements
			"requirements.txt": ReadFile(test, "mnist_pip_requirements.txt"),
			// MNIST training script
			"mnist.py": ReadFile(test, "mnist.py"),
		},
		Immutable: Ptr(true),
	}
	config, err := test.Client().Core().CoreV1().ConfigMaps(namespace.Name).Create(test.Ctx(), config, metav1.CreateOptions{})
	test.Expect(err).NotTo(HaveOccurred())
	test.T().Logf("Created ConfigMap %s/%s successfully", config.Namespace, config.Name)

	// Batch Job
	job := &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			APIVersion: batchv1.SchemeGroupVersion.String(),
			Kind:       "Job",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mnist",
			Namespace: namespace.Name,
		},
		Spec: batchv1.JobSpec{
			Completions: Ptr(int32(1)),
			Parallelism: Ptr(int32(1)),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "job",
							Image: GetPyTorchImage(),
							Env: []corev1.EnvVar{
								{Name: "PYTHONUSERBASE", Value: "/workdir"},
								{Name: "MNIST_DATASET_URL", Value: GetMnistDatasetURL()},
								{Name: "PIP_INDEX_URL", Value: GetPipIndexURL()},
								{Name: "PIP_TRUSTED_HOST", Value: GetPipTrustedHost()},
							},
							Command: []string{"/bin/sh", "-c", "pip install -r /test/requirements.txt && torchrun /test/mnist.py"},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "test",
									MountPath: "/test",
								},
								{
									Name:      "workdir",
									MountPath: "/workdir",
								},
							},
							WorkingDir: "/workdir",
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "test",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: config.Name,
									},
								},
							},
						},
						{
							Name: "workdir",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}

	// Create an AppWrapper resource
	aw := &mcadv1beta2.AppWrapper{
		TypeMeta: metav1.TypeMeta{
			APIVersion: mcadv1beta2.GroupVersion.String(),
			Kind:       "AppWrapper",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mnist",
			Namespace: namespace.Name,
		},
		Spec: mcadv1beta2.AppWrapperSpec{
			Components: []mcadv1beta2.AppWrapperComponent{
				{
					PodSets:  []mcadv1beta2.AppWrapperPodSet{{Replicas: Ptr(int32(1)), Path: "template.spec.template"}},
					Template: Raw(test, job),
				},
			},
		},
	}

	appWrapperResource := mcadv1beta2.GroupVersion.WithResource("appwrappers")
	awMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(aw)
	test.Expect(err).NotTo(HaveOccurred())
	unstruct := unstructured.Unstructured{Object: awMap}
	_, err = test.Client().Dynamic().Resource(appWrapperResource).Namespace(namespace.Name).Create(test.Ctx(), &unstruct, metav1.CreateOptions{})
	test.Expect(err).NotTo(HaveOccurred())
	test.T().Logf("Created AppWrapper %s/%s successfully", aw.Namespace, aw.Name)

	test.T().Logf("Waiting for AppWrapper %s/%s to be running", aw.Namespace, aw.Name)
	test.Eventually(AppWrapper(test, namespace, aw.Name), TestTimeoutMedium).
		Should(WithTransform(AppWrapperPhase, Equal(mcadv1beta2.AppWrapperRunning)))

	test.T().Logf("Waiting for AppWrapper %s/%s to complete", job.Namespace, job.Name)
	test.Eventually(AppWrapper(test, namespace, aw.Name), TestTimeoutLong).Should(
		Or(
			WithTransform(AppWrapperPhase, Equal(mcadv1beta2.AppWrapperSucceeded)),
			WithTransform(AppWrapperPhase, Equal(mcadv1beta2.AppWrapperFailed)),
		))
}
