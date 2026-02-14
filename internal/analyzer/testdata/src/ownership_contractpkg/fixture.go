// Package ownership_contractpkg verifies contract package allowlist behavior.
package ownership_contractpkg

// Repository is an exported contract interface used in exported output.
type Repository interface {
	Save() error
}

type repoImpl struct{}

func (repoImpl) Save() error { return nil }

// NewRepository returns Repository, making it contractual under exportedoutput mode.
func NewRepository() Repository { // want `IFG005-CONSTRUCTOR-INTERFACE-RETURN`
	return repoImpl{}
}
