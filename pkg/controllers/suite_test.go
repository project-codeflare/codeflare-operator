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

package controllers

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	routev1 "github.com/openshift/api/route/v1"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

const (
	RayClusterCRDFileDownload = "https://raw.githubusercontent.com/ray-project/kuberay/master/ray-operator/config/crd/bases/ray.io_rayclusters.yaml"
	RouteCRDFileDownload      = "https://raw.githubusercontent.com/openshift/api/master/route/v1/route.crd.yaml"
)

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	var err error
	var fRoute, fRaycluster *os.File

	By("Creating and downloading necessary crds")
	err = os.Mkdir("./test-crds", os.ModePerm)
	Expect(err).ToNot(HaveOccurred())
	fRoute, err = os.Create("./test-crds/route.yaml")
	Expect(err).ToNot(HaveOccurred())
	defer fRoute.Close()
	resp, err := http.Get(RouteCRDFileDownload)
	Expect(err).ToNot(HaveOccurred())
	_, err = io.Copy(fRoute, resp.Body)
	Expect(err).ToNot(HaveOccurred())
	fRaycluster, err = os.Create("./test-crds/raycluster.yaml")
	Expect(err).ToNot(HaveOccurred())
	defer fRaycluster.Close()
	resp, err = http.Get(RayClusterCRDFileDownload)
	Expect(err).ToNot(HaveOccurred())
	_, err = io.Copy(fRaycluster, resp.Body)
	Expect(err).ToNot(HaveOccurred())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "config", "crd"),
			filepath.Join(".", "test-crds"),
		},
		ErrorIfCRDPathMissing: true,
	}

	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	clientSet, err := kubernetes.NewForConfig(cfg)
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
	err = rayv1.AddToScheme(scheme.Scheme)
	Expect(err).To(Not(HaveOccurred()))
	err = routev1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).NotTo(HaveOccurred())
	err = (&RayClusterReconciler{
		Client:     k8sManager.GetClient(),
		Scheme:     k8sManager.GetScheme(),
		kubeClient: clientSet,
		CookieSalt: "foo",
	}).SetupWithManager(k8sManager)
	Expect(err).NotTo(HaveOccurred())
	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(context.Background())
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := os.RemoveAll("./test-crds")
	Expect(err).NotTo(HaveOccurred())
	err = testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
