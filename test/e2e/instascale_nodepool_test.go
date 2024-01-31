package e2e

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	. "github.com/project-codeflare/codeflare-common/support"
	mcadv1beta1 "github.com/project-codeflare/multi-cluster-app-dispatcher/pkg/apis/controller/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInstascaleNodepool(t *testing.T) {

	test := With(t)
	test.T().Parallel()

	clusterType := GetClusterType(test)
	if clusterType != HypershiftCluster {
		test.T().Skipf("Skipping test as not running on an Hypershift cluster, resolved cluster type: %s", clusterType)
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

	// Setup batch job and AppWrapper
	aw := instaScaleJobAppWrapper(test, namespace, cm)

	expectedLabel := fmt.Sprintf("%s-%s", aw.Name, aw.Namespace)
	// check existing cluster resources
	// look for a node pool with a label key equal to aw.Name-aw.Namespace - expect NOT to find it
	test.Expect(GetNodePools(test, connection)).
		ShouldNot(ContainElement(WithTransform(NodePoolLabels, HaveKey(expectedLabel))))

	// apply AppWrapper to cluster
	_, err := test.Client().MCAD().WorkloadV1beta1().AppWrappers(namespace.Name).Create(test.Ctx(), aw, metav1.CreateOptions{})
	test.Expect(err).NotTo(HaveOccurred())
	test.T().Logf("AppWrapper created successfully %s/%s", aw.Namespace, aw.Name)

	// assert that AppWrapper goes to "Running" state
	test.Eventually(AppWrapper(test, namespace, aw.Name), TestTimeoutGpuProvisioning).
		Should(WithTransform(AppWrapperState, Equal(mcadv1beta1.AppWrapperStateActive)))

	// look for a node pool with a label key equal to aw.Name-aw.Namespace - expect to find it
	test.Eventually(NodePools(test, connection), TestTimeoutLong).
		Should(ContainElement(WithTransform(NodePoolLabels, HaveKey(expectedLabel))))

	// assert that the AppWrapper goes to "Completed" state
	test.Eventually(AppWrapper(test, namespace, aw.Name), TestTimeoutLong).
		Should(WithTransform(AppWrapperState, Equal(mcadv1beta1.AppWrapperStateCompleted)))

	// look for a node pool with a label key equal to aw.Name-aw.Namespace - expect NOT to find it
	test.Eventually(NodePools(test, connection), TestTimeoutLong).
		ShouldNot(ContainElement(WithTransform(NodePoolLabels, HaveKey(expectedLabel))))

}
