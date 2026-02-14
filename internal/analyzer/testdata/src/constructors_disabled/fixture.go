// Package constructors_disabled verifies IFG005 is silent when disabled.
package constructors_disabled

// Runner is returned from constructor-like functions in this fixture.
type Runner interface {
	Run() error
}

// Service implements Runner.
type Service struct{}

// Run implements Runner.
func (Service) Run() error { return nil }

// NewRunner would violate IFG005, but the rule is disabled in the test.
func NewRunner() Runner {
	return Service{}
}
