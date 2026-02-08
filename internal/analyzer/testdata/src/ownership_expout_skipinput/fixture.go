// Package ownership_expout_skipinput tests ownership exportedoutput mode
// with skipifusedasinput=true (default): interface appears in both input AND output.
// Interface should be flagged because it appears in exported output.
package ownership_expout_skipinput

// Transformer is an interface that appears in both exported function input and output.
// With skipifusedasinput=true (default), it IS still contractual because it appears in output.
type Transformer interface { // want `IFG001-OWNERSHIP: interface ifaceguard-testdata/ownership_expout_skipinput\.Transformer has implementation ifaceguard-testdata/ownership_expout_skipinput\.IdentityTransformer`
	Transform(data []byte) ([]byte, error)
}

// IdentityTransformer implements Transformer.
type IdentityTransformer struct{}

// Transform implements Transformer.
func (IdentityTransformer) Transform(data []byte) ([]byte, error) { return data, nil }

// Process takes Transformer as input AND returns Transformer.
// Interface appears in both input params and result.
// With skipifusedasinput=true, the interface IS contractual because it appears in output.
func Process(t Transformer) Transformer {
	return t
}
