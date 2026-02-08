// Package iface tests function-body assertions for assertions.scanfunctionbodies.
// This is the version WITHOUT want comment for tests expecting no diagnostics.
// When scanfunctionbodies=false, assertions inside function bodies should NOT be recognized.
package iface

import "ifaceguard-testdata/assertions_funcbody_nodiag/impl"

// Handler is the contract interface.
type Handler interface {
	Handle()
}

// initAssertions demonstrates var assertions inside a function body.
// This should only be recognized when scanfunctionbodies=true.
// With flag disabled, this should NOT trigger any diagnostic.
func initAssertions() {
	var _ Handler = (*impl.Processor)(nil)
}
