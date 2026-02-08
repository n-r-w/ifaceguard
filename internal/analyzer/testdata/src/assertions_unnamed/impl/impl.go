// Package impl provides the implementation type for unnamed interface literal tests.
package impl

// Handler is the implementation type.
type Handler struct{}

// Handle implements the required behavior.
func (Handler) Handle() {}

// Process is another method for testing.
func (Handler) Process() error { return nil }
