package typeutil

import (
	"go/types"
)

// ImplementsEither checks if type T or *T implements the given interface.
//
// A type T is considered to implement interface I if
// either T or *T satisfies I's method set.
//
// Returns true if either form implements the interface.
func ImplementsEither(t *types.Named, iface *types.Interface) bool {
	if t == nil || iface == nil {
		return false
	}

	// Check if T implements interface
	if types.Implements(t, iface) {
		return true
	}

	// Check if *T implements interface
	ptrT := types.NewPointer(t)
	return types.Implements(ptrT, iface)
}
