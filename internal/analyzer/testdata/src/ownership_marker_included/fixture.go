// Package ownership_marker_included verifies marker interfaces are checked when enabled.
package ownership_marker_included

// Marker is a marker interface (no methods).
type Marker interface { // want `IFG001-OWNERSHIP: interface ifaceguard-testdata/ownership_marker_included\.Marker has implementation ifaceguard-testdata/ownership_marker_included\.MarkerImpl`
}

// MarkerImpl implements Marker.
type MarkerImpl struct{}

// NewMarker returns Marker, making it contractual under exportedoutput mode.
func NewMarker() Marker {
	return MarkerImpl{}
}
