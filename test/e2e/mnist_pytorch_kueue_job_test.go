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

package e2e

import (
	"testing"

	. "github.com/onsi/gomega"
	. "github.com/project-codeflare/codeflare-common/support"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Trains the MNIST dataset as a batch Job managed by Kueue, and asserts successful completion of the training job.
func TestMNISTPyTorchKueue(t *testing.T) {
	test := With(t)
	test.T().Parallel()

	// Create a namespace
	namespace := test.NewTestNamespace()

	// Test configuration
	cm := CreateConfigMap(test, namespace.Name, map[string][]byte{
		// pip requirements
		"requirements.txt": ReadFile(test, "mnist_pip_requirements.txt"),
		// MNIST training script
		"mnist.py": ReadFile(test, "mnist.py"),
	})

	// Create LocalQueue for Kueue
	lq := CreateKueueLocalQueue(test, namespace.Name, GetKueueClusterQueueName(test))

	// Batch Job
	job := &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			APIVersion: batchv1.SchemeGroupVersion.String(),
			Kind:       "Job",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mnist",
			Namespace: namespace.Name,
			Labels:    map[string]string{"kueue.x-k8s.io/queue-name": lq.Name},
		},
		Spec: batchv1.JobSpec{
			Suspend:     Ptr(true),
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
										Name: cm.Name,
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
	_, err := test.Client().Core().BatchV1().Jobs(namespace.Name).Create(test.Ctx(), job, metav1.CreateOptions{})
	test.Expect(err).NotTo(HaveOccurred())
	test.T().Logf("Created Job %s/%s successfully", job.Namespace, job.Name)

	test.T().Logf("Waiting for Job %s/%s to complete", job.Namespace, job.Name)
	test.Eventually(Job(test, job.Namespace, job.Name), TestTimeoutLong).Should(
		Or(
			WithTransform(ConditionStatus(batchv1.JobComplete), Equal(corev1.ConditionTrue)),
			WithTransform(ConditionStatus(batchv1.JobFailed), Equal(corev1.ConditionTrue)),
		))

	// Assert the job has completed successfully
	test.Expect(GetJob(test, job.Namespace, job.Name)).
		To(WithTransform(ConditionStatus(batchv1.JobComplete), Equal(corev1.ConditionTrue)))
}
