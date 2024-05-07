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

	"github.com/onsi/gomega"
	"github.com/project-codeflare/codeflare-common/support"

	"sigs.k8s.io/controller-runtime/pkg/client"
	kueuev1beta1 "sigs.k8s.io/kueue/apis/kueue/v1beta1"
)

//go:embed *.py *.txt *.sh
var files embed.FS

func ReadFile(t support.Test, fileName string) []byte {
	t.T().Helper()
	file, err := files.ReadFile(fileName)
	t.Expect(err).NotTo(gomega.HaveOccurred())
	return file
}

func AssignToLocalQueue(object client.Object, localqueue *kueuev1beta1.LocalQueue) {
	labels := object.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels["kueue.x-k8s.io/queue-name"] = localqueue.Name
	object.SetLabels(labels)
}
