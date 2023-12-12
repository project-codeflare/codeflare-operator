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

package upgrade

import (
	"testing"

	. "github.com/onsi/gomega"
	. "github.com/project-codeflare/codeflare-common/support"
	mcadv1beta1 "github.com/project-codeflare/multi-cluster-app-dispatcher/pkg/apis/controller/v1beta1"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/project-codeflare/codeflare-operator/test/e2e"
)

var (
	namespaceName  = "test-ns-olmupgrade"
	appWrapperName = "mnist"
	jobName        = "mnist-job"
)

func TestMNISTCreateAppWrapper(t *testing.T) {
	test := With(t)
	test.T().Parallel()

	// Create a namespace
	namespace := CreateTestNamespaceWithName(test, namespaceName)

	// Delete namespace only if test failed
	defer func() {
		if t.Failed() {
			DeleteTestNamespace(test, namespace)
		} else {
			StoreNamespaceLogs(test, namespace)
		}
	}()

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
			Name:      jobName,
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
								corev1.EnvVar{Name: "PYTHONUSERBASE", Value: "/workdir"},
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
			Suspend: Ptr(true),
		},
	}

	// Create an AppWrapper resource
	aw := &mcadv1beta1.AppWrapper{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appWrapperName,
			Namespace: namespace.Name,
		},
		Spec: mcadv1beta1.AppWrapperSpec{
			AggrResources: mcadv1beta1.AppWrapperResourceList{
				GenericItems: []mcadv1beta1.AppWrapperGenericResource{
					{
						DesiredAvailable: 1,
						CustomPodResources: []mcadv1beta1.CustomPodResourceTemplate{
							{
								Replicas: 1,
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("250m"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("1"),
									corev1.ResourceMemory: resource.MustParse("1G"),
								},
							},
						},
						GenericTemplate:  Raw(test, job),
						CompletionStatus: "Complete",
					},
				},
			},
		},
	}

	_, err = test.Client().MCAD().WorkloadV1beta1().AppWrappers(namespace.Name).Create(test.Ctx(), aw, metav1.CreateOptions{})
	test.Expect(err).NotTo(HaveOccurred())
	test.T().Logf("Created MCAD AppWrapper %s/%s successfully", aw.Namespace, aw.Name)

	test.T().Logf("Waiting for MCAD %s/%s to be running", aw.Namespace, aw.Name)
	test.Eventually(AppWrapper(test, namespace, aw.Name), TestTimeoutMedium).
		Should(WithTransform(AppWrapperState, Equal(mcadv1beta1.AppWrapperStateActive)))

}

func TestMNISTCheckAppWrapperStatus(t *testing.T) {
	test := With(t)
	test.T().Parallel()

	// get namespace
	namespace := GetNamespaceWithName(test, namespaceName)

	//delete the namespace after test complete
	defer DeleteTestNamespace(test, namespace)

	// Patch job to resume execution
	patch := []byte(`[{"op":"replace","path":"/spec/suspend","value": false}]`)
	job, err := test.Client().Core().BatchV1().Jobs(namespace.Name).Patch(test.Ctx(), jobName, types.JSONPatchType, patch, metav1.PatchOptions{})
	test.Expect(err).NotTo(HaveOccurred())

	test.T().Logf("Waiting for Job %s/%s to complete", job.Namespace, job.Name)
	test.Eventually(Job(test, job.Namespace, job.Name), TestTimeoutLong).Should(
		Or(
			WithTransform(ConditionStatus(batchv1.JobComplete), Equal(corev1.ConditionTrue)),
			WithTransform(ConditionStatus(batchv1.JobFailed), Equal(corev1.ConditionTrue)),
		))

	test.T().Logf("Waiting for AppWrapper %s/%s to complete", namespace.Name, appWrapperName)
	test.Eventually(AppWrapper(test, namespace, appWrapperName), TestTimeoutShort).Should(
		Or(
			WithTransform(AppWrapperState, Equal(mcadv1beta1.AppWrapperStateCompleted)),
			WithTransform(AppWrapperState, Equal(mcadv1beta1.AppWrapperStateFailed)),
		))

	// Assert the AppWrapper has completed successfully
	test.Expect(GetAppWrapper(test, namespace, appWrapperName)).
		To(WithTransform(AppWrapperState, Equal(mcadv1beta1.AppWrapperStateCompleted)))

}
