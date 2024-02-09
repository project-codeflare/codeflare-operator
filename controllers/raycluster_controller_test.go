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
	"k8s.io/apimachinery/pkg/types"

	routev1 "github.com/openshift/api/route/v1"
)

var _ = Describe("RayCluster controller", func() {
	Context("RayCluster controller test", func() {
		var rayClusterName = "test-raycluster"
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
				k8sClient.CoreV1().Namespaces().Delete(ctx, namespace.Name, metav1.DeleteOptions{})
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
								Containers: []corev1.Container{
									corev1.Container{},
								},
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
			}).WithTimeout(time.Second * 10).Should(ContainElement(CodeflareOAuthFinalizer))
		}, SpecTimeout(time.Second*10))

		Context("Cluster has OAuth annotation", func() {
			BeforeEach(func(ctx SpecContext) {
				By("adding OAuth annotation to RayCluster")
				patch := []byte(`{"metadata":{"annotations":{"codeflare.dev/oauth":"true"}}}`)
				foundRayCluster, err := rayClient.RayV1().RayClusters(namespaceName).Patch(ctx, rayClusterName, types.MergePatchType, patch, metav1.PatchOptions{})
				Expect(err).To(Not(HaveOccurred()))

				By("waiting for dependent resources to be created")
				Eventually(func() (*corev1.Secret, error) {
					return k8sClient.CoreV1().Secrets(namespaceName).Get(ctx, oauthSecretNameFromCluster(foundRayCluster), metav1.GetOptions{})
				}).WithTimeout(time.Second * 10).ShouldNot(BeNil())
				Eventually(func() (*corev1.Service, error) {
					return k8sClient.CoreV1().Services(namespaceName).Get(ctx, oauthServiceNameFromCluster(foundRayCluster), metav1.GetOptions{})
				}).WithTimeout(time.Second * 10).ShouldNot(BeNil())
				Eventually(func() (*corev1.ServiceAccount, error) {
					return k8sClient.CoreV1().ServiceAccounts(namespaceName).Get(ctx, foundRayCluster.Name, metav1.GetOptions{})
				}).WithTimeout(time.Second * 10).ShouldNot(BeNil())
				Eventually(func() (*rbacv1.ClusterRoleBinding, error) {
					return k8sClient.RbacV1().ClusterRoleBindings().Get(ctx, crbNameFromCluster(foundRayCluster), metav1.GetOptions{})
				}).WithTimeout(time.Second * 10).ShouldNot(BeNil())
				Eventually(func() (*routev1.Route, error) {
					return routeClient.RouteV1().Routes(namespaceName).Get(ctx, foundRayCluster.Name, metav1.GetOptions{})
				}).WithTimeout(time.Second * 10).ShouldNot(BeNil())
			})

			It("should set owner references for all resources", func(ctx SpecContext) {
				foundRayCluster, err := rayClient.RayV1().RayClusters(namespaceName).Get(ctx, rayClusterName, metav1.GetOptions{})
				Expect(err).To(Not(HaveOccurred()))

				Expect(k8sClient.CoreV1().Secrets(namespaceName).Get(ctx, oauthSecretNameFromCluster(foundRayCluster), metav1.GetOptions{})).To(WithTransform(OwnerReferenceKind, Equal("RayCluster")))
				Expect(k8sClient.CoreV1().Secrets(namespaceName).Get(ctx, oauthSecretNameFromCluster(foundRayCluster), metav1.GetOptions{})).To(WithTransform(OwnerReferenceName, Equal(foundRayCluster.Name)))
				Expect(k8sClient.CoreV1().Services(namespaceName).Get(ctx, oauthServiceNameFromCluster(foundRayCluster), metav1.GetOptions{})).To(WithTransform(OwnerReferenceKind, Equal("RayCluster")))
				Expect(k8sClient.CoreV1().Services(namespaceName).Get(ctx, oauthServiceNameFromCluster(foundRayCluster), metav1.GetOptions{})).To(WithTransform(OwnerReferenceName, Equal(foundRayCluster.Name)))
				Expect(k8sClient.CoreV1().ServiceAccounts(namespaceName).Get(ctx, foundRayCluster.Name, metav1.GetOptions{})).To(WithTransform(OwnerReferenceKind, Equal("RayCluster")))
				Expect(k8sClient.CoreV1().ServiceAccounts(namespaceName).Get(ctx, foundRayCluster.Name, metav1.GetOptions{})).To(WithTransform(OwnerReferenceName, Equal(foundRayCluster.Name)))
				Expect(k8sClient.RbacV1().ClusterRoleBindings().Get(ctx, crbNameFromCluster(foundRayCluster), metav1.GetOptions{})).To(WithTransform(OwnerReferenceKind, Equal("RayCluster")))
				Expect(k8sClient.RbacV1().ClusterRoleBindings().Get(ctx, crbNameFromCluster(foundRayCluster), metav1.GetOptions{})).To(WithTransform(OwnerReferenceName, Equal(foundRayCluster.Name)))
				Expect(routeClient.RouteV1().Routes(namespaceName).Get(ctx, foundRayCluster.Name, metav1.GetOptions{})).To(WithTransform(OwnerReferenceKind, Equal("RayCluster")))
				Expect(routeClient.RouteV1().Routes(namespaceName).Get(ctx, foundRayCluster.Name, metav1.GetOptions{})).To(WithTransform(OwnerReferenceName, Equal(foundRayCluster.Name)))
			})

			It("should delete OAuth resources when annotation is removed", func(ctx SpecContext) {
				foundRayCluster, err := rayClient.RayV1().RayClusters(namespaceName).Get(ctx, rayClusterName, metav1.GetOptions{})
				Expect(err).To(Not(HaveOccurred()))

				// Use loop to remove annotation until it succeed
				Eventually(func() (*rayv1.RayCluster, error) {
					foundRayCluster, err := rayClient.RayV1().RayClusters(namespaceName).Get(ctx, rayClusterName, metav1.GetOptions{})
					if err != nil {
						return nil, err
					}
					delete(foundRayCluster.ObjectMeta.Annotations, "codeflare.dev/oauth")
					return rayClient.RayV1().RayClusters(namespaceName).Update(ctx, foundRayCluster, metav1.UpdateOptions{})
				}).WithTimeout(time.Second * 10).ShouldNot(BeNil())

				Eventually(func() error {
					_, err := k8sClient.CoreV1().Secrets(namespaceName).Get(ctx, oauthSecretNameFromCluster(foundRayCluster), metav1.GetOptions{})
					return err
				}).WithTimeout(time.Second * 10).Should(Satisfy(errors.IsNotFound))
				Eventually(func() error {
					_, err := k8sClient.CoreV1().Services(namespaceName).Get(ctx, oauthServiceNameFromCluster(foundRayCluster), metav1.GetOptions{})
					return err
				}).WithTimeout(time.Second * 10).Should(Satisfy(errors.IsNotFound))
				Eventually(func() error {
					_, err := k8sClient.CoreV1().ServiceAccounts(namespaceName).Get(ctx, foundRayCluster.Name, metav1.GetOptions{})
					return err
				}).WithTimeout(time.Second * 10).Should(Satisfy(errors.IsNotFound))

				// For some reason removal of the Route takes significant time on KinD, I suppose it is caused by Route CRD missing in KinD available schemas
				Eventually(func() error {
					_, err := routeClient.RouteV1().Routes(namespaceName).Get(ctx, foundRayCluster.Name, metav1.GetOptions{})
					return err
				}).WithTimeout(time.Second * 30).Should(Satisfy(errors.IsNotFound))
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
})

func OwnerReferenceKind(meta metav1.Object) string {
	return meta.GetOwnerReferences()[0].Kind
}

func OwnerReferenceName(meta metav1.Object) string {
	return meta.GetOwnerReferences()[0].Name
}
