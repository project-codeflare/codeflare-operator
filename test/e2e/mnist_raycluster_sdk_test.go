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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/project-codeflare/codeflare-operator/test/support"
)

func TestMNISTRayClusterSDK(t *testing.T) {
	test := With(t)
	test.T().Parallel()

	test.T().Skip("Requires https://github.com/project-codeflare/codeflare-sdk/pull/146")

	// Create a namespace
	namespace := test.NewTestNamespace()

	// SDK script
	sdk := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sdk",
			Namespace: namespace.Name,
		},
		BinaryData: map[string][]byte{
			"sdk.py": ReadFile(test, "sdk.py"),
		},
		Immutable: Ptr(true),
	}
	sdk, err := test.Client().Core().CoreV1().ConfigMaps(namespace.Name).Create(test.Ctx(), sdk, metav1.CreateOptions{})
	test.Expect(err).NotTo(HaveOccurred())
	test.T().Logf("Created ConfigMap %s/%s successfully", sdk.Namespace, sdk.Name)

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
			"requirements.txt": ReadFile(test, "requirements.txt"),
		},
		Immutable: Ptr(true),
	}
	requirements, err = test.Client().Core().CoreV1().ConfigMaps(namespace.Name).Create(test.Ctx(), requirements, metav1.CreateOptions{})
	test.Expect(err).NotTo(HaveOccurred())
	test.T().Logf("Created ConfigMap %s/%s successfully", requirements.Namespace, requirements.Name)

	job := &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			APIVersion: batchv1.SchemeGroupVersion.String(),
			Kind:       "Job",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sdk",
			Namespace: namespace.Name,
		},
		Spec: batchv1.JobSpec{
			Completions:  Ptr(int32(1)),
			Parallelism:  Ptr(int32(1)),
			BackoffLimit: Ptr(int32(0)),
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "sdk",
							Image:   "quay.io/opendatahub/notebooks:jupyter-minimal-ubi8-python-3.8-4c8f26e",
							Command: []string{"/bin/sh", "-c", "pip install -r /test/runtime/requirements.txt && python /test/job/sdk.py"},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "sdk",
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
							Name: "sdk",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: sdk.Name,
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
	job, err = test.Client().Core().BatchV1().Jobs(namespace.Name).Create(test.Ctx(), job, metav1.CreateOptions{})
	test.Expect(err).NotTo(HaveOccurred())

	defer JobTroubleshooting(test, job)

	test.T().Logf("Waiting for Job %s/%s to complete successfully", job.Namespace, job.Name)
	test.Eventually(Job(test, job.Namespace, job.Name), TestTimeoutMedium).
		Should(WithTransform(ConditionStatus(batchv1.JobComplete), Equal(corev1.ConditionTrue)))
}
