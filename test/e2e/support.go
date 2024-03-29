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

package e2e

import (
	"embed"
	"fmt"

	"github.com/onsi/gomega"
	"github.com/project-codeflare/codeflare-common/support"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
)

//go:embed *.py *.txt *.sh
var files embed.FS

func ReadFile(t support.Test, fileName string) []byte {
	t.T().Helper()
	file, err := files.ReadFile(fileName)
	t.Expect(err).NotTo(gomega.HaveOccurred())
	return file
}

func EnsureLocalQueue(t support.Test, namespace *corev1.Namespace) *kueue.LocalQueue {
	t.T().Helper()

	lq := &kueue.LocalQueue{
		TypeMeta:   metav1.TypeMeta{APIVersion: kueue.GroupVersion.String(), Kind: "LocalQueue"},
		ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("lq-%v", namespace.Name), Namespace: namespace.Name},
		Spec:       kueue.LocalQueueSpec{ClusterQueue: kueue.ClusterQueueReference("e2e-cluster-queue")},
	}
	localQueueResource := kueue.GroupVersion.WithResource("localqueues")
	lqMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(lq)
	t.Expect(err).NotTo(gomega.HaveOccurred())
	unstruct := unstructured.Unstructured{Object: lqMap}
	_, err = t.Client().Dynamic().Resource(localQueueResource).Namespace(namespace.Name).Create(t.Ctx(), &unstruct, metav1.CreateOptions{})
	t.Expect(client.IgnoreAlreadyExists(err)).NotTo(gomega.HaveOccurred())

	return lq
}
