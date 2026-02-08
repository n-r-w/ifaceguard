// Package iface declares the interface and incorrectly places assertions.
// Assertions for impl.Service should NOT be here - this triggers IFG002-ASSERTION-PLACEMENT.
package iface

import "ifaceguard-testdata/assertions_basic/impl"

// Runner is the contract interface.
type Runner interface {
	Run() error
}

// VIOLATION: assertion is in iface package but Service is declared in impl package.
var _ Runner = (*impl.Service)(nil) // want "IFG002-ASSERTION-PLACEMENT"
