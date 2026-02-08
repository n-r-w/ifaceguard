package ownership_ignorefile

// MockExternalAPI is a mock implementation that should be excluded by file masks.
type MockExternalAPI struct{}

// FetchData implements ExternalAPI.
func (MockExternalAPI) FetchData() (string, error) {
	return "", nil
}
