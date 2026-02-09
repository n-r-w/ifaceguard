package impl

type Service struct{} // want "IFG003-ASSERTION-MISSING"

func (Service) StrictFetchData() (string, error) {
	return "", nil
}
