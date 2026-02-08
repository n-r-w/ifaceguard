// Package ownership_ignorefile tests global exclude.files masks.
package ownership_ignorefile

// ExternalAPI is a contractual interface (exported and returned by an exported function).
type ExternalAPI interface {
	FetchData() (string, error)
}

// NewExternalAPI returns ExternalAPI, making it contractual under exportedoutput mode.
func NewExternalAPI() ExternalAPI {
	return MockExternalAPI{}
}
