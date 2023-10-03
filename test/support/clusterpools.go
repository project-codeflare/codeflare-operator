package support

import (
	"github.com/onsi/gomega"

	ocmsdk "github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

func MachinePools(t Test, connection *ocmsdk.Connection) func(g gomega.Gomega) []*cmv1.MachinePool {
	osdClusterId, found := GetOsdClusterId()
	t.Expect(found).To(gomega.BeTrue(), "OSD cluster id not found, please configure environment properly")

	return func(g gomega.Gomega) []*cmv1.MachinePool {
		machinePoolsListResponse, err := connection.ClustersMgmt().V1().Clusters().Cluster(osdClusterId).MachinePools().List().Send()
		g.Expect(err).NotTo(gomega.HaveOccurred())
		return machinePoolsListResponse.Items().Slice()
	}
}

func GetMachinePools(t Test, connection *ocmsdk.Connection) []*cmv1.MachinePool {
	t.T().Helper()
	return MachinePools(t, connection)(t)
}

func MachinePoolId(machinePool *cmv1.MachinePool) string {
	return machinePool.ID()
}
