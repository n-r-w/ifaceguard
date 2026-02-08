// Package iface tests all supported RHS assertion forms for the assertions rule.
// All assertions here are VIOLATIONS because Server is in impl package.
package iface

import "ifaceguard-testdata/assertions_forms/impl"

// Executor is the contract interface.
type Executor interface {
	Execute()
}

// Test (*T)(nil) form - pointer conversion to nil.
var _ Executor = (*impl.Server)(nil) // want "IFG002-ASSERTION-PLACEMENT"

// Test new(T) form - pointer via new.
var _ Executor = new(impl.Server) // want "IFG002-ASSERTION-PLACEMENT"

// Test T{} form - value composite literal.
var _ Executor = impl.Server{} // want "IFG002-ASSERTION-PLACEMENT"

// Test &T{} form - pointer to composite literal.
var _ Executor = &impl.Server{} // want "IFG002-ASSERTION-PLACEMENT"
