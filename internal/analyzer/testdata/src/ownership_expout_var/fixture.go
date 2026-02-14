// Package ownership_expout_var tests ownership exportedoutput mode:
// interface appears in exported variable type.
package ownership_expout_var

// Handler is an interface that appears in exported variable type.
// With contractscope=exportedoutput, this IS contractual.
type Handler interface { // want `IFG001-OWNERSHIP: interface "Handler" has implementation "DefaultHandler" in same package; move interface to consumer package or dedicated contract package`
	Handle() error
}

// DefaultHandler implements Handler.
type DefaultHandler struct{}

// Handle implements Handler.
func (DefaultHandler) Handle() error { return nil }

// GlobalHandler is an exported variable with interface type.
// This makes Handler contractual under exportedoutput mode.
var GlobalHandler Handler = DefaultHandler{}
