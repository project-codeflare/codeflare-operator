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

package support

import (
	"github.com/onsi/gomega"
	mcadv1beta1 "github.com/project-codeflare/multi-cluster-app-dispatcher/pkg/apis/controller/v1beta1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func AppWrapper(t Test, namespace *corev1.Namespace, name string) func(g gomega.Gomega) *mcadv1beta1.AppWrapper {
	return func(g gomega.Gomega) *mcadv1beta1.AppWrapper {
		aw, err := t.Client().MCAD().WorkloadV1beta1().AppWrappers(namespace.Name).Get(t.Ctx(), name, metav1.GetOptions{})
		g.Expect(err).NotTo(gomega.HaveOccurred())
		return aw
	}
}

func AppWrapperState(aw *mcadv1beta1.AppWrapper) mcadv1beta1.AppWrapperState {
	return aw.Status.State
}
