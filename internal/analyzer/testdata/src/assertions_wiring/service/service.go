// Package service provides the implementation type for wiring package tests.
package service

// Database is the implementation type.
type Database struct{}

// Query implements the Querier behavior.
func (Database) Query(q string) ([]string, error) { return nil, nil }
