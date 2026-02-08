// Package iface defines the interface for test-file assertions.
package iface

// CacheGetter is the contract interface.
type CacheGetter interface {
	Get(key string) (string, bool)
}
