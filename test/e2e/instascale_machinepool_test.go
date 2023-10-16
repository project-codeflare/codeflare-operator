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
	"testing"

	. "github.com/onsi/gomega"
	mcadv1beta1 "github.com/project-codeflare/multi-cluster-app-dispatcher/pkg/apis/controller/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/project-codeflare/codeflare-operator/test/support"
)

func TestInstascaleMachinePool(t *testing.T) {
	test := With(t)
	test.T().Parallel()

	clusterType := GetClusterType(test)
	if clusterType != OsdCluster {
		test.T().Skipf("Skipping test as not running on an OSD cluster, resolved cluster type: %s", clusterType)
	}

	namespace := test.NewTestNamespace()

	// Test configuration
	cm := CreateConfigMap(test, namespace.Name, map[string][]byte{
		// pip requirements
		"requirements.txt": ReadFile(test, "mnist_pip_requirements.txt"),
		// MNIST training script
		"mnist.py": ReadFile(test, "mnist.py"),
	})

	//create OCM connection
	connection := CreateOCMConnection(test)
	defer connection.Close()

	// check existing cluster machine pool resources
	// look for machine pool with aw name - expect not to find it
	test.Expect(GetMachinePools(test, connection)).
		ShouldNot(ContainElement(WithTransform(MachinePoolId, Equal("test-instascale-g4dn-xlarge"))))

	// Setup batch job and AppWrapper
	aw := instaScaleJobAppWrapper(test, namespace, cm)

	// apply AppWrapper to cluster
	_, err := test.Client().MCAD().WorkloadV1beta1().AppWrappers(namespace.Name).Create(test.Ctx(), aw, metav1.CreateOptions{})
	test.Expect(err).NotTo(HaveOccurred())
	test.T().Logf("AppWrapper created successfully %s/%s", aw.Namespace, aw.Name)

	// assert that AppWrapper goes to "Running" state
	test.Eventually(AppWrapper(test, namespace, aw.Name), TestTimeoutGpuProvisioning).
		Should(WithTransform(AppWrapperState, Equal(mcadv1beta1.AppWrapperStateActive)))

	// look for machine pool with aw name - expect to find it
	test.Eventually(MachinePools(test, connection), TestTimeoutLong).
		Should(ContainElement(WithTransform(MachinePoolId, Equal("test-instascale-g4dn-xlarge"))))

	test.Eventually(AppWrapper(test, namespace, aw.Name), TestTimeoutShort).
		Should(WithTransform(AppWrapperState, Equal(mcadv1beta1.AppWrapperStateCompleted)))

	// look for machine pool with aw name - expect not to find it
	test.Eventually(MachinePools(test, connection), TestTimeoutLong).
		ShouldNot(ContainElement(WithTransform(MachinePoolId, Equal("test-instascale-g4dn-xlarge"))))

}
