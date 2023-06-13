package support

import (
	"encoding/json"

	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"
)

func Raw(t Test, obj runtime.Object) runtime.RawExtension {
	t.T().Helper()
	data, err := json.Marshal(obj)
	t.Expect(err).NotTo(gomega.HaveOccurred())
	return runtime.RawExtension{
		Raw: data,
	}
}
