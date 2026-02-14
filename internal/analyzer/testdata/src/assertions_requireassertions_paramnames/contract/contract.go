// Package contract defines a contractual interface with named parameters.
package contract

import "context"

type Runner interface {
	Run(ctx context.Context) error
}
