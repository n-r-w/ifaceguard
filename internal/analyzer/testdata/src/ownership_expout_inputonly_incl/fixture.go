// Package ownership_expout_inputonly_incl tests ownership exportedoutput mode
// with skipifusedasinput=false: interface appears ONLY in input params.
// Diagnostic expected because skipifusedasinput=false treats input params as exported output.
package ownership_expout_inputonly_incl

// Validator is an interface that appears ONLY in exported function input params.
// With skipifusedasinput=false, it IS contractual (input params count as exported output).
type Validator interface { // want `IFG001-OWNERSHIP: interface ifaceguard-testdata/ownership_expout_inputonly_incl\.Validator has implementation ifaceguard-testdata/ownership_expout_inputonly_incl\.StrictValidator`
	Validate(data []byte) error
}

// StrictValidator implements Validator.
type StrictValidator struct{}

// Validate implements Validator.
func (StrictValidator) Validate(data []byte) error { return nil }

// ValidateData takes Validator as input but does NOT return it.
// Interface appears only in input params.
// With skipifusedasinput=false, diagnostic expected.
func ValidateData(v Validator, data []byte) error {
	return v.Validate(data)
}
