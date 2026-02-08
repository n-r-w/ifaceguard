// Package iface tests function-body assertions for assertions.scanfunctionbodies.
// When scanfunctionbodies=false, assertions inside function bodies should NOT be recognized.
// When scanfunctionbodies=true, assertions inside function bodies SHOULD be recognized and
// will trigger IFG002-ASSERTION-PLACEMENT if placed in wrong package.
package iface

import "ifaceguard-testdata/assertions_funcbody/impl"

// Handler is the contract interface.
type Handler interface {
	Handle()
}

// initAssertions demonstrates var assertions inside a function body.
// This should only be recognized when scanfunctionbodies=true.
// If recognized, this is a VIOLATION because impl.Processor is declared in impl package.
func initAssertions() {
	var _ Handler = (*impl.Processor)(nil) // want "IFG002-ASSERTION-PLACEMENT"
}
