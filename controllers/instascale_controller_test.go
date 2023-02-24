package controllers

import (
	"context"

	mfc "github.com/manifestival/controller-runtime-client"
	mf "github.com/manifestival/manifestival"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	codeflarev1alpha1 "github.com/project-codeflare/codeflare-operator/api/v1alpha1"
)

const (
	instascaleCRCase1             = "./testdata/instascale_test_cases/case_1.yaml"
	instascaleConfigMap1          = "./testdata/instascale_test_results/case_1/configmap.yaml"
	instascaleServiceAccount1     = "./testdata/instascale_test_results/case_1/serviceaccount.yaml"
	instascaleClusterRole1        = "./testdata/instascale_test_results/case_1/clusterrole.yaml"
	instascaleClusterRoleBinding1 = "./testdata/instascale_test_results/case_1/clusterrolebinding.yaml"
	instascaleDeployment1         = "./testdata/instascale_test_results/case_1/deployment.yaml"
)

func deployInstaScale(ctx context.Context, path string, opts mf.Option) {
	instascale := &codeflarev1alpha1.InstaScale{}
	err := convertToStructuredResource(path, instascale, opts)
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient.Create(ctx, instascale)).Should(Succeed())
}

var _ = Describe("The Instascale Controller", func() {
	client := mfc.NewClient(k8sClient)
	opts := mf.UseClient(client)
	ctx := context.Background()

	Context("In a namespace, when a blank InstaScale Custom Resource is deployed", func() {

		It("It should deploy InstaScale with default settings", func() {
			deployInstaScale(ctx, instascaleCRCase1, opts)
			compareConfigMaps(instascaleConfigMap1, opts)
			compareServiceAccounts(instascaleServiceAccount1, opts)
			compareDeployments(instascaleDeployment1, opts)
			compareClusterRoles(instascaleClusterRole1, opts)
			compareClusterRoleBindings(instascaleClusterRoleBinding1, opts)
		})
	})

})
