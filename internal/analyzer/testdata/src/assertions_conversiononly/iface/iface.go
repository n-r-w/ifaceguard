// Package iface tests conversion-only form assertions for assertions.acceptconversiononlyform.
// When acceptconversiononlyform=false, conversion-only assertions should NOT be recognized.
// When acceptconversiononlyform=true, conversion-only assertions SHOULD be recognized and
// will trigger IFG002-ASSERTION-PLACEMENT if placed in wrong package.
package iface

import "ifaceguard-testdata/assertions_conversiononly/impl"

// Worker is the contract interface.
type Worker interface {
	Work()
}

// CONVERSION-ONLY FORM: var _ = I(expr) without explicit type annotation.
// This is a conversion-only form (no ValueSpec.Type), should only be recognized
// when acceptconversiononlyform=true.
// If recognized, this is a VIOLATION because impl.Task is declared in impl package.
var _ = Worker((*impl.Task)(nil)) // want "IFG002-ASSERTION-PLACEMENT"
