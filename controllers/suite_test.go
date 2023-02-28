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
	"github.com/project-codeflare/codeflare-operator/controllers/util"
	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	mf "github.com/manifestival/manifestival"
	codeflarev1alpha1 "github.com/project-codeflare/codeflare-operator/api/v1alpha1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment
	ctx       context.Context
	cancel    context.CancelFunc
)

const (
	workingNamespace = "default"
	timeout          = time.Second * 30
	interval         = time.Millisecond * 10
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controllers Suite")
}

var _ = BeforeSuite(func() {
	ctx, cancel = context.WithCancel(context.TODO())

	// Initialize logger
	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.TimeEncoderOfLayout(time.RFC3339),
	}
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseFlagOptions(&opts)))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	// Register API objects
	utilruntime.Must(clientgoscheme.AddToScheme(scheme.Scheme))
	utilruntime.Must(codeflarev1alpha1.AddToScheme(scheme.Scheme))
	//+kubebuilder:scaffold:scheme

	// Initialize Kubernetes client
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// Setup controller manager
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme:             scheme.Scheme,
		LeaderElection:     false,
		MetricsBindAddress: "0",
	})
	Expect(err).NotTo(HaveOccurred())

	err = (&MCADReconciler{
		Client:        k8sClient,
		Log:           ctrl.Log.WithName("controllers").WithName("mcad-controller"),
		Scheme:        scheme.Scheme,
		TemplatesPath: "../config/internal/",
	}).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	// Start the manager
	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "Failed to run manager")
	}()

})

var _ = AfterSuite(func() {
	// Give some time to allow workers to gracefully shutdown
	time.Sleep(5 * time.Second)
	cancel()
	By("tearing down the test environment")
	time.Sleep(1 * time.Second)
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

// Cleanup resources to not contaminate between tests
var _ = AfterEach(func() {
	inNamespace := client.InNamespace(workingNamespace)
	Expect(k8sClient.DeleteAllOf(context.TODO(), &codeflarev1alpha1.MCAD{}, inNamespace)).ToNot(HaveOccurred())

})

func convertToStructuredResource(path string, out interface{}, opts mf.Option) error {
	m, err := mf.ManifestFrom(mf.Recursive(path), opts)
	if err != nil {
		return err
	}
	m, err = m.Transform(mf.InjectNamespace(workingNamespace))
	if err != nil {
		return err
	}
	if err != nil {
		return err
	}
	err = scheme.Scheme.Convert(&m.Resources()[0], out, nil)
	if err != nil {
		return err
	}
	return nil
}

func compareConfigMaps(path string, opts mf.Option) {
	expectedConfigMap := &corev1.ConfigMap{}
	Expect(convertToStructuredResource(path, expectedConfigMap, opts)).NotTo(HaveOccurred())

	actualConfigMap := &corev1.ConfigMap{}
	Eventually(func() error {
		namespacedNamed := types.NamespacedName{Name: expectedConfigMap.Name, Namespace: workingNamespace}
		return k8sClient.Get(ctx, namespacedNamed, actualConfigMap)
	}, timeout, interval).ShouldNot(HaveOccurred())

	Expect(util.ConfigMapsAreEqual(*expectedConfigMap, *actualConfigMap)).Should(BeTrue())
}

//func compareRoleBindings(path string, opts mf.Option) {
//	expectedRB := &k8srbacv1.RoleBinding{}
//	Expect(convertToStructuredResource(path, expectedRB, opts)).NotTo(HaveOccurred())
//	expectedRB.Subjects[0].Namespace = workingNamespace
//
//	actualRB := &k8srbacv1.RoleBinding{}
//	Eventually(func() error {
//		namespacedNamed := types.NamespacedName{Name: expectedRB.Name, Namespace: workingNamespace}
//		return k8sClient.Get(ctx, namespacedNamed, actualRB)
//	}, timeout, interval).ShouldNot(HaveOccurred())
//
//	Expect(util.RoleBindingsAreEqual(*expectedRB, *actualRB)).Should(BeTrue())
//}

func compareServiceAccounts(path string, opts mf.Option) {
	expectedSA := &corev1.ServiceAccount{}
	Expect(convertToStructuredResource(path, expectedSA, opts)).NotTo(HaveOccurred())
	expectedSA.Namespace = workingNamespace

	actualSA := &corev1.ServiceAccount{}
	Eventually(func() error {
		namespacedNamed := types.NamespacedName{Name: expectedSA.Name, Namespace: workingNamespace}
		return k8sClient.Get(ctx, namespacedNamed, actualSA)
	}, timeout, interval).ShouldNot(HaveOccurred())

	Expect(util.ServiceAccountsAreEqual(*expectedSA, *actualSA)).Should(BeTrue())
}

func compareServices(path string, opts mf.Option) {
	expectedService := &corev1.Service{}
	Expect(convertToStructuredResource(path, expectedService, opts)).NotTo(HaveOccurred())

	actualService := &corev1.Service{}
	Eventually(func() error {
		namespacedNamed := types.NamespacedName{Name: expectedService.Name, Namespace: workingNamespace}
		return k8sClient.Get(ctx, namespacedNamed, actualService)
	}, timeout, interval).ShouldNot(HaveOccurred())

	Expect(util.ServicesAreEqual(*expectedService, *actualService)).Should(BeTrue())
}
