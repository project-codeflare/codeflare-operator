//go:build tools
// +build tools

package hack

// Add tools that hack scripts depend on here, to ensure they are vendored.
import (
	goimports "github.com/openshift-eng/openshift-goimports"
)
