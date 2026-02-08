// Package ownership_expout_inputonly_nodiag tests ownership exportedoutput mode
// with skipifusedasinput=true (default): interface appears ONLY in input params.
// NO diagnostic expected because skipifusedasinput suppresses input-only usage.
package ownership_expout_inputonly_nodiag

// Validator is an interface that appears ONLY in exported function input params.
// With skipifusedasinput=true (default), it is NOT contractual.
// No diagnostic annotation - no diagnostic expected.
type Validator interface {
	Validate(data []byte) error
}

// StrictValidator implements Validator.
type StrictValidator struct{}

// Validate implements Validator.
func (StrictValidator) Validate(data []byte) error { return nil }

// ValidateData takes Validator as input but does NOT return it.
// Interface appears only in input params, NOT in results or exported vars.
// With skipifusedasinput=true, no diagnostic expected.
func ValidateData(v Validator, data []byte) error {
	return v.Validate(data)
}
