package impl

import "ifaceguard-testdata/assertions_requireassertions_generic/contract"

type Service[T any] struct{} // want "IFG003-ASSERTION-MISSING"

func (Service[T]) Run() error { return nil }

func Use(r contract.Runner) {
	_ = r
}
