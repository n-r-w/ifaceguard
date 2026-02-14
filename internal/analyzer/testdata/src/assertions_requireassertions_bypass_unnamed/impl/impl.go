package impl

import "ifaceguard-testdata/assertions_requireassertions_bypass_unnamed/contract"

type Service struct{} // want "IFG003-ASSERTION-MISSING"

func (Service) Run() error { return nil }

var _ interface{ Run() error } = (*Service)(nil) // want "IFG004-ASSERTION-BYPASS"

func Use(r contract.Runner) {
	_ = r
}
