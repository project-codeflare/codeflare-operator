package e2e

import (
	"testing"

	. "github.com/onsi/gomega"
	mcadv1beta1 "github.com/project-codeflare/multi-cluster-app-dispatcher/pkg/apis/controller/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/project-codeflare/codeflare-operator/test/support"
)

func TestInstascaleMachineSet(t *testing.T) {
	test := With(t)
	test.T().Parallel()

	// skip test if not using machine sets
	clusterType := GetClusterType(test)
	if clusterType != OcpCluster {
		test.T().Skipf("Skipping test as not running on an OCP cluster, resolved cluster type: %s", clusterType)
	}

	namespace := test.NewTestNamespace()

	// Test configuration
	cm := CreateConfigMap(test, namespace.Name, map[string][]byte{
		// pip requirements
		"requirements.txt": ReadFile(test, "mnist_pip_requirements.txt"),
		// MNIST training script
		"mnist.py": ReadFile(test, "mnist.py"),
	})

	// // Setup batch job and AppWrapper
	aw := instaScaleJobAppWrapper(test, namespace, cm)

	// look for machine set with aw name - expect to find it
	test.Expect(GetMachineSets(test)).Should(ContainElement(WithTransform(MachineSetId, Equal(aw.Name))))
	// look for machine belonging to the machine set, there should be none
	test.Expect(GetMachines(test, aw.Name)).Should(BeEmpty())

	// apply AppWrapper to cluster
	_, err := test.Client().MCAD().WorkloadV1beta1().AppWrappers(namespace.Name).Create(test.Ctx(), aw, metav1.CreateOptions{})
	test.Expect(err).NotTo(HaveOccurred())

	// assert that AppWrapper goes to "Running" state
	test.Eventually(AppWrapper(test, namespace, aw.Name), TestTimeoutGpuProvisioning).
		Should(WithTransform(AppWrapperState, Equal(mcadv1beta1.AppWrapperStateActive)))

	// look for machine belonging to the machine set - expect to find it
	test.Eventually(Machines(test, aw.Name), TestTimeoutLong).Should(HaveLen(1))

	// assert that the AppWrapper goes to "Completed" state
	test.Eventually(AppWrapper(test, namespace, aw.Name), TestTimeoutMedium).
		Should(WithTransform(AppWrapperState, Equal(mcadv1beta1.AppWrapperStateCompleted)))

	// look for machine belonging to the machine set - there should be none
	test.Eventually(Machines(test, aw.Name), TestTimeoutLong).Should(BeEmpty())

}
