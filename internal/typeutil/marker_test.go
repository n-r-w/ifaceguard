package typeutil

import (
	"go/importer"
	"go/token"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIsMarkerInterface tests marker interface detection.
func TestIsMarkerInterface(t *testing.T) {
	t.Parallel()

	pkg := types.NewPackage("test/pkg", "pkg")

	t.Run("EmptyInterfaceIsMarker", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify empty interface (no methods, no embeds) is marker
		// Input: type Marker interface {}
		// Expected: true
		markerIface := types.NewInterfaceType(nil, nil)
		markerIface.Complete()
		markerTypeName := types.NewTypeName(token.NoPos, pkg, "Marker", nil)
		markerNamed := types.NewNamed(markerTypeName, markerIface, nil)
		assert.True(t, IsMarkerInterface(markerNamed))
	})

	t.Run("InterfaceWithMethodNotMarker", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify interface with methods is not marker
		// Input: type Reader interface { Read() }
		// Expected: false
		readMethod := types.NewFunc(token.NoPos, pkg, "Read", types.NewSignatureType(
			nil, nil, nil, nil, nil, false,
		))
		readerIface := types.NewInterfaceType([]*types.Func{readMethod}, nil)
		readerIface.Complete()
		readerTypeName := types.NewTypeName(token.NoPos, pkg, "Reader", nil)
		readerNamed := types.NewNamed(readerTypeName, readerIface, nil)
		assert.False(t, IsMarkerInterface(readerNamed))
	})

	t.Run("InterfaceWithEmbeddedNotMarker", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify interface with embedded interface is not marker
		// Input: type Extended interface { Marker }
		// Expected: false (has embedded, even if marker is empty)
		innerIface := types.NewInterfaceType(nil, nil)
		innerIface.Complete()
		innerTypeName := types.NewTypeName(token.NoPos, pkg, "Inner", nil)
		innerNamed := types.NewNamed(innerTypeName, innerIface, nil)

		outerIface := types.NewInterfaceType(nil, []types.Type{innerNamed})
		outerIface.Complete()
		outerTypeName := types.NewTypeName(token.NoPos, pkg, "Outer", nil)
		outerNamed := types.NewNamed(outerTypeName, outerIface, nil)
		assert.False(t, IsMarkerInterface(outerNamed))
	})

	t.Run("NilReturnsFalse", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify nil input returns false
		// Expected: false
		assert.False(t, IsMarkerInterface(nil))
	})

	t.Run("NonInterfaceReturnsFalse", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify non-interface named type returns false
		// Input: type MyInt int
		// Expected: false
		myIntTypeName := types.NewTypeName(token.NoPos, pkg, "MyInt", nil)
		myIntNamed := types.NewNamed(myIntTypeName, types.Typ[types.Int], nil)
		assert.False(t, IsMarkerInterface(myIntNamed))
	})

	t.Run("StructReturnsFalse", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify struct type returns false
		// Input: type MyStruct struct {}
		// Expected: false
		structType := types.NewStruct(nil, nil)
		structTypeName := types.NewTypeName(token.NoPos, pkg, "MyStruct", nil)
		structNamed := types.NewNamed(structTypeName, structType, nil)
		assert.False(t, IsMarkerInterface(structNamed))
	})

	// Test with real types from standard library
	t.Run("RealTypesFromStdlib", func(t *testing.T) {
		t.Parallel()
		imp := importer.Default()

		// io.Reader has method Read - not marker
		ioPkg, err := imp.Import("io")
		require.NoError(t, err)
		readerObj := ioPkg.Scope().Lookup("Reader")
		require.NotNil(t, readerObj)
		readerNamed := readerObj.Type().(*types.Named)
		assert.False(t, IsMarkerInterface(readerNamed))

		// error interface has method Error - not marker
		errorObj := types.Universe.Lookup("error")
		require.NotNil(t, errorObj)
		errorNamed := errorObj.Type().(*types.Named)
		assert.False(t, IsMarkerInterface(errorNamed))

		// any (alias for interface{}) - check if we can test it
		// Note: 'any' is a type alias, not a Named type in Go 1.18+
	})
}

// TestIsMarkerInterface_Constraints tests constraint interface handling.
func TestIsMarkerInterface_Constraints(t *testing.T) {
	t.Parallel()

	pkg := types.NewPackage("test/pkg", "pkg")

	t.Run("ConstraintWithTypeSetNotMarker", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify constraint interface with type set is NOT marker
		// Input: type Signed interface { ~int | ~int8 | ~int16 | ~int32 | ~int64 }
		// Expected: false (IsMethodSet() == false for type sets)

		// Create union type: ~int | ~int64
		term1 := types.NewTerm(true, types.Typ[types.Int])   // ~int
		term2 := types.NewTerm(true, types.Typ[types.Int64]) // ~int64
		union := types.NewUnion([]*types.Term{term1, term2})

		// Create interface with type set (no methods, but has type constraint)
		constraintIface := types.NewInterfaceType(nil, []types.Type{union})
		constraintIface.Complete()

		constraintTypeName := types.NewTypeName(token.NoPos, pkg, "Signed", nil)
		constraintNamed := types.NewNamed(constraintTypeName, constraintIface, nil)

		// This should NOT be a marker because IsMethodSet() returns false for type sets
		assert.False(t, IsMarkerInterface(constraintNamed))
	})

	t.Run("ConstraintWithMethodAndTypeSetNotMarker", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify constraint with both method and type set is not marker
		// Input: type Stringer interface { ~string; String() string }
		// Expected: false

		term := types.NewTerm(true, types.Typ[types.String]) // ~string
		union := types.NewUnion([]*types.Term{term})

		// Create method
		resultVar := types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])
		results := types.NewTuple(resultVar)
		stringSig := types.NewSignatureType(nil, nil, nil, nil, results, false)
		stringMethod := types.NewFunc(token.NoPos, pkg, "String", stringSig)

		constraintIface := types.NewInterfaceType([]*types.Func{stringMethod}, []types.Type{union})
		constraintIface.Complete()

		constraintTypeName := types.NewTypeName(token.NoPos, pkg, "MyStringer", nil)
		constraintNamed := types.NewNamed(constraintTypeName, constraintIface, nil)

		assert.False(t, IsMarkerInterface(constraintNamed))
	})

	t.Run("ComparableIsNotMarker", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify comparable constraint is not marker
		// comparable is a predeclared constraint

		comparableObj := types.Universe.Lookup("comparable")
		require.NotNil(t, comparableObj)

		// comparable is a TypeName but its type is an interface with implicit type set
		comparableTypeName, ok := comparableObj.(*types.TypeName)
		require.True(t, ok)

		comparableType := comparableTypeName.Type()
		comparableNamed, ok := comparableType.(*types.Named)
		if !ok {
			// In some Go versions, comparable might not be a Named type
			// Skip this test in that case
			t.Skip("comparable is not a Named type in this Go version")
		}

		// comparable has implicit type set, so IsMethodSet() should be false
		assert.False(t, IsMarkerInterface(comparableNamed))
	})
}
