// Package assert tests unnamed interface literal as LHS for assertions.
// The interface type is defined inline, not as a named type.
package assert

import "ifaceguard-testdata/assertions_unnamed/impl"

// VIOLATION: assertion with unnamed interface literal LHS.
// The assertions rule allows LHS as `var _ interface{ ... } = (*T)(nil)`.
var _ interface{ Handle() } = (*impl.Handler)(nil) // want "IFG002-ASSERTION-PLACEMENT"

// VIOLATION: another unnamed interface literal form.
var _ interface { // want "IFG002-ASSERTION-PLACEMENT"
	Process() error
} = &impl.Handler{}
