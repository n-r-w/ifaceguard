// Package iface tests conversion-only form assertions for assertions.acceptconversiononlyform.
// This is the version WITHOUT want comment for tests expecting no diagnostics.
// When acceptconversiononlyform=false, conversion-only assertions should NOT be recognized.
package iface

import "ifaceguard-testdata/assertions_conversiononly_nodiag/impl"

// Worker is the contract interface.
type Worker interface {
	Work()
}

// CONVERSION-ONLY FORM: var _ = I(expr) without explicit type annotation.
// This is a conversion-only form (no ValueSpec.Type), should only be recognized
// when acceptconversiononlyform=true.
// With flag disabled, this should NOT trigger any diagnostic.
var _ = Worker((*impl.Task)(nil))
