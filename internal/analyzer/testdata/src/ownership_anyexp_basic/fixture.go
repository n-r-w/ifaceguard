// Package ownership_anyexp_basic tests ownership anyexported mode:
// any exported interface is considered contractual.
package ownership_anyexp_basic

// Runner is an EXPORTED interface - with contractscope=anyexported, it IS contractual
// regardless of whether it appears in exported API surfaces.
type Runner interface { // want `IFG001-OWNERSHIP: interface ifaceguard-testdata/ownership_anyexp_basic\.Runner has implementation ifaceguard-testdata/ownership_anyexp_basic\.Service`
	Run() error
}

// Service implements Runner.
type Service struct{}

// Run implements Runner.
func (Service) Run() error { return nil }

// handler is an UNEXPORTED interface - with contractscope=anyexported,
// unexported interfaces are NOT contractual.
type handler interface {
	Handle() error
}

// handlerImpl implements handler.
type handlerImpl struct{}

// Handle implements handler.
func (handlerImpl) Handle() error { return nil }
