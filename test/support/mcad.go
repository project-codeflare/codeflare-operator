package support

import (
	"github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	mcadv1beta1 "github.com/project-codeflare/multi-cluster-app-dispatcher/pkg/apis/controller/v1beta1"
)

func AppWrapper(t Test, namespace *corev1.Namespace, name string) func(g gomega.Gomega) *mcadv1beta1.AppWrapper {
	return func(g gomega.Gomega) *mcadv1beta1.AppWrapper {
		aw, err := t.Client().MCAD().ArbV1().AppWrappers(namespace.Name).Get(name, metav1.GetOptions{})
		g.Expect(err).NotTo(gomega.HaveOccurred())
		return aw
	}
}

func AppWrapperState(aw *mcadv1beta1.AppWrapper) mcadv1beta1.AppWrapperState {
	return aw.Status.State
}
