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
	"context"
	"math/rand"
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

func stringInList(l []string, s string) bool {
	for _, i := range l {
		if i == s {
			return true
		}
	}
	return false
}

var letters = []rune("abcdefghijklmnopqrstuvwxyz")
var r = rand.New(rand.NewSource(time.Now().UnixNano()))

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[r.Intn(len(letters))]
	}
	return string(b)
}

var _ = Describe("RayCluster controller", func() {
	Context("RayCluster controller test", func() {
		var rayClusterName = "test-raycluster"
		var typeNamespaceName types.NamespacedName
		ctx := context.Background()
		BeforeEach(func() {
			By("Generate random number so each run is creating unique")
			rString := randSeq(10)
			rayClusterName = rayClusterName + "-" + rString
			typeNamespaceName = types.NamespacedName{Name: rayClusterName, Namespace: rayClusterName}
			By("Creating a namespace for running the tests.")
			namespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: rayClusterName,
				},
			}
			var err error
			err = k8sClient.Create(ctx, namespace)
			Expect(err).To(Not(HaveOccurred()))

			By("creating a basic instance of the RayCluster CR")
			raycluster := &rayv1.RayCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      rayClusterName,
					Namespace: rayClusterName,
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
			err = k8sClient.Get(ctx, typeNamespaceName, &rayv1.RayCluster{})
			Expect(errors.IsNotFound(err)).To(Equal(true))
			err = k8sClient.Create(ctx, raycluster)
			Expect(err).To(Not(HaveOccurred()))
		})

		AfterEach(func() {
			By("removing the instance of the RayCluster used")
			// err := clientSet.CoreV1().Namespaces().Delete(ctx, RayClusterName, metav1.DeleteOptions{})
			foundRayCluster := rayv1.RayCluster{}
			err := k8sClient.Get(ctx, typeNamespaceName, &foundRayCluster)
			if err != nil {
				Expect(errors.IsNotFound(err)).To(Equal(true))
			} else {
				Expect(err).To(Not(HaveOccurred()))
				_ = k8sClient.Delete(ctx, &foundRayCluster)
			}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, typeNamespaceName, &foundRayCluster)
				return errors.IsNotFound(err)
			}, SpecTimeout(time.Second*10)).Should(Equal(true))
		})

		It("should have oauth finalizer set", func() {
			foundRayCluster := rayv1.RayCluster{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, typeNamespaceName, &foundRayCluster)
				Expect(err).To(Not(HaveOccurred()))
				return stringInList(foundRayCluster.Finalizers, oAuthFinalizer)
			}, SpecTimeout(time.Second*10)).Should(Equal(true))
		})

		It("should create all oauth resources", func() {
			Eventually(func() error {
				foundRayCluster := rayv1.RayCluster{}
				err := k8sClient.Get(ctx, typeNamespaceName, &foundRayCluster)
				if err != nil {
					return err
				}
				err = k8sClient.Get(ctx, types.NamespacedName{Name: oauthSecretNameFromCluster(&foundRayCluster), Namespace: foundRayCluster.Namespace}, &corev1.Secret{})
				if err != nil {
					return err
				}
				err = k8sClient.Get(ctx, types.NamespacedName{Name: oauthServiceNameFromCluster(&foundRayCluster), Namespace: foundRayCluster.Namespace}, &corev1.Service{})
				if err != nil {
					return err
				}
				err = k8sClient.Get(ctx, types.NamespacedName{Name: foundRayCluster.Name, Namespace: foundRayCluster.Namespace}, &corev1.ServiceAccount{})
				if err != nil {
					return err
				}
				err = k8sClient.Get(ctx, types.NamespacedName{Name: crbNameFromCluster(&foundRayCluster)}, &rbacv1.ClusterRoleBinding{})
				if err != nil {
					return err
				}
				err = k8sClient.Get(ctx, types.NamespacedName{Name: foundRayCluster.Name, Namespace: foundRayCluster.Namespace}, &routev1.Route{})
				if err != nil {
					return err
				}
				return nil
			}, SpecTimeout(time.Second*10)).Should(Not(HaveOccurred()))
		})

		It("should set owner references for all resources", func() {
			foundRayCluster := rayv1.RayCluster{}
			err := k8sClient.Get(ctx, typeNamespaceName, &foundRayCluster)
			Expect(err).ToNot(HaveOccurred())
			err = k8sClient.Get(ctx, types.NamespacedName{Name: oauthSecretNameFromCluster(&foundRayCluster), Namespace: foundRayCluster.Namespace}, &corev1.Secret{})
			Expect(err).To(Not(HaveOccurred()))
			err = k8sClient.Get(ctx, types.NamespacedName{Name: oauthServiceNameFromCluster(&foundRayCluster), Namespace: foundRayCluster.Namespace}, &corev1.Service{})
			Expect(err).To(Not(HaveOccurred()))
			err = k8sClient.Get(ctx, types.NamespacedName{Name: foundRayCluster.Name, Namespace: foundRayCluster.Namespace}, &corev1.ServiceAccount{})
			Expect(err).To(Not(HaveOccurred()))
			err = k8sClient.Get(ctx, types.NamespacedName{Name: crbNameFromCluster(&foundRayCluster)}, &rbacv1.ClusterRoleBinding{})
			Expect(err).To(Not(HaveOccurred()))
			err = k8sClient.Get(ctx, types.NamespacedName{Name: foundRayCluster.Name, Namespace: foundRayCluster.Namespace}, &routev1.Route{})
			Expect(err).To(Not(HaveOccurred()))
		})

		It("should remove CRB when the RayCluster is deleted", func() {
			foundRayCluster := rayv1.RayCluster{}
			err := k8sClient.Get(ctx, typeNamespaceName, &foundRayCluster)
			Expect(err).To(Not(HaveOccurred()))
			err = k8sClient.Delete(ctx, &foundRayCluster)
			Expect(err).To(Not(HaveOccurred()))
			Eventually(func() bool {
				return errors.IsNotFound(k8sClient.Get(ctx, types.NamespacedName{Name: crbNameFromCluster(&foundRayCluster)}, &rbacv1.ClusterRoleBinding{}))
			}, SpecTimeout(time.Second*10)).Should(Equal(true))
		})
	})
})
