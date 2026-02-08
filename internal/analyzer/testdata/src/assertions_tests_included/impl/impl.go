// Package impl provides the implementation type for test-file assertions.
package impl

// Cache is the implementation type.
type Cache struct{}

// Get implements the CacheGetter behavior.
func (Cache) Get(key string) (string, bool) { return "", false }
