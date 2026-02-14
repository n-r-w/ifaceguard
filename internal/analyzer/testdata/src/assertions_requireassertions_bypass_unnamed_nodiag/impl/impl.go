package impl

import "ifaceguard-testdata/assertions_requireassertions_bypass_unnamed_nodiag/contract"

type Service struct{}

func (Service) Run() error { return nil }

var _ interface{ Run() error } = (*Service)(nil)

func Use(r contract.Runner) {
	_ = r
}
