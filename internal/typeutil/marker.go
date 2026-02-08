package typeutil

import (
	"go/types"
)

// IsMarkerInterface checks if the given named type is a marker interface.
//
// A marker interface is defined as:
//   - It has zero explicit methods, AND
//   - It has zero embedded interfaces.
//
// Additionally, if the interface contains a type set (i.e., types.Interface.IsMethodSet() == false),
// it is NOT considered a marker interface, regardless of methods/embeds.
// This ensures constraint interfaces with type sets are not treated as markers.
//
// The function expects a named type whose underlying is an interface.
// Returns false if named is nil or its underlying is not an interface.
func IsMarkerInterface(named *types.Named) bool {
	if named == nil {
		return false
	}

	underlying := named.Underlying()
	iface, ok := underlying.(*types.Interface)
	if !ok {
		return false
	}

	// Constraint interfaces with type sets are not markers
	if !iface.IsMethodSet() {
		return false
	}

	// Marker iff: zero explicit methods AND zero embedded interfaces
	return iface.NumMethods() == 0 && iface.NumEmbeddeds() == 0
}
