// Package impl provides the implementation type for disabled assertions tests.
package impl

// Service is the implementation type.
type Service struct{}

// Run implements Runner.
func (Service) Run() error { return nil }
