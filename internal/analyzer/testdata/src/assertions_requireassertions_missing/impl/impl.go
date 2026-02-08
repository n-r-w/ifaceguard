// Package impl declares a type implementing an external contract without an assertion.
package impl

import "ifaceguard-testdata/assertions_requireassertions_missing/contract"

type Service struct{} // want "IFG003-ASSERTION-MISSING"

func (Service) Run() error { return nil }

func Use(r contract.Runner) {
	_ = r
}
