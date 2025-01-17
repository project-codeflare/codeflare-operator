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

package controllers

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	routev1 "github.com/openshift/api/route/v1"
)

var _ = Describe("RayCluster controller", func() {
	Context("RayCluster controller test", func() {
		rayClusterName := "test-raycluster"
		var namespaceName string
		BeforeEach(func(ctx SpecContext) {
			By("Creating a namespace for running the tests.")
			namespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: "test-",
				},
			}
			namespace, err := k8sClient.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func(ctx SpecContext) {
				err := k8sClient.CoreV1().Namespaces().Delete(ctx, namespace.Name, metav1.DeleteOptions{})
				Expect(err).To(Not(HaveOccurred()))
			})
			namespaceName = namespace.Name

			By("creating a basic instance of the RayCluster CR")
			raycluster := &rayv1.RayCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      rayClusterName,
					Namespace: namespace.Name,
				},
				Spec: rayv1.RayClusterSpec{
					HeadGroupSpec: rayv1.HeadGroupSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								Containers: []corev1.Container{},
							},
						},
						RayStartParams: map[string]string{},
					},
				},
			}
			_, err = rayClient.RayV1().RayClusters(namespace.Name).Create(ctx, raycluster, metav1.CreateOptions{})
			Expect(err).To(Not(HaveOccurred()))
		})

		AfterEach(func(ctx SpecContext) {
			By("removing instances of the RayClusters used")
			rayClusters, err := rayClient.RayV1().RayClusters(namespaceName).List(ctx, metav1.ListOptions{})
			Expect(err).To(Not(HaveOccurred()))

			for _, rayCluster := range rayClusters.Items {
				err = rayClient.RayV1().RayClusters(namespaceName).Delete(ctx, rayCluster.Name, metav1.DeleteOptions{})
				Expect(err).To(Not(HaveOccurred()))
			}

			Eventually(func() ([]rayv1.RayCluster, error) {
				rayClusters, err := rayClient.RayV1().RayClusters(namespaceName).List(ctx, metav1.ListOptions{})
				return rayClusters.Items, err
			}).WithTimeout(time.Second * 10).Should(BeEmpty())
		})

		It("should have oauth finalizer set", func(ctx SpecContext) {
			Eventually(func() ([]string, error) {
				foundRayCluster, err := rayClient.RayV1().RayClusters(namespaceName).Get(ctx, rayClusterName, metav1.GetOptions{})
				return foundRayCluster.Finalizers, err
			}).WithTimeout(time.Second * 10).Should(ContainElement(oAuthFinalizer))
		}, SpecTimeout(time.Second*10))

		It("should create all oauth resources", func(ctx SpecContext) {
			foundRayCluster, err := rayClient.RayV1().RayClusters(namespaceName).Get(ctx, rayClusterName, metav1.GetOptions{})
			Expect(err).To(Not(HaveOccurred()))

			Eventually(func() (*corev1.Secret, error) {
				return k8sClient.CoreV1().Secrets(namespaceName).Get(ctx, oauthSecretNameFromCluster(foundRayCluster), metav1.GetOptions{})
			}).WithTimeout(time.Second * 10).ShouldNot(BeNil())
			Eventually(func() (*corev1.Service, error) {
				return k8sClient.CoreV1().Services(namespaceName).Get(ctx, oauthServiceNameFromCluster(foundRayCluster), metav1.GetOptions{})
			}).WithTimeout(time.Second * 10).ShouldNot(BeNil())
			Eventually(func() (*corev1.ServiceAccount, error) {
				return k8sClient.CoreV1().ServiceAccounts(namespaceName).Get(ctx, oauthServiceAccountNameFromCluster(foundRayCluster), metav1.GetOptions{})
			}).WithTimeout(time.Second * 10).ShouldNot(BeNil())
			Eventually(func() (*rbacv1.ClusterRoleBinding, error) {
				return k8sClient.RbacV1().ClusterRoleBindings().Get(ctx, crbNameFromCluster(foundRayCluster), metav1.GetOptions{})
			}).WithTimeout(time.Second * 10).ShouldNot(BeNil())
			Eventually(func() (*routev1.Route, error) {
				return routeClient.RouteV1().Routes(namespaceName).Get(ctx, dashboardNameFromCluster(foundRayCluster), metav1.GetOptions{})
			}).WithTimeout(time.Second * 10).ShouldNot(BeNil())
		})

		It("should set owner references for all resources", func(ctx SpecContext) {
			foundRayCluster, err := rayClient.RayV1().RayClusters(namespaceName).Get(ctx, rayClusterName, metav1.GetOptions{})
			Expect(err).To(Not(HaveOccurred()))

			Eventually(func() (*corev1.Secret, error) {
				return k8sClient.CoreV1().Secrets(namespaceName).Get(ctx, oauthSecretNameFromCluster(foundRayCluster), metav1.GetOptions{})
			}).WithTimeout(time.Second * 10).Should(WithTransform(OwnerReferenceKind, Equal("RayCluster")))
			Eventually(func() (*corev1.Secret, error) {
				return k8sClient.CoreV1().Secrets(namespaceName).Get(ctx, oauthSecretNameFromCluster(foundRayCluster), metav1.GetOptions{})
			}).WithTimeout(time.Second * 10).Should(WithTransform(OwnerReferenceName, Equal(foundRayCluster.Name)))
			Eventually(func() (*corev1.Service, error) {
				return k8sClient.CoreV1().Services(namespaceName).Get(ctx, oauthServiceNameFromCluster(foundRayCluster), metav1.GetOptions{})
			}).WithTimeout(time.Second * 10).Should(WithTransform(OwnerReferenceKind, Equal("RayCluster")))
			Eventually(func() (*corev1.Service, error) {
				return k8sClient.CoreV1().Services(namespaceName).Get(ctx, oauthServiceNameFromCluster(foundRayCluster), metav1.GetOptions{})
			}).WithTimeout(time.Second * 10).Should(WithTransform(OwnerReferenceName, Equal(foundRayCluster.Name)))
			Eventually(func() (*corev1.ServiceAccount, error) {
				return k8sClient.CoreV1().ServiceAccounts(namespaceName).Get(ctx, oauthServiceAccountNameFromCluster(foundRayCluster), metav1.GetOptions{})
			}).WithTimeout(time.Second * 10).Should(WithTransform(OwnerReferenceKind, Equal("RayCluster")))
			Eventually(func() (*corev1.ServiceAccount, error) {
				return k8sClient.CoreV1().ServiceAccounts(namespaceName).Get(ctx, oauthServiceAccountNameFromCluster(foundRayCluster), metav1.GetOptions{})
			}).WithTimeout(time.Second * 10).Should(WithTransform(OwnerReferenceName, Equal(foundRayCluster.Name)))
			Eventually(func() (*routev1.Route, error) {
				return routeClient.RouteV1().Routes(namespaceName).Get(ctx, dashboardNameFromCluster(foundRayCluster), metav1.GetOptions{})
			}).WithTimeout(time.Second * 10).Should(WithTransform(OwnerReferenceKind, Equal("RayCluster")))
			Eventually(func() (*routev1.Route, error) {
				return routeClient.RouteV1().Routes(namespaceName).Get(ctx, dashboardNameFromCluster(foundRayCluster), metav1.GetOptions{})
			}).WithTimeout(time.Second * 10).Should(WithTransform(OwnerReferenceName, Equal(foundRayCluster.Name)))
		})

		It("should delete the head pod if missing image pull secrets", func(ctx SpecContext) {
			foundRayCluster, err := rayClient.RayV1().RayClusters(namespaceName).Get(ctx, rayClusterName, metav1.GetOptions{})
			Expect(err).To(Not(HaveOccurred()))

			Eventually(func() (*corev1.ServiceAccount, error) {
				return k8sClient.CoreV1().ServiceAccounts(namespaceName).Get(ctx, oauthServiceAccountNameFromCluster(foundRayCluster), metav1.GetOptions{})
			}).WithTimeout(time.Second * 10).Should(WithTransform(OwnerReferenceKind, Equal("RayCluster")))

			headPodName := "head-pod"
			headPod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      headPodName,
					Namespace: namespaceName,
					Labels: map[string]string{
						"ray.io/node-type": "head",
						"ray.io/cluster":   foundRayCluster.Name,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "head-container",
							Image: "busybox",
						},
					},
				},
			}
			_, err = k8sClient.CoreV1().Pods(namespaceName).Create(ctx, headPod, metav1.CreateOptions{})
			Expect(err).To(Not(HaveOccurred()))

			Eventually(func() (*corev1.Pod, error) {
				return k8sClient.CoreV1().Pods(namespaceName).Get(ctx, headPodName, metav1.GetOptions{})
			}).WithTimeout(time.Second * 10).ShouldNot(BeNil())

			sa, err := k8sClient.CoreV1().ServiceAccounts(namespaceName).Get(ctx, oauthServiceAccountNameFromCluster(foundRayCluster), metav1.GetOptions{})
			Expect(err).To(Not(HaveOccurred()))

			sa.ImagePullSecrets = append(sa.ImagePullSecrets, corev1.LocalObjectReference{Name: "test-image-pull-secret"})
			_, err = k8sClient.CoreV1().ServiceAccounts(namespaceName).Update(ctx, sa, metav1.UpdateOptions{})
			Expect(err).To(Not(HaveOccurred()))

			Eventually(func() error {
				_, err := k8sClient.CoreV1().Pods(namespaceName).Get(ctx, headPodName, metav1.GetOptions{})
				return err
			}).WithTimeout(time.Second * 10).Should(Satisfy(errors.IsNotFound))
		})

		It("should not delete the head pod if RayCluster CR provides image pull secrets", func(ctx SpecContext) {
			By("creating an instance of the RayCluster CR with imagePullSecret")
			rayclusterWithPullSecret := &rayv1.RayCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pull-secret-cluster",
					Namespace: namespaceName,
				},
				Spec: rayv1.RayClusterSpec{
					HeadGroupSpec: rayv1.HeadGroupSpec{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								ImagePullSecrets: []corev1.LocalObjectReference{{Name: "custom-pull-secret"}},
								Containers:       []corev1.Container{},
							},
						},
						RayStartParams: map[string]string{},
					},
				},
			}
			_, err := rayClient.RayV1().RayClusters(namespaceName).Create(ctx, rayclusterWithPullSecret, metav1.CreateOptions{})
			Expect(err).To(Not(HaveOccurred()))

			Eventually(func() (*corev1.ServiceAccount, error) {
				return k8sClient.CoreV1().ServiceAccounts(namespaceName).Get(ctx, oauthServiceAccountNameFromCluster(rayclusterWithPullSecret), metav1.GetOptions{})
			}).WithTimeout(time.Second * 10).Should(WithTransform(OwnerReferenceKind, Equal("RayCluster")))

			headPodName := "head-pod"
			headPod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      headPodName,
					Namespace: namespaceName,
					Labels: map[string]string{
						"ray.io/node-type": "head",
						"ray.io/cluster":   rayclusterWithPullSecret.Name,
					},
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets: []corev1.LocalObjectReference{
						{Name: "custom-pull-secret"},
					},
					Containers: []corev1.Container{
						{
							Name:  "head-container",
							Image: "busybox",
						},
					},
				},
			}
			_, err = k8sClient.CoreV1().Pods(namespaceName).Create(ctx, headPod, metav1.CreateOptions{})
			Expect(err).To(Not(HaveOccurred()))

			Eventually(func() (*corev1.Pod, error) {
				return k8sClient.CoreV1().Pods(namespaceName).Get(ctx, headPodName, metav1.GetOptions{})
			}).WithTimeout(time.Second * 10).ShouldNot(BeNil())

			sa, err := k8sClient.CoreV1().ServiceAccounts(namespaceName).Get(ctx, oauthServiceAccountNameFromCluster(rayclusterWithPullSecret), metav1.GetOptions{})
			Expect(err).To(Not(HaveOccurred()))

			sa.ImagePullSecrets = append(sa.ImagePullSecrets, corev1.LocalObjectReference{Name: "test-image-pull-secret"})
			_, err = k8sClient.CoreV1().ServiceAccounts(namespaceName).Update(ctx, sa, metav1.UpdateOptions{})
			Expect(err).To(Not(HaveOccurred()))

			Consistently(func() (*corev1.Pod, error) {
				return k8sClient.CoreV1().Pods(namespaceName).Get(ctx, headPodName, metav1.GetOptions{})
			}).WithTimeout(time.Second * 5).Should(Not(BeNil()))
		})

		It("should remove CRB when the RayCluster is deleted", func(ctx SpecContext) {
			foundRayCluster, err := rayClient.RayV1().RayClusters(namespaceName).Get(ctx, rayClusterName, metav1.GetOptions{})
			Expect(err).To(Not(HaveOccurred()))

			err = rayClient.RayV1().RayClusters(namespaceName).Delete(ctx, foundRayCluster.Name, metav1.DeleteOptions{})
			Expect(err).To(Not(HaveOccurred()))

			Eventually(func() error {
				_, err := k8sClient.RbacV1().ClusterRoleBindings().Get(ctx, crbNameFromCluster(foundRayCluster), metav1.GetOptions{})
				return err
			}).WithTimeout(time.Second * 10).Should(Satisfy(errors.IsNotFound))
		})
	})
})

func OwnerReferenceKind(meta metav1.Object) string {
	return meta.GetOwnerReferences()[0].Kind
}

func OwnerReferenceName(meta metav1.Object) string {
	return meta.GetOwnerReferences()[0].Name
}
