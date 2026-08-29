// Package typeutil provides shared type-related utilities for the ifaceguard analyzer.
package typeutil

import (
	"go/types"
)

// ContainsInterface checks whether the given type t contains the target interface
// (identified by its *types.TypeName object) either directly or nested.
//
// This implements the Contains(P.I, T) predicate used to detect nested interface usage:
// DFS traversal with visited map (identity-based) through type graph.
//
// Traversal rules:
//   - *types.Named: match Obj() == target; otherwise walk Underlying()
//   - *types.Pointer, *types.Slice, *types.Array, *types.Chan: walk Elem()
//   - *types.Map: walk Key() and Elem()
//   - *types.Struct: walk all Field(i).Type() including embedded
//   - *types.Tuple: walk all At(i).Type()
//   - *types.Signature: walk Params() and Results()
//   - *types.Interface: walk EmbeddedType(i) and Method(i).Type()
//   - *types.TypeParam: stop (do not expand constraint)
//   - *types.Basic and others: stop
func ContainsInterface(target *types.TypeName, t types.Type) bool {
	visited := make(map[types.Type]bool)
	return containsWalk(target, t, visited)
}

func containsWalk(target *types.TypeName, t types.Type, visited map[types.Type]bool) bool {
	if t == nil {
		return false
	}

	if visited[t] {
		return false
	}
	visited[t] = true

	switch typ := t.(type) {
	case *types.Named:
		return containsWalkNamed(target, typ, visited)
	case *types.Pointer:
		return containsWalk(target, typ.Elem(), visited)
	case *types.Slice:
		return containsWalk(target, typ.Elem(), visited)
	case *types.Array:
		return containsWalk(target, typ.Elem(), visited)
	case *types.Map:
		return containsWalkMap(target, typ, visited)
	case *types.Chan:
		return containsWalk(target, typ.Elem(), visited)
	case *types.Struct:
		return containsWalkStruct(target, typ, visited)
	case *types.Tuple:
		return containsWalkTuple(target, typ, visited)
	case *types.Signature:
		return containsWalkSignature(target, typ, visited)
	case *types.Interface:
		return containsWalkInterface(target, typ, visited)
	case *types.TypeParam:
		// Stop at TypeParam; do not expand constraint
		return false
	default:
		// *types.Basic and any other types: stop
		return false
	}
}

func containsWalkNamed(target *types.TypeName, typ *types.Named, visited map[types.Type]bool) bool {
	if typ.Obj() == target {
		return true
	}
	return containsWalk(target, typ.Underlying(), visited)
}

func containsWalkMap(target *types.TypeName, typ *types.Map, visited map[types.Type]bool) bool {
	if containsWalk(target, typ.Key(), visited) {
		return true
	}
	return containsWalk(target, typ.Elem(), visited)
}

func containsWalkStruct(target *types.TypeName, typ *types.Struct, visited map[types.Type]bool) bool {
	for field := range typ.Fields() {
		if containsWalk(target, field.Type(), visited) {
			return true
		}
	}
	return false
}

func containsWalkTuple(target *types.TypeName, typ *types.Tuple, visited map[types.Type]bool) bool {
	for v := range typ.Variables() {
		if containsWalk(target, v.Type(), visited) {
			return true
		}
	}
	return false
}

func containsWalkSignature(target *types.TypeName, typ *types.Signature, visited map[types.Type]bool) bool {
	if containsWalk(target, typ.Params(), visited) {
		return true
	}
	return containsWalk(target, typ.Results(), visited)
}

func containsWalkInterface(target *types.TypeName, typ *types.Interface, visited map[types.Type]bool) bool {
	for etyp := range typ.EmbeddedTypes() {
		if containsWalk(target, etyp, visited) {
			return true
		}
	}
	for method := range typ.Methods() {
		if containsWalk(target, method.Type(), visited) {
			return true
		}
	}
	return false
}
