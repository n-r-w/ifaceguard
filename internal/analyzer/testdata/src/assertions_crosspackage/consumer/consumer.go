// Package consumer defines the interface and correctly imports the implementation.
package consumer

import "ifaceguard-testdata/assertions_crosspackage/provider"

// WorkerIface is the contract interface in consumer package.
type WorkerIface interface {
	Work()
}

// VIOLATION: assertion is in consumer package but Worker is in provider package.
// This tests the cross-package scenario where assertion is not in impl package.
var _ WorkerIface = (*provider.Worker)(nil) // want "IFG002-ASSERTION-PLACEMENT"

// VIOLATION: using new(T) form for cross-package test.
var _ WorkerIface = new(provider.Worker) // want "IFG002-ASSERTION-PLACEMENT"
