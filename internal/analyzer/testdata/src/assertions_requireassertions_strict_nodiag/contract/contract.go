package contract

type ExternalAPI interface {
	NoCoImportFetchData() (string, error)
}
