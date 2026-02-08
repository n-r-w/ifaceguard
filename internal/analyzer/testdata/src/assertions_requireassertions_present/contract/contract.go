// Package contract defines a contractual interface for requireassertions tests.
package contract

type Runner interface {
	Run() error
}
