package e2e

import (
	"testing"

	. "github.com/onsi/gomega"
	mcadv1beta1 "github.com/project-codeflare/multi-cluster-app-dispatcher/pkg/apis/controller/v1beta1"

	. "github.com/project-codeflare/codeflare-operator/test/support"
)

func TestInstascaleMachineSet(t *testing.T) {
	test := With(t)
	test.T().Parallel()

	// skip test if not using machine sets
	if !MachineSetsExist(test) {
		test.T().Skip("Skipping test as machine sets don't exist")
	}

	namespace := test.NewTestNamespace()

	// Test configuration
	cm := CreateConfigMap(test, namespace.Name, map[string][]byte{
		// pip requirements
		"requirements.txt": ReadFile(test, "mnist_pip_requirements.txt"),
		// MNIST training script
		"mnist.py": ReadFile(test, "mnist.py"),
	})

	// look for machine set with aw name - expect to find it
	test.Expect(MachineSets(test)).Should(ContainElement(WithTransform(MachineSetId, Equal("test-instascale"))))
	// look for machine set replica - expect not to find it
	test.Expect(MachineExists(test)).Should(BeFalse())

	// // Setup batch job and AppWrapper
	_, aw, err := createInstaScaleJobAppWrapper(test, namespace, cm)
	test.Expect(err).NotTo(HaveOccurred())

	// assert that AppWrapper goes to "Running" state
	test.Eventually(AppWrapper(test, namespace, aw.Name), TestTimeoutGpuProvisioning).
		Should(WithTransform(AppWrapperState, Equal(mcadv1beta1.AppWrapperStateActive)))

	//look for machine set replica - expect to find it
	test.Eventually(MachineExists(test), TestTimeoutLong).Should(BeTrue())

	// assert that the AppWrapper goes to "Completed" state
	test.Eventually(AppWrapper(test, namespace, aw.Name), TestTimeoutShort).
		Should(WithTransform(AppWrapperState, Equal(mcadv1beta1.AppWrapperStateCompleted)))

	// look for machine set replica - expect not to find it
	test.Eventually(MachineExists(test), TestTimeoutLong).Should(BeFalse())

}
