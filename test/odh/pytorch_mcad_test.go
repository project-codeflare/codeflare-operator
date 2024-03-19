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

package odh

import (
	"testing"

	. "github.com/onsi/gomega"
	mcadv1beta2 "github.com/project-codeflare/appwrapper/api/v1beta2"
	. "github.com/project-codeflare/codeflare-common/support"

	rbacv1 "k8s.io/api/rbac/v1"
)

func TestMnistPyTorchMCAD(t *testing.T) {
	test := With(t)

	// Create a namespace
	namespace := test.NewTestNamespace()

	// Test configuration
	jupyterNotebookConfigMapFileName := "mnist_mcad_mini.ipynb"
	config := CreateConfigMap(test, namespace.Name, map[string][]byte{
		// MNIST MCAD Notebook
		jupyterNotebookConfigMapFileName: ReadFile(test, "resources/mnist_mcad_mini.ipynb"),
	})

	// Create RBAC, retrieve token for user with limited rights
	policyRules := []rbacv1.PolicyRule{
		{
			Verbs:     []string{"get", "create", "delete", "list", "patch", "update"},
			APIGroups: []string{mcadv1beta2.GroupVersion.Group},
			Resources: []string{"appwrappers"},
		},
		// Needed for job.logs()
		{
			Verbs:     []string{"get"},
			APIGroups: []string{""},
			Resources: []string{"pods/log"},
		},
	}
	sa := CreateServiceAccount(test, namespace.Name)
	role := CreateRole(test, namespace.Name, policyRules)
	CreateRoleBinding(test, namespace.Name, sa, role)
	token := CreateToken(test, namespace.Name, sa)

	// Create Notebook CR
	createNotebook(test, namespace, token, config.Name, jupyterNotebookConfigMapFileName)

	// Make sure the AppWrapper is created and running
	test.Eventually(AppWrappers(test, namespace.Name), TestTimeoutLong).
		Should(
			And(
				HaveLen(1),
				ContainElement(WithTransform(AppWrapperName, HavePrefix("mnistjob"))),
				ContainElement(WithTransform(AppWrapperPhase, Equal(mcadv1beta2.AppWrapperRunning))),
			),
		)

	// Make sure the AppWrapper finishes and is deleted
	test.Eventually(AppWrappers(test, namespace.Name), TestTimeoutLong).
		Should(HaveLen(0))
}
