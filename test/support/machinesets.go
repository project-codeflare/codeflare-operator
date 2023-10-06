package support

import (
	"github.com/onsi/gomega"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1beta1 "github.com/openshift/api/machine/v1beta1"
)

func MachineSets(t Test) ([]v1beta1.MachineSet, error) {
	ms, err := t.Client().Machine().MachineV1beta1().MachineSets("openshift-machine-api").List(t.Ctx(), v1.ListOptions{})
	t.Expect(err).NotTo(gomega.HaveOccurred())
	return ms.Items, err
}

func MachineExists(t Test) (foundReplica bool) {
	machineSetList, err := MachineSets(t)
	t.Expect(err).NotTo(gomega.HaveOccurred())
	for _, ms := range machineSetList {
		if ms.Name == "test-instascale" && &ms.Status.AvailableReplicas == Ptr(int32(1)) {
			foundReplica = true
		}
	}
	return foundReplica
}

func MachineSetId(machineSet v1beta1.MachineSet) string {
	return machineSet.Name
}

func MachineSetsExist(t Test) bool {
	ms, err := MachineSets(t)
	t.Expect(err).NotTo(gomega.HaveOccurred())
	return ms != nil
}
