// Package correct demonstrates a proper assertion placement.
// The assertion is in the same package as the implementation type - no violation.
package correct

// Processor is the implementation type.
type Processor struct{}

// Process implements the ProcessorIface behavior.
func (Processor) Process() {}

// ProcessorIface is the contract interface.
type ProcessorIface interface {
	Process()
}

// Correct: assertion is in the same package as Processor.
// No violation expected because impl and assertion are in same package.
var _ ProcessorIface = (*Processor)(nil)
var _ ProcessorIface = new(Processor)
var _ ProcessorIface = Processor{}
var _ ProcessorIface = &Processor{}
