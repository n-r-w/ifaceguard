// Package constructors_enabled verifies IFG005 constructor return checks.
package constructors_enabled

// Runner is returned from constructor-like functions in this fixture.
type Runner interface {
	Run() error
}

// Service implements Runner.
type Service struct{}

// Run implements Runner.
func (Service) Run() error { return nil }

// NewRunner returns an interface and should trigger IFG005.
func NewRunner() Runner { // want `IFG005-CONSTRUCTOR-INTERFACE-RETURN: constructor "NewRunner" returns interface "Runner"; return concrete implementation type instead`
	return Service{}
}

// MustNewRunner returns an interface and should trigger IFG005.
func MustNewRunner() Runner { // want `IFG005-CONSTRUCTOR-INTERFACE-RETURN: constructor "MustNewRunner" returns interface "Runner"; return concrete implementation type instead`
	return Service{}
}

// NewService demonstrates that builtin error returns are ignored by default.
func NewService() (*Service, error) {
	return &Service{}, nil
}

// BuildRunner does not match default constructor patterns.
func BuildRunner() Runner {
	return Service{}
}
