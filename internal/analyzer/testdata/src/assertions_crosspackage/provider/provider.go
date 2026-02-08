// Package provider contains the implementation type in a separate package hierarchy.
package provider

// Worker is the implementation type.
type Worker struct{}

// Work implements the Worker behavior.
func (Worker) Work() {}
