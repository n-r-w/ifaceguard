// Package iface tests that conversion to non-interface type is NOT treated as assertion.
// acceptconversiononlyform recognizes "var _ = I(expr)" only when the
// conversion target type-checks to an interface. If the target is NOT an interface,
// the form should NOT be recognized as an assertion at all.
package iface

import "ifaceguard-testdata/assertions_convnoniface/impl"

// ConcreteTask is a concrete struct type (not an interface).
type ConcreteTask struct{}

// Work is a method on ConcreteTask.
func (ConcreteTask) Work() {}

// CONVERSION TO NON-INTERFACE: var _ = StructType(expr)
// This converts to a struct type, NOT an interface.
// Even with acceptconversiononlyform=true, this should NOT be recognized as an assertion
// because ConcreteTask is not an interface type.
// Therefore, NO IFG002-ASSERTION-PLACEMENT diagnostic should be reported.
var _ = ConcreteTask(impl.Task{})
