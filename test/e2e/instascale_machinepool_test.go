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

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"

	. "github.com/project-codeflare/codeflare-operator/test/support"
)

func TestInstascaleMachinePool(t *testing.T) {

	test := With(t)
	test.T().Parallel()

	namespace := test.NewTestNamespace()

	// Test configuration
	testConfigData := map[string][]byte{
		// pip requirements
		"requirements.txt": ReadFile(test, "mnist_pip_requirements.txt"),
		// MNIST training script
		"mnist.py": ReadFile(test, "mnist.py"),
	}
	cm := CreateConfigMap(test, namespace.Name, testConfigData)

	//create OCM connection
	connection := CreateOCMConnection(test)

	defer connection.Close()

	// check existing cluster machine pool resources
	// look for machine pool with aw name - expect not to find it
	test.Expect(GetMachinePools(test, connection)).
		ShouldNot(ContainElement(WithTransform(MachinePoolId, Equal("test-instascale-g4dn-xlarge"))))

	// Setup batch job and AppWrapper
	job, aw, err := createInstaScaleJobAppWrapper(test, namespace, cm)
	test.Expect(err).NotTo(HaveOccurred())

	// look for machine pool with aw name - expect to find it
	test.Eventually(MachinePools(test, connection), TestTimeoutLong).
		Should(ContainElement(WithTransform(MachinePoolId, Equal("test-instascale-g4dn-xlarge"))))

	// Assert that the job has completed
	test.T().Logf("Waiting for Job %s/%s to complete", job.Namespace, job.Name)
	test.Eventually(Job(test, job.Namespace, job.Name), TestTimeoutLong).Should(
		Or(
			WithTransform(ConditionStatus(batchv1.JobComplete), Equal(corev1.ConditionTrue)),
			WithTransform(ConditionStatus(batchv1.JobFailed), Equal(corev1.ConditionTrue)),
		))

	// Assert the job has completed successfully
	test.Expect(GetJob(test, job.Namespace, job.Name)).
		To(WithTransform(ConditionStatus(batchv1.JobComplete), Equal(corev1.ConditionTrue)))

	test.Eventually(AppWrapper(test, namespace, aw.Name), TestTimeoutShort).
		Should(WithTransform(AppWrapperState, Equal(mcadv1beta1.AppWrapperStateCompleted)))

	// look for machine pool with aw name - expect not to find it
	test.Eventually(MachinePools(test, connection), TestTimeoutLong).
		ShouldNot(ContainElement(WithTransform(MachinePoolId, Equal("test-instascale-g4dn-xlarge"))))

}
