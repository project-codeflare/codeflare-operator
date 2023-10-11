package e2e

import (
	"testing"

	. "github.com/onsi/gomega"
	mcadv1beta1 "github.com/project-codeflare/multi-cluster-app-dispatcher/pkg/apis/controller/v1beta1"

	"k8s.io/apimachinery/pkg/api/errors"

	. "github.com/project-codeflare/codeflare-operator/test/support"
)

func TestInstascaleMachineSet(t *testing.T) {
	test := With(t)
	test.T().Parallel()

	// skip test if not using machine sets
	ms, err := MachineSetsExist(test)
	if !ms || err != nil && errors.IsNotFound(err) {
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
	test.Expect(GetMachineSets(test)).Should(ContainElement(WithTransform(MachineSetId, Equal("test-instascale"))))
	// look for machine belonging to the machine set, there should be none
	test.Expect(GetMachines(test, "test-instascale")).Should(BeEmpty())

	// // Setup batch job and AppWrapper
	_, aw, err := createInstaScaleJobAppWrapper(test, namespace, cm)
	test.Expect(err).NotTo(HaveOccurred())

	// assert that AppWrapper goes to "Running" state
	test.Eventually(AppWrapper(test, namespace, aw.Name), TestTimeoutGpuProvisioning).
		Should(WithTransform(AppWrapperState, Equal(mcadv1beta1.AppWrapperStateActive)))

	// look for machine belonging to the machine set - expect to find it
	test.Eventually(Machines(test, "test-instascale"), TestTimeoutLong).Should(HaveLen(1))

	// assert that the AppWrapper goes to "Completed" state
	test.Eventually(AppWrapper(test, namespace, aw.Name), TestTimeoutMedium).
		Should(WithTransform(AppWrapperState, Equal(mcadv1beta1.AppWrapperStateCompleted)))

	// look for machine belonging to the machine set - there should be none
	test.Eventually(Machines(test, "test-instascale"), TestTimeoutLong).Should(BeEmpty())

}
