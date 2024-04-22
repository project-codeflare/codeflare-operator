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
	gomega "github.com/onsi/gomega"
	. "github.com/project-codeflare/codeflare-common/support"
	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMCADRay(t *testing.T) {
	test := With(t)

	// Create a namespace
	namespace := test.NewTestNamespace()

	// Test configuration
	jupyterNotebookConfigMapFileName := "mnist_ray_mini.ipynb"
	config := CreateConfigMap(test, namespace.Name, map[string][]byte{
		// MNIST Ray Notebook
		jupyterNotebookConfigMapFileName: ReadFile(test, "resources/mnist_ray_mini.ipynb"),
		"mnist.py":                       readMnistPy(test),
		"requirements.txt":               readRequirementsTxt(test),
	})

	// Create RBAC, retrieve token for user with limited rights
	policyRules := []rbacv1.PolicyRule{
		{
			Verbs:     []string{"get", "list"},
			APIGroups: []string{rayv1.GroupVersion.Group},
			Resources: []string{"rayclusters", "rayclusters/status"},
		},
		{
			Verbs:     []string{"get", "list"},
			APIGroups: []string{"route.openshift.io"},
			Resources: []string{"routes"},
		},
		{
			Verbs:     []string{"get", "list"},
			APIGroups: []string{"networking.k8s.io"},
			Resources: []string{"ingresses"},
		},
	}

	// Create cluster wide RBAC, required for SDK OpenShift check
	// TODO reevaluate once SDK change OpenShift detection logic
	clusterPolicyRules := []rbacv1.PolicyRule{
		{
			Verbs:         []string{"get", "list"},
			APIGroups:     []string{"config.openshift.io"},
			Resources:     []string{"ingresses"},
			ResourceNames: []string{"cluster"},
		},
	}

	sa := CreateServiceAccount(test, namespace.Name)
	role := CreateRole(test, namespace.Name, policyRules)
	CreateRoleBinding(test, namespace.Name, sa, role)
	clusterRole := CreateClusterRole(test, clusterPolicyRules)
	CreateClusterRoleBinding(test, sa, clusterRole)
	token := CreateToken(test, namespace.Name, sa)

	// Create Notebook CR
	createNotebook(test, namespace, token, config.Name, jupyterNotebookConfigMapFileName)

	// Make sure the RayCluster is created and running
	test.Eventually(rayClusters(test, namespace), TestTimeoutLong).
		Should(
			And(
				HaveLen(1),
				ContainElement(WithTransform(RayClusterState, Equal(rayv1.Ready))),
			),
		)

	// Make sure the RayCluster finishes and is deleted
	test.Eventually(rayClusters(test, namespace), TestTimeoutLong).
		Should(HaveLen(0))
}

func readRequirementsTxt(test Test) []byte {
	// Read the requirements.txt from resources and perform replacements for custom values using go template
	props := struct {
		PipIndexUrl    string
		PipTrustedHost string
	}{
		PipIndexUrl: "--index " + GetPipIndexURL(),
	}

	// Provide trusted host only if defined
	if len(GetPipTrustedHost()) > 0 {
		props.PipTrustedHost = "--trusted-host " + GetPipTrustedHost()
	}

	template, err := files.ReadFile("resources/requirements.txt")
	test.Expect(err).NotTo(HaveOccurred())

	return ParseTemplate(test, template, props)
}

func readMnistPy(test Test) []byte {
	// Read the mnist.py from resources and perform replacements for custom values using go template
	props := struct {
		MnistDatasetURL string
	}{
		MnistDatasetURL: GetMnistDatasetURL(),
	}
	template, err := files.ReadFile("resources/mnist.py")
	test.Expect(err).NotTo(HaveOccurred())

	return ParseTemplate(test, template, props)
}

// TODO: This belongs on codeflare-common/support/ray.go
func rayClusters(t Test, namespace *corev1.Namespace) func(g gomega.Gomega) []*rayv1.RayCluster {
	return func(g gomega.Gomega) []*rayv1.RayCluster {
		rcs, err := t.Client().Ray().RayV1().RayClusters(namespace.Name).List(t.Ctx(), metav1.ListOptions{})
		g.Expect(err).NotTo(gomega.HaveOccurred())

		rcsp := []*rayv1.RayCluster{}
		for _, v := range rcs.Items {
			rcsp = append(rcsp, &v)
		}

		return rcsp
	}
}
