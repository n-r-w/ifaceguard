package impl

import "ifaceguard-testdata/assertions_requireassertions_bypass_private_nodiag/contract"

type runner interface {
	Run() error
}

type Service struct{} // want "IFG003-ASSERTION-MISSING"

func (Service) Run() error { return nil }

var _ runner = (*Service)(nil)

func Use(r contract.Runner) {
	_ = r
}
