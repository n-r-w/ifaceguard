// Package constructors_ignoreinterfaces verifies IFG005 ignoreinterfaces suppression.
package constructors_ignoreinterfaces

// Allowed is intentionally ignored by constructors.ignoreinterfaces in test config.
type Allowed interface {
	Run() error
}

// Service implements Allowed.
type Service struct{}

// Run implements Allowed.
func (Service) Run() error { return nil }

// NewAllowed returns an ignored interface and should not trigger IFG005.
func NewAllowed() Allowed {
	return Service{}
}
