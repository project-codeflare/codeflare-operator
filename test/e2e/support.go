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
	"strings"

	"github.com/onsi/gomega"
	"github.com/project-codeflare/codeflare-common/support"

	"k8s.io/apimachinery/pkg/runtime"
)

//go:embed *.py *.txt *.sh
var files embed.FS

func ReadFile(t support.Test, fileName string) []byte {
	t.T().Helper()
	file, err := files.ReadFile(fileName)
	t.Expect(err).NotTo(gomega.HaveOccurred())
	return file
}

func RemoveCreationTimestamp(t support.Test, rawExtension runtime.RawExtension) runtime.RawExtension {
	t.T().Helper()
	patchedRaw := strings.ReplaceAll(string(rawExtension.Raw), `"metadata":{"creationTimestamp":null},`, "")
	patchedRaw = strings.ReplaceAll(patchedRaw, `"metadata":{"creationTimestamp":null,`, `"metadata":{`)
	patchedRaw = strings.ReplaceAll(patchedRaw, `"creationTimestamp":null,`, "")
	return runtime.RawExtension{
		Raw: []byte(patchedRaw),
	}
}
