package testutil

import (
	"fmt"
	v1 "k8s.io/api/core/v1"
	"reflect"
)

func notEqualMsg(value string) {
	print(fmt.Sprintf("%s are not equal.", value))
}

func ConfigMapsAreEqual(expected v1.ConfigMap, actual v1.ConfigMap) bool {
	if expected.Name != actual.Name {
		notEqualMsg("Configmap Names are not equal.")
		return false
	}

	if !reflect.DeepEqual(expected.Data, actual.Data) {
		notEqualMsg("Configmap's Data values")
		return false
	}
	return true

}
