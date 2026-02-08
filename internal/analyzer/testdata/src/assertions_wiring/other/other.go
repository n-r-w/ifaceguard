// Package other is NOT in wiringpackages allowlist, so assertions here are violations.
package other

import (
	"ifaceguard-testdata/assertions_wiring/port"
	"ifaceguard-testdata/assertions_wiring/service"
)

// VIOLATION: this package is not in wiringpackages allowlist.
var _ port.Querier = (*service.Database)(nil) // want "IFG002-ASSERTION-PLACEMENT"
