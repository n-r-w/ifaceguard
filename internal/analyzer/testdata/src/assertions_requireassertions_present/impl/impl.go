// Package impl declares a type implementing an external contract with an assertion.
package impl

import "ifaceguard-testdata/assertions_requireassertions_present/contract"

type Service struct{}

func (Service) Run() error { return nil }

var _ contract.Runner = (*Service)(nil)
