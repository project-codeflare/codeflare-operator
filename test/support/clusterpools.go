package support

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/onsi/gomega"
	ocmsdk "github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

var (
	ClusterID string = os.Getenv("CLUSTERID")
	TestName  string = "test-instascale"
)

func MachinePools(t Test, connection *ocmsdk.Connection) func(g gomega.Gomega) []*cmv1.MachinePool {
	return func(g gomega.Gomega) []*cmv1.MachinePool {
		machinePoolsListResponse, err := connection.ClustersMgmt().V1().Clusters().Cluster(ClusterID).MachinePools().List().Send()
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

func CheckNodePools(connection *ocmsdk.Connection, awName string) (foundNodePool bool, err error) {
	nodePoolsConnection := connection.ClustersMgmt().V1().Clusters().Cluster(ClusterID).NodePools().List()
	nodePoolsListResponse, err := nodePoolsConnection.SendContext(context.Background())
	if err != nil {
		return false, fmt.Errorf("unable to send request, error: %v", err)
	}
	nodePoolsList := nodePoolsListResponse.Items()
	nodePoolsList.Range(func(index int, item *cmv1.NodePool) bool {
		instanceName, _ := item.GetID()
		if strings.Contains(instanceName, awName) {
			foundNodePool = true
		}
		return true
	})

	return foundNodePool, err
}
