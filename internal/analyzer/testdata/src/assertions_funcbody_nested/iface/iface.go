// Package iface tests that assertions inside nested func literals ARE scanned
// when scanfunctionbodies=true.
//
// Implementation note: scanFuncBodyForAssertions uses ast.Inspect which recursively
// visits ALL AST nodes including nested function literals. This test documents
// and locks that behavior.
package iface

import "ifaceguard-testdata/assertions_funcbody_nested/impl"

// Doer is the contract interface.
type Doer interface {
	Do()
}

// outerFunc contains a nested function literal with an assertion inside.
// When scanfunctionbodies=true, the nested func literal body IS scanned,
// so the assertion WILL be recognized and trigger IFG002-ASSERTION-PLACEMENT.
func outerFunc() {
	// Nested function literal.
	_ = func() {
		// Assertion inside nested func literal.
		// This SHOULD be recognized when scanfunctionbodies=true.
		var _ Doer = (*impl.Worker)(nil) // want "IFG002-ASSERTION-PLACEMENT"
	}
}
