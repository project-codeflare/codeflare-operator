package support

import (
	"context"
	"fmt"
	"os"

	ocmsdk "github.com/openshift-online/ocm-sdk-go"
	mapiclientset "github.com/openshift/client-go/machine/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	ClusterID         string = os.Getenv("CLUSTERID")
	machineClient     mapiclientset.Interface
)

const (
	namespaceToList = "openshift-machine-api"
)

func CreateOCMConnection(secret string) (*ocmsdk.Connection, error) {
	logger, err := ocmsdk.NewGoLoggerBuilder().
		Debug(false).
		Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't build logger: %v\n", err)
		os.Exit(1)
	}
	connection, err := ocmsdk.NewConnectionBuilder().
		Logger(logger).
		Tokens(string(secret)).
		Build()
	if err != nil || connection == nil {
		fmt.Fprintf(os.Stderr, "Can't build connection: %v\n", err)
		os.Exit(1)
	}

	return connection, nil
}


func MachinePoolsCount(connection *ocmsdk.Connection) (numMachinePools int, err error) {
	machinePoolsConnection := connection.ClustersMgmt().V1().Clusters().Cluster(ClusterID).MachinePools().List()
	machinePoolsListResponse, err := machinePoolsConnection.Send()
	if err != nil {
		return 0, fmt.Errorf("unable to send request, error: %v", err)
	}
	machinePoolsList := machinePoolsListResponse.Items()
	numMachinePools = machinePoolsList.Len()

	return numMachinePools, nil
}

func NodePoolsCount(connection *ocmsdk.Connection) (numNodePools int, err error) {
	nodePoolsConnection := connection.ClustersMgmt().V1().Clusters().Cluster(ClusterID).NodePools().List()
	nodePoolsListResponse, err := nodePoolsConnection.SendContext(context.Background())
	if err != nil {
		return 0, fmt.Errorf("unable to send request, error: %v", err)
	}
	nodePoolsList := nodePoolsListResponse.Items()
	numNodePools = nodePoolsList.Len()
	fmt.Println(numNodePools)

	return numNodePools, nil
}

func MachineSetsCount() (numMachineSets int, err error) {
	machineSets, err := machineClient.MachineV1beta1().MachineSets(namespaceToList).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return 0, fmt.Errorf("error while listing machine sets, error: %v", err)
	}
	machineSetsSize := machineSets.ListMeta.Size()

	return machineSetsSize, nil
}
