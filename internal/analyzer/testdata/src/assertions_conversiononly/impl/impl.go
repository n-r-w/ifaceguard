// Package impl provides the implementation type for testing conversion-only form.
package impl

// Task is the implementation type.
type Task struct{}

// Work implements Worker interface.
func (Task) Work() {}
