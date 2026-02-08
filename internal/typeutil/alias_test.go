package typeutil

import (
	"go/importer"
	"go/token"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUnaliasTypeName tests alias ownership resolution.
func TestUnaliasTypeName(t *testing.T) {
	t.Parallel()

	t.Run("NilReturnsNil", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify nil input returns nil
		// Expected: nil
		assert.Nil(t, UnaliasTypeName(nil))
	})

	t.Run("NonAliasReturnsSelf", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify non-alias TypeName returns itself
		// Input: type Reader interface { Read() }
		// Expected: same TypeName
		pkg := types.NewPackage("test/pkg", "pkg")
		readMethod := types.NewFunc(token.NoPos, pkg, "Read", types.NewSignatureType(
			nil, nil, nil, nil, nil, false,
		))
		readerIface := types.NewInterfaceType([]*types.Func{readMethod}, nil)
		readerIface.Complete()
		readerTypeName := types.NewTypeName(token.NoPos, pkg, "Reader", nil)
		_ = types.NewNamed(readerTypeName, readerIface, nil)

		result := UnaliasTypeName(readerTypeName)
		assert.Equal(t, readerTypeName, result)
	})

	t.Run("AliasToInterface", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify alias to interface returns original TypeName
		// Input: type Alias = Original (where Original is in another package)
		// Expected: Original's TypeName

		// Create "original" package with interface
		origPkg := types.NewPackage("test/original", "original")
		readMethod := types.NewFunc(token.NoPos, origPkg, "Read", types.NewSignatureType(
			nil, nil, nil, nil, nil, false,
		))
		origIface := types.NewInterfaceType([]*types.Func{readMethod}, nil)
		origIface.Complete()
		origTypeName := types.NewTypeName(token.NoPos, origPkg, "Reader", nil)
		origNamed := types.NewNamed(origTypeName, origIface, nil)

		// Create "alias" package with alias to original
		aliasPkg := types.NewPackage("test/alias", "alias")
		aliasTypeName := types.NewTypeName(token.NoPos, aliasPkg, "ReaderAlias", origNamed)
		// In Go 1.22+, NewTypeName with a non-nil rhs creates an alias

		result := UnaliasTypeName(aliasTypeName)
		// Should resolve to original
		assert.Equal(t, origTypeName, result)
		assert.Equal(t, "test/original", result.Pkg().Path())
	})

	t.Run("RealAliasFromStdlib", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify with real stdlib type
		// io.ReadCloser is NOT an alias, but we can verify the function works
		imp := importer.Default()
		ioPkg, err := imp.Import("io")
		require.NoError(t, err)

		readerObj := ioPkg.Scope().Lookup("Reader")
		require.NotNil(t, readerObj)
		readerTypeName, ok := readerObj.(*types.TypeName)
		require.True(t, ok)

		// Reader is not an alias, should return itself
		result := UnaliasTypeName(readerTypeName)
		assert.Equal(t, readerTypeName, result)
		assert.Equal(t, "io", result.Pkg().Path())
	})

	t.Run("AliasChain", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify alias chain is fully resolved
		// Input: type A = B, type B = Original
		// Expected: Original's TypeName

		// Create original
		origPkg := types.NewPackage("test/original", "original")
		origIface := types.NewInterfaceType(nil, nil)
		origIface.Complete()
		origTypeName := types.NewTypeName(token.NoPos, origPkg, "Original", nil)
		origNamed := types.NewNamed(origTypeName, origIface, nil)

		// Create intermediate alias B = Original
		midPkg := types.NewPackage("test/mid", "mid")
		midTypeName := types.NewTypeName(token.NoPos, midPkg, "B", origNamed)

		// Create final alias A = B
		finalPkg := types.NewPackage("test/final", "final")
		midType := midTypeName.Type()
		finalTypeName := types.NewTypeName(token.NoPos, finalPkg, "A", midType)

		result := UnaliasTypeName(finalTypeName)
		// Should resolve all the way to original
		assert.Equal(t, origTypeName, result)
		assert.Equal(t, "test/original", result.Pkg().Path())
	})

	t.Run("AliasToBasicType", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify alias to basic type returns original TypeName
		// Input: type MyInt = int
		// Expected: returns the alias TypeName (basic types don't have TypeName in usual sense)

		pkg := types.NewPackage("test/pkg", "pkg")
		aliasTypeName := types.NewTypeName(token.NoPos, pkg, "MyInt", types.Typ[types.Int])

		result := UnaliasTypeName(aliasTypeName)
		// For basic types, Unalias returns the basic type which is not *Named
		// So we fall back to returning the original
		assert.Equal(t, aliasTypeName, result)
	})
}
