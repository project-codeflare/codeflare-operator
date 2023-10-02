package support

import (
	"context"
	"fmt"
	"os"
	"strings"

	ocmsdk "github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

var (
	ClusterID string = os.Getenv("CLUSTERID")
	TestName  string = "test-instascale"
)

func CreateOCMConnection(secret string) (*ocmsdk.Connection, error) {
	logger, err := ocmsdk.NewGoLoggerBuilder().
		Debug(false).
		Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't build logger: %v\n", err)
		return nil, err
	}
	connection, err := ocmsdk.NewConnectionBuilder().
		Logger(logger).
		Tokens(string(secret)).
		Build()
	if err != nil || connection == nil {
		fmt.Fprintf(os.Stderr, "Can't build connection: %v\n", err)
		return nil, err
	}

	return connection, nil
}

func CheckMachinePools(connection *ocmsdk.Connection, awName string) (foundMachinePool bool, err error) {
	machinePoolsConnection := connection.ClustersMgmt().V1().Clusters().Cluster(ClusterID).MachinePools().List()
	machinePoolsListResponse, err := machinePoolsConnection.Send()
	if err != nil {
		return false, fmt.Errorf("unable to send request, error: %v", err)
	}
	machinePoolsList := machinePoolsListResponse.Items()
	machinePoolsList.Range(func(index int, item *cmv1.MachinePool) bool {
		instanceName, _ := item.GetID()
		if strings.Contains(instanceName, awName) {
			foundMachinePool = true
		}
		return true
	})

	return foundMachinePool, err
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
