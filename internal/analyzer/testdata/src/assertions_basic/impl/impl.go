// Package impl provides the implementation type for assertions tests.
// This is the correct package for assertions - assertions should be here.
package impl

// Service is the implementation type.
type Service struct{}

// Run implements the Runner behavior.
func (Service) Run() error { return nil }
