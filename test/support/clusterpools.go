package support

import (
	"context"
	"fmt"
	"os"
	"strings"

	ocmsdk "github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	mapiclientset "github.com/openshift/client-go/machine/clientset/versioned"
	"github.com/openshift/client-go/machine/listers/machine/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var (
	ClusterID     string = os.Getenv("CLUSTERID")
	machineClient mapiclientset.Interface
	msLister      v1beta1.MachineSetLister
	TestName      string = "test-instascale"
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

func MachinePoolsExist(connection *ocmsdk.Connection) (bool, error) {
	machinePools := connection.ClustersMgmt().V1().Clusters().Cluster(ClusterID).MachinePools()
	return machinePools != nil, nil
}

func NodePoolsExist(connection *ocmsdk.Connection) (bool, error) {
	nodePools := connection.ClustersMgmt().V1().Clusters().Cluster(ClusterID).NodePools()
	return nodePools != nil, nil
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

func MachineSetsCount() (numMachineSets int, err error) {
	machineSets, err := machineClient.MachineV1beta1().MachineSets(namespaceToList).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return 0, fmt.Errorf("error while listing machine sets, error: %v", err)
	}
	machineSetsSize := machineSets.ListMeta.Size()

	return machineSetsSize, nil
}

func CheckMachineSets(awName string) (foundMachineSet bool, err error) {
	machineSets, err := msLister.MachineSets("").List(labels.Everything())
	if err != nil {
		return false, fmt.Errorf("error listing machine sets, error: %v", err)
	}
	for _, machineSet := range machineSets {
		if strings.Contains(machineSet.Name, awName) {
			foundMachineSet = true
		}
	}
	return foundMachineSet, err
}
