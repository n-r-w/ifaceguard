package impl

type Service struct{}

func (Service) NoCoImportFetchData() (string, error) {
	return "", nil
}
