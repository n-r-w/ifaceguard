package impl

type Service struct{} // want "IFG003-ASSERTION-MISSING"

func (Service) FetchData() (string, error) {
	return "", nil
}
