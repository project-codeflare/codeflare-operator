package support

import (
	"github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
)

func GetMachineSets(t Test) ([]machinev1beta1.MachineSet, error) {
	ms, err := t.Client().Machine().MachineV1beta1().MachineSets("openshift-machine-api").List(t.Ctx(), metav1.ListOptions{})
	t.Expect(err).NotTo(gomega.HaveOccurred())
	return ms.Items, err
}

func Machines(t Test, machineSetName string) func(g gomega.Gomega) []machinev1beta1.Machine {
	return func(g gomega.Gomega) []machinev1beta1.Machine {
		machine, err := t.Client().Machine().MachineV1beta1().Machines("openshift-machine-api").List(t.Ctx(), metav1.ListOptions{LabelSelector: "machine.openshift.io/cluster-api-machineset=" + machineSetName})
		g.Expect(err).NotTo(gomega.HaveOccurred())
		return machine.Items
	}
}

func GetMachines(t Test, machineSetName string) []machinev1beta1.Machine {
	t.T().Helper()
	return Machines(t, machineSetName)(t)
}

func MachineSetId(machineSet machinev1beta1.MachineSet) string {
	return machineSet.Name
}
