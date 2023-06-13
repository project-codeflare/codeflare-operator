package support

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	TestTimeoutShort  = 1 * time.Minute
	TestTimeoutMedium = 2 * time.Minute
	TestTimeoutLong   = 5 * time.Minute
)

var (
	ApplyOptions = metav1.ApplyOptions{FieldManager: "codeflare-test", Force: true}
)
