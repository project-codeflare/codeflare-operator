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

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"

	. "github.com/project-codeflare/codeflare-operator/test/support"
	mcadv1beta1 "github.com/project-codeflare/multi-cluster-app-dispatcher/pkg/apis/controller/v1beta1"
)

func TestMNISTPyTorchMCAD(t *testing.T) {
	test := With(t)
	test.T().Parallel()

	// Create a namespace
	namespace := test.NewTestNamespace()

	// MNIST training script
	mnist, err := scripts.ReadFile("mnist.py")
	test.Expect(err).NotTo(HaveOccurred())

	mnistScript := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mnist",
			Namespace: namespace.Name,
		},
		BinaryData: map[string][]byte{
			"mnist.py": mnist,
		},
		Immutable: Ptr(true),
	}
	mnistScript, err = test.Client().Core().CoreV1().ConfigMaps(namespace.Name).Create(test.Ctx(), mnistScript, metav1.CreateOptions{})
	test.Expect(err).NotTo(HaveOccurred())
	test.T().Logf("Created ConfigMap %s/%s successfully", mnistScript.Namespace, mnistScript.Name)

	// pip requirements
	requirements := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "requirements",
			Namespace: namespace.Name,
		},
		BinaryData: map[string][]byte{
			"requirements.txt": []byte(`
pytorch_lightning==1.5.10
torchmetrics==0.9.1
torchvision==0.12.0
`),
		},
		Immutable: Ptr(true),
	}
	requirements, err = test.Client().Core().CoreV1().ConfigMaps(namespace.Name).Create(test.Ctx(), requirements, metav1.CreateOptions{})
	test.Expect(err).NotTo(HaveOccurred())
	test.T().Logf("Created ConfigMap %s/%s successfully", requirements.Namespace, requirements.Name)

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
							Name:    "job",
							Image:   "pytorch/pytorch:1.11.0-cuda11.3-cudnn8-runtime",
							Command: []string{"/bin/sh", "-c", "pip install -r /test/runtime/requirements.txt && torchrun /test/job/mnist.py"},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "mnist",
									MountPath: "/test/job",
								},
								{
									Name:      "requirements",
									MountPath: "/test/runtime",
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
										Name: mnistScript.Name,
									},
								},
							},
						},
						{
							Name: "requirements",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: requirements.Name,
									},
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}

	// Create an AppWrapper resource
	aw := &mcadv1beta1.AppWrapper{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mnist",
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
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("1G"),
								},
							},
						},
						GenericTemplate: Raw(test, job),
					},
				},
			},
		},
	}

	_, err = test.Client().MCAD().ArbV1().AppWrappers(namespace.Name).Create(aw)
	test.Expect(err).NotTo(HaveOccurred())
	test.T().Logf("Created MCAD %s/%s successfully", aw.Namespace, aw.Name)

	test.T().Logf("Waiting for MCAD %s/%s to be running", aw.Namespace, aw.Name)
	test.Eventually(AppWrapper(test, namespace, aw.Name), TestTimeoutMedium).
		Should(WithTransform(AppWrapperState, Equal(mcadv1beta1.AppWrapperStateActive)))

	defer troubleshooting(test, job)

	test.T().Logf("Waiting for Job %s/%s to complete successfully", job.Namespace, job.Name)
	test.Eventually(Job(test, job.Namespace, job.Name), TestTimeoutLong).
		Should(WithTransform(ConditionStatus(batchv1.JobComplete), Equal(corev1.ConditionTrue)))

	// Refresh the job to get the generated pod selector
	job = GetJob(test, job.Namespace, job.Name)

	// Get the job Pod
	pods := GetPods(test, job.Namespace, metav1.ListOptions{
		LabelSelector: labels.FormatLabels(job.Spec.Selector.MatchLabels)},
	)
	test.Expect(pods).To(HaveLen(1))

	// Print the job logs
	test.T().Logf("Printing Job %s/%s logs", job.Namespace, job.Name)
	test.T().Log(GetPodLogs(test, &pods[0], corev1.PodLogOptions{}))
}

func troubleshooting(test Test, job *batchv1.Job) {
	if !test.T().Failed() {
		return
	}
	job = GetJob(test, job.Namespace, job.Name)

	test.T().Errorf("Job %s/%s hasn't completed in time: %s", job.Namespace, job.Name, job)

	pods := GetPods(test, job.Namespace, metav1.ListOptions{
		LabelSelector: labels.FormatLabels(job.Spec.Selector.MatchLabels)},
	)

	if len(pods) == 0 {
		test.T().Errorf("Job %s/%s has no pods scheduled", job.Namespace, job.Name)
	} else {
		for i, pod := range pods {
			test.T().Logf("Printing Pod %s/%s logs", pod.Namespace, pod.Name)
			test.T().Log(GetPodLogs(test, &pods[i], corev1.PodLogOptions{}))
		}
	}
}
