// Package impl declares a type implementing an external contract using blank parameter names.
package impl

import "context"

type Service struct{} // want "IFG003-ASSERTION-MISSING"

func (Service) Run(_ context.Context) error { return nil }
