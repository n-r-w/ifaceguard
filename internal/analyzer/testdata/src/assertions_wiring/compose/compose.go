// Package compose is a wiring/compose package that should be in the allowlist.
// When assertions.wiringpackages contains this package path, NO diagnostics should be reported.
package compose

import (
	"ifaceguard-testdata/assertions_wiring/port"
	"ifaceguard-testdata/assertions_wiring/service"
)

// Assertions in wiring/compose packages are allowed.
// No violation expected because this package is in wiringpackages allowlist.
var (
	_ port.Querier = (*service.Database)(nil)
	_ port.Querier = &service.Database{}
)
