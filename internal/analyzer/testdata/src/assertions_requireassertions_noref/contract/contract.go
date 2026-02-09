package contract

type ExternalAPI interface {
	FetchData() (string, error)
}
