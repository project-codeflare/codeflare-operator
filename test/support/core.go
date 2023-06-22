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
	"encoding/json"
	"io"

	"github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func Raw(t Test, obj runtime.Object) runtime.RawExtension {
	t.T().Helper()
	data, err := json.Marshal(obj)
	t.Expect(err).NotTo(gomega.HaveOccurred())
	return runtime.RawExtension{
		Raw: data,
	}
}

func GetPods(t Test, namespace *corev1.Namespace, options metav1.ListOptions) []corev1.Pod {
	t.T().Helper()
	pods, err := t.Client().Core().CoreV1().Pods(namespace.Name).List(t.Ctx(), options)
	t.Expect(err).NotTo(gomega.HaveOccurred())
	return pods.Items
}

func GetPodLogs(t Test, pod *corev1.Pod, options corev1.PodLogOptions) string {
	t.T().Helper()
	stream, err := t.Client().Core().CoreV1().Pods(pod.GetNamespace()).GetLogs(pod.GetName(), &options).Stream(t.Ctx())
	t.Expect(err).NotTo(gomega.HaveOccurred())

	defer func() {
		t.Expect(stream.Close()).To(gomega.Succeed())
	}()

	bytes, err := io.ReadAll(stream)
	t.Expect(err).NotTo(gomega.HaveOccurred())

	return string(bytes)
}
