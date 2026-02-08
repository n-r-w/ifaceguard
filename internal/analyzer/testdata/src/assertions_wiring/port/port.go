// Package port defines the interface for wiring tests.
package port

// Querier is the contract interface.
type Querier interface {
	Query(q string) ([]string, error)
}
