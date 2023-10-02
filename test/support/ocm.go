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
	"fmt"
	"os"

	"github.com/onsi/gomega"
	ocmsdk "github.com/openshift-online/ocm-sdk-go"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateOCMConnection(test Test) *ocmsdk.Connection {
	instascaleOCMSecret, err := test.Client().Core().CoreV1().Secrets(GetInstaScaleOcmSecretNamespace()).Get(test.Ctx(), GetInstaScaleOcmSecretName(), metav1.GetOptions{})
	test.Expect(err).NotTo(gomega.HaveOccurred())

	ocmToken := string(instascaleOCMSecret.Data["token"])
	test.T().Logf("Retrieved Secret %s/%s successfully", instascaleOCMSecret.Namespace, instascaleOCMSecret.Name)

	connection, err := buildOCMConnection(ocmToken)
	test.Expect(err).NotTo(gomega.HaveOccurred())
	return connection
}

func buildOCMConnection(secret string) (*ocmsdk.Connection, error) {
	connection, err := ocmsdk.NewConnectionBuilder().
		Tokens(secret).
		Build()
	if err != nil || connection == nil {
		fmt.Fprintf(os.Stderr, "Can't build connection: %v\n", err)
		return nil, err
	}

	return connection, nil
}
