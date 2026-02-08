// Package iface tests the interaction between acceptconversiononlyform and scanfunctionbodies.
// A conversion-only form assertion inside a function body requires BOTH flags to be true
// for the assertion to be recognized.
package iface

import "ifaceguard-testdata/assertions_convfuncbody/impl"

// Streamer is the contract interface.
type Streamer interface {
	Stream()
}

// setupStreamer demonstrates conversion-only form assertion inside a function body.
// Requires BOTH acceptconversiononlyform=true AND scanfunctionbodies=true to be recognized.
// If recognized, this is a VIOLATION because impl.Producer is declared in impl package.
func setupStreamer() {
	_ = Streamer((*impl.Producer)(nil)) // want "IFG002-ASSERTION-PLACEMENT"
}
