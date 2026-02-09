package contract

type ExternalAPI interface {
	StrictFetchData() (string, error)
}
