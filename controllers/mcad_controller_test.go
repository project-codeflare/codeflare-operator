package controllers

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	mfc "github.com/manifestival/controller-runtime-client"
	mf "github.com/manifestival/manifestival"

	codeflarev1alpha1 "github.com/project-codeflare/codeflare-operator/api/codeflare/v1alpha1"
)

const (
	mcadCRCase1         = "./testdata/mcad_test_cases/case_1.yaml"
	mcadConfigMap1      = "./testdata/mcad_test_results/case_1/configmap.yaml"
	mcadService1        = "./testdata/mcad_test_results/case_1/service.yaml"
	mcadServiceAccount1 = "./testdata/mcad_test_results/case_1/serviceaccount.yaml"
	mcadCRCase2         = "./testdata/mcad_test_cases/case_2.yaml"
	mcadConfigMap2      = "./testdata/mcad_test_results/case_2/configmap.yaml"
	mcadService2        = "./testdata/mcad_test_results/case_2/service.yaml"
	mcadServiceAccount2 = "./testdata/mcad_test_results/case_2/serviceaccount.yaml"
)

func deployMCAD(ctx context.Context, path string, opts mf.Option) {
	mcad := &codeflarev1alpha1.MCAD{}
	err := convertToStructuredResource(path, mcad, opts)
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient.Create(ctx, mcad)).Should(Succeed())
}

var _ = Describe("The MCAD Controller", func() {
	client := mfc.NewClient(k8sClient)
	opts := mf.UseClient(client)
	ctx := context.Background()

	Context("In a namespace, when a blank MCAD Custom Resource is deployed", func() {

		It("It should create a configmap", func() {
			deployMCAD(ctx, mcadCRCase1, opts)
			compareConfigMaps(mcadConfigMap1, opts)
			compareServiceAccounts(mcadServiceAccount1, opts)
			compareServices(mcadService1, opts)
		})
	})

	Context("In a namespace, when a populated MCAD Custom Resource is deployed", func() {

		It("It should create a configmap", func() {
			deployMCAD(ctx, mcadCRCase2, opts)
			compareConfigMaps(mcadConfigMap2, opts)
			compareServiceAccounts(mcadServiceAccount2, opts)
			compareServices(mcadService2, opts)
		})
	})
})
