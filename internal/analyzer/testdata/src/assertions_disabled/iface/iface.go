// Package iface defines an assertion that should be ignored when assertions are disabled.
package iface

import "ifaceguard-testdata/assertions_disabled/impl"

// Runner is the contract interface.
type Runner interface {
	Run() error
}

// Assertion is in the wrong package, but assertions are disabled in the test.
var _ Runner = (*impl.Service)(nil)
