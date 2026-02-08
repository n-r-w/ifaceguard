// This test file contains assertions that should trigger diagnostics.
package iface

import (
	"testing"

	"ifaceguard-testdata/assertions_tests_included/impl"
)

// VIOLATION: assertion in test file should be reported.
var _ CacheGetter = (*impl.Cache)(nil) // want "IFG002-ASSERTION-PLACEMENT"

// TestDummy is a placeholder test.
func TestDummy(t *testing.T) {
	t.Parallel()
}
