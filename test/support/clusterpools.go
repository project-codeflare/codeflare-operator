package support

import (
	"context"
	"fmt"
	"os"

	ocmsdk "github.com/openshift-online/ocm-sdk-go"
	mapiclientset "github.com/openshift/client-go/machine/clientset/versioned"
	"github.com/openshift/client-go/machine/listers/machine/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	ocmToken          string = os.Getenv("OCMTOKEN")
	ClusterID         string = os.Getenv("CLUSTERID")
	machinePoolsExist bool
	machineClient     mapiclientset.Interface
	msLister          v1beta1.MachineSetLister
)

const (
	namespaceToList = "openshift-machine-api"
)

func CreateOCMConnection() (*ocmsdk.Connection, error) {

	logger, err := ocmsdk.NewGoLoggerBuilder().
		Debug(false).
		Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't build logger: %v\n", err)
		os.Exit(1)
	}

	connection, err := ocmsdk.NewConnectionBuilder().
		Logger(logger).
		Tokens(ocmToken).
		Build()
	fmt.Println("connection", connection, err)
	if err != nil || connection == nil {
		fmt.Println("something went wrong", connection, err)
		fmt.Fprintf(os.Stderr, "Can't build connection: %v\n", err)
		os.Exit(1)
	}

	return connection, nil
}


func MachinePoolsCount(connection *ocmsdk.Connection) (numMachinePools int, err error) {
	fmt.Println("clusterID %v", ClusterID)
	machinePoolsConnection := connection.ClustersMgmt().V1().Clusters().Cluster(ClusterID).MachinePools().List()
	fmt.Println("machine pools connection %v", machinePoolsConnection)

	machinePoolsListResponse, err := machinePoolsConnection.Send()
	if err != nil {
		fmt.Println("machine pools list response, %v error, %v", machinePoolsListResponse, err)
		return 0, fmt.Errorf("unable to send request, error: %v", err)
	}
	machinePoolsList := machinePoolsListResponse.Items()
	fmt.Println("machine pool list %v", machinePoolsList)
	//check the current number of machine pools
	// TODO to be more precise could we check the machineTypes?
	numMachinePools = machinePoolsList.Len()
	fmt.Println(numMachinePools)

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

func MachineSetsCount(connection *ocmsdk.Connection) (numMachineSets int, err error) {
	machineSets, err := machineClient.MachineV1beta1().MachineSets(namespaceToList).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return 0, fmt.Errorf("error while listing machine sets, error: %v", err)
	}
	machineSetsSize := machineSets.ListMeta.Size()

	return machineSetsSize, nil
}
