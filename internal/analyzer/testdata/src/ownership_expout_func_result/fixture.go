// Package ownership_expout_func_result tests ownership exportedoutput mode:
// interface appears in exported function result.
package ownership_expout_func_result

// Runner is an interface that appears in exported function result.
// With contractscope=exportedoutput, this IS contractual.
type Runner interface { // want `IFG001-OWNERSHIP: interface ifaceguard-testdata/ownership_expout_func_result\.Runner has implementation ifaceguard-testdata/ownership_expout_func_result\.Service`
	Run() error
}

// Service implements Runner.
type Service struct{}

// Run implements Runner.
func (Service) Run() error { return nil }

// NewRunner returns a Runner - the interface appears in exported function result.
// This makes Runner contractual under exportedoutput mode.
func NewRunner() Runner {
	return Service{}
}
