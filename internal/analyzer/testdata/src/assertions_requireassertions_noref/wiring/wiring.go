package wiring

import (
	"ifaceguard-testdata/assertions_requireassertions_noref/contract"
	"ifaceguard-testdata/assertions_requireassertions_noref/impl"
)

func wire() contract.ExternalAPI {
	return &impl.Service{}
}
