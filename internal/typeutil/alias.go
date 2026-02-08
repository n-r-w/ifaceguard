package typeutil

import (
	"go/types"
)

// UnaliasTypeName resolves the owner of an interface through type aliases.
//
// If an interface is declared as alias `type I = other.I`,
// the owner is the package of `other.I`, not the alias package.
//
// This function walks through alias chains using types.Unalias to find
// the ultimate non-alias definition and returns its TypeName.
//
// Returns the input obj if it's not an alias or if unaliasing doesn't
// lead to a named type.
func UnaliasTypeName(obj *types.TypeName) *types.TypeName {
	if obj == nil {
		return nil
	}

	// Unalias walks through alias chain
	unaliased := types.Unalias(obj.Type())

	// If result is a named type, return its TypeName
	if named, ok := unaliased.(*types.Named); ok {
		return named.Obj()
	}

	// Fallback to original if not named (e.g., basic types)
	return obj
}
