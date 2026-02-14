// Package ownership_expout_nested tests ownership exportedoutput mode:
// interface nested in exported output via Contains() predicate.
package ownership_expout_nested

// Processor is an interface that appears nested in exported function result.
// With contractscope=exportedoutput, this IS contractual via Contains() predicate.
type Processor interface { // want `IFG001-OWNERSHIP: interface "Processor" has implementation "BasicProcessor" in same package; move interface to consumer package or dedicated contract package`
	Process() error
}

// BasicProcessor implements Processor.
type BasicProcessor struct{}

// Process implements Processor.
func (BasicProcessor) Process() error { return nil }

// Result contains Processor in a struct field.
type Result struct {
	// Proc is a nested interface reference.
	Proc Processor
}

// GetResult returns a struct containing Processor.
// Interface appears nested via Contains(Processor, Result) == true.
// This makes Processor contractual under exportedoutput mode.
func GetResult() Result {
	return Result{Proc: BasicProcessor{}}
}

// GetProcessors returns a slice of Processor.
// Interface appears nested via Contains(Processor, []Processor) == true.
func GetProcessors() []Processor {
	return []Processor{BasicProcessor{}}
}
