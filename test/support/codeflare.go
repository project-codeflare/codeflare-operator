package support

import (
	"github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	codeflarev1alpha1 "github.com/project-codeflare/codeflare-operator/api/codeflare/v1alpha1"
)

func MCAD(t Test, namespace *corev1.Namespace, name string) func(g gomega.Gomega) *codeflarev1alpha1.MCAD {
	return func(g gomega.Gomega) *codeflarev1alpha1.MCAD {
		mcad, err := t.Client().CodeFlare().CodeflareV1alpha1().MCADs(namespace.Name).Get(t.Ctx(), name, metav1.GetOptions{})
		g.Expect(err).NotTo(gomega.HaveOccurred())
		return mcad
	}
}

func ReadyStatus(mcad *codeflarev1alpha1.MCAD) bool {
	return mcad.Status.Ready
}
