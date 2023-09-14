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
	"os"
	"testing"

	. "github.com/onsi/gomega"
	mcadv1beta1 "github.com/project-codeflare/multi-cluster-app-dispatcher/pkg/apis/controller/v1beta1"
	rayv1alpha1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1alpha1"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/project-codeflare/codeflare-operator/test/support"
)

var namespaceName string = "test-ns-olmupgrade"
var namespace *corev1.Namespace

// Creates a Ray cluster
func TestMNISTRayClusterUp(t *testing.T) {

	test := With(t)
	test.T().Parallel()
	if os.Getenv("RUN_OLM_TESTS") == "true" {
		// Create a namespace
		namespace = CreateTestNamespaceWithName(test, namespaceName)
		test.T().Logf("Created namespace %s successfully", namespace.Name)

		// Delete namespace only if test failed
		defer func() {
			if t.Failed() {
				DeleteTestNamespace(test, namespaceName)
			}
		}()

		// Test configuration
		config := &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.String(),
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mnist-raycluster-sdk",
				Namespace: namespace.Name,
			},
			BinaryData: map[string][]byte{
				// SDK script
				"raycluster_sdk.py": ReadFile(test, "raycluster_sdk.py"),
			},
			Immutable: Ptr(true),
		}
		config, err := test.Client().Core().CoreV1().ConfigMaps(namespace.Name).Create(test.Ctx(), config, metav1.CreateOptions{})
		test.Expect(err).NotTo(HaveOccurred())
		test.T().Logf("Created ConfigMap %s/%s successfully", config.Namespace, config.Name)

		// SDK client RBAC
		serviceAccount := &corev1.ServiceAccount{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.String(),
				Kind:       "ServiceAccount",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sdk-user",
				Namespace: namespace.Name,
			},
		}
		serviceAccount, err = test.Client().Core().CoreV1().ServiceAccounts(namespace.Name).Create(test.Ctx(), serviceAccount, metav1.CreateOptions{})
		test.Expect(err).NotTo(HaveOccurred())

		role := &rbacv1.Role{
			TypeMeta: metav1.TypeMeta{
				APIVersion: rbacv1.SchemeGroupVersion.String(),
				Kind:       "Role",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "sdk",
				Namespace: namespace.Name,
			},
			Rules: []rbacv1.PolicyRule{
				{
					Verbs:     []string{"get", "create", "delete", "list", "patch", "update"},
					APIGroups: []string{mcadv1beta1.GroupName},
					Resources: []string{"appwrappers"},
				},
				{
					Verbs:     []string{"get", "list"},
					APIGroups: []string{rayv1alpha1.GroupVersion.Group},
					Resources: []string{"rayclusters", "rayclusters/status"},
				},
				{
					Verbs:     []string{"get", "list"},
					APIGroups: []string{"route.openshift.io"},
					Resources: []string{"routes"},
				},
			},
		}
		role, err = test.Client().Core().RbacV1().Roles(namespace.Name).Create(test.Ctx(), role, metav1.CreateOptions{})
		test.Expect(err).NotTo(HaveOccurred())

		roleBinding := &rbacv1.RoleBinding{
			TypeMeta: metav1.TypeMeta{
				APIVersion: rbacv1.SchemeGroupVersion.String(),
				Kind:       "RoleBinding",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "sdk",
			},
			RoleRef: rbacv1.RoleRef{
				APIGroup: rbacv1.SchemeGroupVersion.Group,
				Kind:     "Role",
				Name:     role.Name,
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					APIGroup:  corev1.SchemeGroupVersion.Group,
					Name:      serviceAccount.Name,
					Namespace: serviceAccount.Namespace,
				},
			},
		}
		_, err = test.Client().Core().RbacV1().RoleBindings(namespace.Name).Create(test.Ctx(), roleBinding, metav1.CreateOptions{})
		test.Expect(err).NotTo(HaveOccurred())

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
								Name: "test",
								// FIXME: switch to base Python image once the dependency on OpenShift CLI is removed
								// See https://github.com/project-codeflare/codeflare-sdk/pull/146
								Image:   "quay.io/opendatahub/notebooks:jupyter-minimal-ubi8-python-3.8-4c8f26e",
								Command: []string{"/bin/sh", "-c", "pip install codeflare-sdk==" + GetCodeFlareSDKVersion() + " && cp /test/* . && python raycluster_sdk.py" + " " + namespace.Name},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "test",
										MountPath: "/test",
									},
								},
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
						},
						RestartPolicy:      corev1.RestartPolicyNever,
						ServiceAccountName: serviceAccount.Name,
					},
				},
			},
		}
		job, err = test.Client().Core().BatchV1().Jobs(namespace.Name).Create(test.Ctx(), job, metav1.CreateOptions{})
		test.Expect(err).NotTo(HaveOccurred())
		test.T().Logf("Created Job %s/%s successfully", job.Namespace, job.Name)

		// Retrieving the job logs once it has completed or timed out
		defer WriteJobLogs(test, job.Namespace, job.Name)

		test.T().Logf("Waiting for Job %s/%s to complete", job.Namespace, job.Name)
		test.Eventually(Job(test, job.Namespace, job.Name), TestTimeoutLong).Should(
			Or(
				WithTransform(ConditionStatus(batchv1.JobComplete), Equal(corev1.ConditionTrue)),
				WithTransform(ConditionStatus(batchv1.JobFailed), Equal(corev1.ConditionTrue)),
			))

		// Assert the job has completed successfully
		test.Expect(GetJob(test, job.Namespace, job.Name)).
			To(WithTransform(ConditionStatus(batchv1.JobComplete), Equal(corev1.ConditionTrue)))

	} else {
		test.T().Skip("Skipping OLM upgarde test because RUN_OLM_TESTS is not set")
	}
}

// Submit a Job to the Ray cluster and trains the MNIST dataset using the CodeFlare SDK.
func TestMnistJobSubmit(t *testing.T) {

	test := With(t)
	test.T().Parallel()
	if os.Getenv("RUN_OLM_TESTS") == "true" {

		//delete the namespace after test complete
		defer DeleteTestNamespace(test, namespaceName)

		// Test configuration
		config := &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.String(),
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mnist-ray-job",
				Namespace: namespaceName,
			},
			BinaryData: map[string][]byte{
				// SDK script
				"mnist_rayjob.py": ReadFile(test, "mnist_rayjob.py"),
				// pip requirements
				"requirements.txt": ReadFile(test, "mnist_pip_requirements.txt"),
				// MNIST training script
				"mnist.py": ReadFile(test, "mnist.py"),
			},
			Immutable: Ptr(true),
		}
		config, err := test.Client().Core().CoreV1().ConfigMaps(namespaceName).Create(test.Ctx(), config, metav1.CreateOptions{})
		test.Expect(err).NotTo(HaveOccurred())
		test.T().Logf("Created ConfigMap %s/%s successfully", config.Namespace, config.Name)

		// SDK client RBAC
		serviceAccount := &corev1.ServiceAccount{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.String(),
				Kind:       "ServiceAccount",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ray-user",
				Namespace: namespaceName,
			},
		}
		serviceAccount, err = test.Client().Core().CoreV1().ServiceAccounts(namespaceName).Create(test.Ctx(), serviceAccount, metav1.CreateOptions{})
		test.Expect(err).NotTo(HaveOccurred())

		job := &batchv1.Job{
			TypeMeta: metav1.TypeMeta{
				APIVersion: batchv1.SchemeGroupVersion.String(),
				Kind:       "Job",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rayjob",
				Namespace: namespaceName,
			},
			Spec: batchv1.JobSpec{
				Completions:  Ptr(int32(1)),
				Parallelism:  Ptr(int32(1)),
				BackoffLimit: Ptr(int32(0)),
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "test",
								// FIXME: switch to base Python image once the dependency on OpenShift CLI is removed
								// See https://github.com/project-codeflare/codeflare-sdk/pull/146
								Image:   "quay.io/opendatahub/notebooks:jupyter-minimal-ubi8-python-3.8-4c8f26e",
								Command: []string{"/bin/sh", "-c", "pip install codeflare-sdk==" + GetCodeFlareSDKVersion() + " && cp /test/* . && python mnist_rayjob.py" + " " + namespaceName},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "test",
										MountPath: "/test",
									},
								},
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
						},
						RestartPolicy:      corev1.RestartPolicyNever,
						ServiceAccountName: serviceAccount.Name,
					},
				},
			},
		}
		job, err = test.Client().Core().BatchV1().Jobs(namespaceName).Create(test.Ctx(), job, metav1.CreateOptions{})
		test.Expect(err).NotTo(HaveOccurred())
		test.T().Logf("Created Job %s/%s successfully", job.Namespace, job.Name)

		// Retrieving the job logs once it has completed or timed out
		defer WriteJobLogs(test, job.Namespace, job.Name)

		test.T().Logf("Waiting for Job %s/%s to complete", job.Namespace, job.Name)
		test.Eventually(Job(test, job.Namespace, job.Name), TestTimeoutLong).Should(
			Or(
				WithTransform(ConditionStatus(batchv1.JobComplete), Equal(corev1.ConditionTrue)),
				WithTransform(ConditionStatus(batchv1.JobFailed), Equal(corev1.ConditionTrue)),
			))

		// Assert the job has completed successfully
		test.Expect(GetJob(test, job.Namespace, job.Name)).
			To(WithTransform(ConditionStatus(batchv1.JobComplete), Equal(corev1.ConditionTrue)))

	} else {
		test.T().Skip("Skipping OLM upgarde test because RUN_OLM_TESTS is not set")
	}

}
