package support

import (
	"time"

	"github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
	"github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"
)

func init() {
	// Gomega settings
	gomega.SetDefaultEventuallyTimeout(TestTimeoutShort)
	gomega.SetDefaultEventuallyPollingInterval(1 * time.Second)
	gomega.SetDefaultConsistentlyDuration(30 * time.Second)
	gomega.SetDefaultConsistentlyPollingInterval(1 * time.Second)
	// Disable object truncation on test results
	format.MaxLength = 0
}

func EqualP(expected interface{}) types.GomegaMatcher {
	return gstruct.PointTo(gomega.Equal(expected))
}

func MatchFieldsP(options gstruct.Options, fields gstruct.Fields) types.GomegaMatcher {
	return gstruct.PointTo(gstruct.MatchFields(options, fields))
}
