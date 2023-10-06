/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package support

import (
	"fmt"
	"os"

	"github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ocmsdk "github.com/openshift-online/ocm-sdk-go"
	cmv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

func CreateOCMConnection(test Test) *ocmsdk.Connection {
	secretNamespace, secretName := GetInstascaleOcmSecret()
	instascaleOCMSecret, err := test.Client().Core().CoreV1().Secrets(secretNamespace).Get(test.Ctx(), secretName, metav1.GetOptions{})
	test.Expect(err).NotTo(gomega.HaveOccurred())

	ocmToken := string(instascaleOCMSecret.Data["token"])
	test.T().Logf("Retrieved Secret %s/%s successfully", instascaleOCMSecret.Namespace, instascaleOCMSecret.Name)

	connection, err := buildOCMConnection(ocmToken)
	test.Expect(err).NotTo(gomega.HaveOccurred())
	return connection
}

func buildOCMConnection(secret string) (*ocmsdk.Connection, error) {
	connection, err := ocmsdk.NewConnectionBuilder().
		Tokens(secret).
		Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Can't build connection: %v\n", err)
		return nil, err
	}

	return connection, nil
}

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
