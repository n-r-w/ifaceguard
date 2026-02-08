// Package ownership_expout_method_result tests ownership exportedoutput mode:
// interface appears in exported method result.
package ownership_expout_method_result

// Worker is an interface that appears in exported method result.
// With contractscope=exportedoutput, this IS contractual.
type Worker interface { // want `IFG001-OWNERSHIP: interface ifaceguard-testdata/ownership_expout_method_result\.Worker has implementation ifaceguard-testdata/ownership_expout_method_result\.Task`
	Work() error
}

// Task implements Worker.
type Task struct{}

// Work implements Worker.
func (Task) Work() error { return nil }

// Factory produces workers.
type Factory struct{}

// CreateWorker returns a Worker - interface appears in exported method result.
// This makes Worker contractual under exportedoutput mode.
func (Factory) CreateWorker() Worker {
	return Task{}
}
