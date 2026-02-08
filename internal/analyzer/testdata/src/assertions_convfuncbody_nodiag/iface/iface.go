// Package iface tests the interaction between acceptconversiononlyform and scanfunctionbodies.
// This is the version WITHOUT want comment for tests expecting no diagnostics.
// A conversion-only form assertion inside a function body requires BOTH flags to be true.
package iface

import "ifaceguard-testdata/assertions_convfuncbody_nodiag/impl"

// Streamer is the contract interface.
type Streamer interface {
	Stream()
}

// setupStreamer demonstrates conversion-only form assertion inside a function body.
// Requires BOTH acceptconversiononlyform=true AND scanfunctionbodies=true to be recognized.
// With either flag disabled, this should NOT trigger any diagnostic.
func setupStreamer() {
	_ = Streamer((*impl.Producer)(nil))
}
