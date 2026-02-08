// Package ownership_ignoreimpl tests global exclude.types masks.
// The interface is contractual (appears in exported output) and implemented only by a mock.
package ownership_ignoreimpl

// ExternalAPI is a contractual interface (exported and returned by an exported function).
type ExternalAPI interface {
	FetchData() (string, error)
}

// MockExternalAPI is a mock implementation that should be ignored by ownership masks.
type MockExternalAPI struct{}

// FetchData implements ExternalAPI.
func (MockExternalAPI) FetchData() (string, error) {
	return "", nil
}

// NewExternalAPI returns ExternalAPI, making it contractual under exportedoutput mode.
func NewExternalAPI() ExternalAPI {
	return MockExternalAPI{}
}
