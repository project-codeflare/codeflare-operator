package e2e

import (
	"fmt"
	. "github.com/onsi/gomega"
	. "github.com/project-codeflare/codeflare-operator/test/support"
	mcadv1beta1 "github.com/project-codeflare/multi-cluster-app-dispatcher/pkg/apis/controller/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

var (
	ocmToken              string
	machinePoolsExist     bool
	numInitialNodePools   int
	numInitialMachineSets int
)

func TestInstascale(t *testing.T) {

	test := With(t)
	test.T().Parallel()

	namespace := test.NewTestNamespace()

	connection, err := CreateOCMConnection()
	if err != nil {
		test.T().Errorf("Unable to create ocm connection - Error : %v", err)
	}
	defer connection.Close()

	machinePoolsExist = true
	// check existing machine pools
	numInitialMachinePools, err := MachinePoolsCount(connection)
	if err != nil {
		test.T().Errorf("Unable to count machine pools - Error : %v", err)
	}

	if numInitialMachinePools == 0 {
		machinePoolsExist = false
		numInitialNodePools, err = NodePoolsCount(connection)
		if err != nil {
			test.T().Errorf("Unable to count node pools - Error : %v", err)
		}
		if numInitialNodePools == 0 {
			numInitialMachineSets, err = MachineSetsCount(connection)
			if err != nil {
				test.T().Errorf("Unable to count machine sets - Error : %v", err)
			}
		}
	}

	// create an appwrapper
	aw := &mcadv1beta1.AppWrapper{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-instascale",
			Namespace: namespace.Name,
			Labels: map[string]string{
				"orderedinstance": "m5.xlarge_g4dn.xlarge",
			},
		},
		Spec: mcadv1beta1.AppWrapperSpec{
			AggrResources: mcadv1beta1.AppWrapperResourceList{
				GenericItems: []mcadv1beta1.AppWrapperGenericResource{
					{
						CustomPodResources: []mcadv1beta1.CustomPodResourceTemplate{
							{
								Replicas: 1,
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("250m"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("1G"),
								},
							},
							{
								Replicas: 1,
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("250m"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("1G"),
								},
							},
						},
					},
				},
			},
		},
	}

	aw, err = test.Client().MCAD().McadV1beta1().AppWrappers(namespace.Name).Create(test.Ctx(), aw, metav1.CreateOptions{})
	test.Expect(err).NotTo(HaveOccurred())
	test.T().Logf("AppWrapper created successfully %s/%s", aw.Namespace, aw.Name)

	test.Eventually(AppWrapper(test, namespace, aw.Name), TestTimeoutMedium).
		Should(WithTransform(AppWrapperState, Equal(mcadv1beta1.AppWrapperStateActive)))

	if !machinePoolsExist {
		numNodePools, err := NodePoolsCount(connection)
		if err != nil {
			test.T().Errorf("Unable to count node pools - Error : %v", err)
		}
		fmt.Println(numNodePools)
		test.Expect(numNodePools).To(BeNumerically(">", numInitialNodePools))
		test.T().Logf("number of machine pools increased")

	} else if machinePoolsExist {
		numMachinePools, err := MachinePoolsCount(connection)
		if err != nil {
			test.T().Errorf("Unable to count machine pools - Error : %v", err)
		}
		fmt.Println(numMachinePools)
		test.Expect(numMachinePools).To(BeNumerically(">", numInitialMachinePools))
		test.T().Logf("number of machine pools increased")
	} else {
		numMachineSets, err := MachineSetsCount(connection)
		if err != nil {
			test.T().Errorf("Unable to count machine sets - Error : %v", err)
		}
		fmt.Println(numMachineSets)
		test.Expect(numMachineSets).To(BeNumerically(">", numInitialMachineSets))
		test.T().Logf("number of machine sets increased")
	}

	// TODO submit and check that the job has completed and that resources are released/scaled down
}
