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

	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
	. "github.com/project-codeflare/codeflare-common/support"
	mcadv1beta1 "github.com/project-codeflare/multi-cluster-app-dispatcher/pkg/apis/controller/v1beta1"
	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// This test covers the Ray Cluster creation authentication functionality with openshift_oauth
func TestRayClusterSDKAuth(t *testing.T) {
	test := With(t)
	test.T().Parallel()
	if GetClusterType(test) == KindCluster {
		test.T().Skipf("Skipping test as not running on an openshift cluster")
	}

	// Create a namespace
	namespace := test.NewTestNamespace()
	// Test configuration
	config := CreateConfigMap(test, namespace.Name, map[string][]byte{
		// SDK script
		"mnist_raycluster_sdk_auth.py": ReadFile(test, "mnist_raycluster_sdk_auth.py"),
		// pip requirements
		"requirements.txt": ReadFile(test, "mnist_pip_requirements.txt"),
		// MNIST training script
		"mnist.py": ReadFile(test, "mnist.py"),
		// codeflare-sdk installation script
		"install-codeflare-sdk.sh": ReadFile(test, "install-codeflare-sdk.sh"),
	})

	// // Create RBAC, retrieve token for user with limited rights
	policyRules := []rbacv1.PolicyRule{
		{
			Verbs:     []string{"get", "create", "delete", "list", "patch", "update"},
			APIGroups: []string{mcadv1beta1.GroupName},
			Resources: []string{"appwrappers"},
		},
		{
			Verbs:     []string{"get", "list"},
			APIGroups: []string{rayv1.GroupVersion.Group},
			Resources: []string{"rayclusters", "rayclusters/status"},
		},
		{
			Verbs:     []string{"get", "list"},
			APIGroups: []string{"route.openshift.io"},
			Resources: []string{"routes"},
		},
		{
			Verbs:     []string{"get", "list", "create", "delete"},
			APIGroups: []string{"networking.k8s.io"},
			Resources: []string{"ingresses"},
		},
	}

	// // Create cluster wide RBAC, required for SDK OpenShift check
	// // TODO reevaluate once SDK change OpenShift detection logic
	clusterPolicyRules := []rbacv1.PolicyRule{
		{
			Verbs:         []string{"get", "list"},
			APIGroups:     []string{"config.openshift.io"},
			Resources:     []string{"ingresses"},
			ResourceNames: []string{"cluster"},
		},
		{
			Verbs:     []string{"create", "update"},
			APIGroups: []string{""},
			Resources: []string{"services", "serviceaccounts"},
		},
		{
			Verbs:     []string{"create"},
			APIGroups: []string{"authorization.k8s.io"},
			Resources: []string{"subjectaccessreviews"},
		},
		{
			Verbs:     []string{"get", "create", "delete", "list", "patch", "update"},
			APIGroups: []string{"rbac.authorization.k8s.io"},
			Resources: []string{"clusterrolebindings"},
		},
		{
			Verbs:     []string{"create"},
			APIGroups: []string{"authentication.k8s.io"},
			Resources: []string{"tokenreviews"},
		},
	}

	sa := CreateServiceAccount(test, namespace.Name)
	role := CreateRole(test, namespace.Name, policyRules)
	CreateRoleBinding(test, namespace.Name, sa, role)
	clusterRole := CreateClusterRole(test, clusterPolicyRules)
	CreateClusterRoleBinding(test, sa, clusterRole)

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
							Image: "quay.io/opendatahub/notebooks:jupyter-minimal-ubi8-python-3.8-4c8f26e",
							Env: []corev1.EnvVar{
								{Name: "PYTHONUSERBASE", Value: "/workdir"},
								{Name: "RAY_IMAGE", Value: GetRayImage()},
							},
							Command: []string{"/bin/sh", "-c", "cp /test/* . && chmod +x install-codeflare-sdk.sh && ./install-codeflare-sdk.sh && python mnist_raycluster_sdk_auth.py" + " " + namespace.Name},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "test",
									MountPath: "/test",
								},
								{
									Name:      "codeflare-sdk",
									MountPath: "/codeflare-sdk",
								},
								{
									Name:      "workdir",
									MountPath: "/workdir",
								},
							},
							WorkingDir: "/workdir",
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: Ptr(false),
								SeccompProfile: &corev1.SeccompProfile{
									Type: "RuntimeDefault",
								},
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{"ALL"},
								},
								RunAsNonRoot: Ptr(true),
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
						{
							Name: "codeflare-sdk",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "workdir",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					RestartPolicy:      corev1.RestartPolicyNever,
					ServiceAccountName: sa.Name,
				},
			},
		},
	}

	job, err := test.Client().Core().BatchV1().Jobs(namespace.Name).Create(test.Ctx(), job, metav1.CreateOptions{})
	test.Expect(err).NotTo(HaveOccurred())
	test.T().Logf("Created Job %s/%s successfully", job.Namespace, job.Name)

	test.T().Logf("Waiting for Job %s/%s to complete", job.Namespace, job.Name)
	test.Eventually(Job(test, job.Namespace, job.Name), TestTimeoutLong).Should(
		WithTransform(ConditionStatus(batchv1.JobFailed), Equal(corev1.ConditionTrue)))

	// get Pod associated with created job
	options := metav1.ListOptions{
		LabelSelector: "job-name=sdk",
	}
	pods := GetPods(test, namespace.Name, options)

	// get job pod logs and compare the expected error in logs
	for _, pod := range pods {
		podLogs := GetPodLogs(test, &pod, corev1.PodLogOptions{})
		expectedLogError := "kubernetes.client.exceptions.ApiException: (403)\nReason: Forbidden"
		test.Expect(string(podLogs)).To(gomega.ContainSubstring(expectedLogError))
	}

}
