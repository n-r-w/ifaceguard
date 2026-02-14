package impl

import "ifaceguard-testdata/assertions_requireassertions_private_reused/contract"

type runner interface {
	Run() error
}

type Service struct{} // want "IFG003-ASSERTION-MISSING"

func (Service) Run() error { return nil }

var _ runner = (*Service)(nil)

func UsePrivate(r runner) {
	_ = r
}

func Use(r contract.Runner) {
	_ = r
}
