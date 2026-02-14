// Package ownership_ignoreinterfaces verifies ignoreinterfaces suppression.
package ownership_ignoreinterfaces

// Ignored is an exported interface that should be ignored by config.
type Ignored interface {
	Execute() error
}

// IgnoredImpl implements Ignored.
type IgnoredImpl struct{}

func (IgnoredImpl) Execute() error { return nil }

// NewIgnored returns Ignored, making it contractual under exportedoutput mode.
func NewIgnored() Ignored { // want `IFG005-CONSTRUCTOR-INTERFACE-RETURN`
	return IgnoredImpl{}
}
