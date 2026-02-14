// Package ownership_disabled verifies ownership diagnostics are suppressed when disabled.
package ownership_disabled

// Service is a contractual interface used in exported output.
type Service interface {
	Run() error
}

type serviceImpl struct{}

func (serviceImpl) Run() error { return nil }

// NewService returns Service, making it contractual under exportedoutput mode.
func NewService() Service { // want `IFG005-CONSTRUCTOR-INTERFACE-RETURN`
	return serviceImpl{}
}
