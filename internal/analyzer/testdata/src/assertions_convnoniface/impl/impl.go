// Package impl provides the implementation type for testing conversion to non-interface.
package impl

// Task is a struct type (not an interface).
type Task struct{}

// Work is a method on Task.
func (Task) Work() {}
