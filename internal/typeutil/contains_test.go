package typeutil

import (
	"go/importer"
	"go/token"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestContainsInterface tests the ContainsInterface predicate behavior.
func TestContainsInterface(t *testing.T) {
	t.Parallel()

	// Create a test package with various types
	pkg := types.NewPackage("test/pkg", "pkg")

	// Create target interface: type Reader interface { Read() }
	readMethod := types.NewFunc(token.NoPos, pkg, "Read", types.NewSignatureType(
		nil, nil, nil, nil, nil, false,
	))
	readerIface := types.NewInterfaceType([]*types.Func{readMethod}, nil)
	readerIface.Complete()
	readerTypeName := types.NewTypeName(token.NoPos, pkg, "Reader", nil)
	readerNamed := types.NewNamed(readerTypeName, readerIface, nil)
	_ = readerNamed // ensure type is set

	// Create another interface for negative tests
	writeMethod := types.NewFunc(token.NoPos, pkg, "Write", types.NewSignatureType(
		nil, nil, nil, nil, nil, false,
	))
	writerIface := types.NewInterfaceType([]*types.Func{writeMethod}, nil)
	writerIface.Complete()
	writerTypeName := types.NewTypeName(token.NoPos, pkg, "Writer", nil)
	writerNamed := types.NewNamed(writerTypeName, writerIface, nil)
	_ = writerNamed

	t.Run("DirectMatch", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify direct named type match
		// Input: readerNamed, target=readerTypeName
		// Expected: true
		assert.True(t, ContainsInterface(readerTypeName, readerNamed))
	})

	t.Run("NoMatchDifferentInterface", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify different named interface doesn't match
		// Input: writerNamed, target=readerTypeName
		// Expected: false
		assert.False(t, ContainsInterface(readerTypeName, writerNamed))
	})

	t.Run("PointerToInterface", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify pointer wrapping is traversed
		// Input: *Reader, target=readerTypeName
		// Expected: true
		ptrType := types.NewPointer(readerNamed)
		assert.True(t, ContainsInterface(readerTypeName, ptrType))
	})

	t.Run("SliceOfInterface", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify slice element is traversed
		// Input: []Reader, target=readerTypeName
		// Expected: true
		sliceType := types.NewSlice(readerNamed)
		assert.True(t, ContainsInterface(readerTypeName, sliceType))
	})

	t.Run("ArrayOfInterface", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify array element is traversed
		// Input: [5]Reader, target=readerTypeName
		// Expected: true
		arrayType := types.NewArray(readerNamed, 5)
		assert.True(t, ContainsInterface(readerTypeName, arrayType))
	})

	t.Run("MapKeyInterface", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify map key is traversed
		// Input: map[Reader]int, target=readerTypeName
		// Expected: true
		mapType := types.NewMap(readerNamed, types.Typ[types.Int])
		assert.True(t, ContainsInterface(readerTypeName, mapType))
	})

	t.Run("MapValueInterface", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify map value is traversed
		// Input: map[string]Reader, target=readerTypeName
		// Expected: true
		mapType := types.NewMap(types.Typ[types.String], readerNamed)
		assert.True(t, ContainsInterface(readerTypeName, mapType))
	})

	t.Run("ChanOfInterface", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify channel element is traversed
		// Input: chan Reader, target=readerTypeName
		// Expected: true
		chanType := types.NewChan(types.SendRecv, readerNamed)
		assert.True(t, ContainsInterface(readerTypeName, chanType))
	})

	t.Run("StructFieldInterface", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify struct fields are traversed
		// Input: struct { R Reader }, target=readerTypeName
		// Expected: true
		field := types.NewField(token.NoPos, pkg, "R", readerNamed, false)
		structType := types.NewStruct([]*types.Var{field}, nil)
		assert.True(t, ContainsInterface(readerTypeName, structType))
	})

	t.Run("StructEmbeddedInterface", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify embedded fields in struct are traversed
		// Input: struct { Reader }, target=readerTypeName (embedded)
		// Expected: true
		embeddedField := types.NewField(token.NoPos, pkg, "Reader", readerNamed, true)
		structType := types.NewStruct([]*types.Var{embeddedField}, nil)
		assert.True(t, ContainsInterface(readerTypeName, structType))
	})

	t.Run("NestedStructInterface", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify deeply nested interface is found
		// Input: struct { Inner struct { R Reader } }, target=readerTypeName
		// Expected: true
		innerField := types.NewField(token.NoPos, pkg, "R", readerNamed, false)
		innerStruct := types.NewStruct([]*types.Var{innerField}, nil)
		outerField := types.NewField(token.NoPos, pkg, "Inner", innerStruct, false)
		outerStruct := types.NewStruct([]*types.Var{outerField}, nil)
		assert.True(t, ContainsInterface(readerTypeName, outerStruct))
	})

	t.Run("SignatureParamInterface", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify function signature params are traversed
		// Input: func(Reader), target=readerTypeName
		// Expected: true
		param := types.NewVar(token.NoPos, pkg, "r", readerNamed)
		params := types.NewTuple(param)
		sig := types.NewSignatureType(nil, nil, nil, params, nil, false)
		assert.True(t, ContainsInterface(readerTypeName, sig))
	})

	t.Run("SignatureResultInterface", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify function signature results are traversed
		// Input: func() Reader, target=readerTypeName
		// Expected: true
		result := types.NewVar(token.NoPos, pkg, "", readerNamed)
		results := types.NewTuple(result)
		sig := types.NewSignatureType(nil, nil, nil, nil, results, false)
		assert.True(t, ContainsInterface(readerTypeName, sig))
	})

	t.Run("TupleInterface", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify tuple elements are traversed
		// Input: (Reader, error), target=readerTypeName
		// Expected: true
		v1 := types.NewVar(token.NoPos, pkg, "", readerNamed)
		v2 := types.NewVar(token.NoPos, pkg, "", types.Universe.Lookup("error").Type())
		tuple := types.NewTuple(v1, v2)
		assert.True(t, ContainsInterface(readerTypeName, tuple))
	})

	t.Run("InterfaceEmbeddedInterface", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify interface embedded types are traversed
		// Input: interface { Reader }, target=readerTypeName
		// Expected: true (Reader is embedded)
		compositeIface := types.NewInterfaceType(nil, []types.Type{readerNamed})
		compositeIface.Complete()
		assert.True(t, ContainsInterface(readerTypeName, compositeIface))
	})

	t.Run("InterfaceMethodSignature", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify interface method signatures are traversed
		// Input: interface { Process(Reader) }, target=readerTypeName
		// Expected: true
		param := types.NewVar(token.NoPos, pkg, "r", readerNamed)
		params := types.NewTuple(param)
		methodSig := types.NewSignatureType(nil, nil, nil, params, nil, false)
		method := types.NewFunc(token.NoPos, pkg, "Process", methodSig)
		iface := types.NewInterfaceType([]*types.Func{method}, nil)
		iface.Complete()
		assert.True(t, ContainsInterface(readerTypeName, iface))
	})

	t.Run("BasicTypeNoMatch", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify basic types don't match
		// Input: int, target=readerTypeName
		// Expected: false
		assert.False(t, ContainsInterface(readerTypeName, types.Typ[types.Int]))
	})

	t.Run("NilTypeNoMatch", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify nil type returns false
		// Input: nil, target=readerTypeName
		// Expected: false
		assert.False(t, ContainsInterface(readerTypeName, nil))
	})

	t.Run("CyclicTypeHandling", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify cyclic types don't cause infinite recursion
		// Create a struct that references itself: type Node struct { Next *Node }
		// Input: cyclic struct, target=readerTypeName
		// Expected: false (no Reader in the cycle)
		nodeTypeName := types.NewTypeName(token.NoPos, pkg, "Node", nil)
		nodeNamed := types.NewNamed(nodeTypeName, nil, nil)
		nodeField := types.NewField(token.NoPos, pkg, "Next", types.NewPointer(nodeNamed), false)
		nodeStruct := types.NewStruct([]*types.Var{nodeField}, nil)
		nodeNamed.SetUnderlying(nodeStruct)
		assert.False(t, ContainsInterface(readerTypeName, nodeNamed))
	})

	// Test with real types from standard library
	t.Run("RealTypesFromStdlib", func(t *testing.T) {
		t.Parallel()
		imp := importer.Default()
		ioPkg, err := imp.Import("io")
		require.NoError(t, err)

		readerObj := ioPkg.Scope().Lookup("Reader")
		require.NotNil(t, readerObj)
		ioReaderTypeName, ok := readerObj.(*types.TypeName)
		require.True(t, ok)
		ioReaderNamed := ioReaderTypeName.Type().(*types.Named)

		// io.Reader directly
		assert.True(t, ContainsInterface(ioReaderTypeName, ioReaderNamed))

		// []io.Reader
		sliceOfReader := types.NewSlice(ioReaderNamed)
		assert.True(t, ContainsInterface(ioReaderTypeName, sliceOfReader))

		// io.Writer should not contain io.Reader
		writerObj := ioPkg.Scope().Lookup("Writer")
		require.NotNil(t, writerObj)
		ioWriterNamed := writerObj.Type().(*types.Named)
		assert.False(t, ContainsInterface(ioReaderTypeName, ioWriterNamed))
	})
}

// TestContainsInterface_TypeParam verifies TypeParam handling.
func TestContainsInterface_TypeParam(t *testing.T) {
	t.Parallel()

	// Create test types
	pkg := types.NewPackage("test/pkg", "pkg")

	// Create target interface
	readMethod := types.NewFunc(token.NoPos, pkg, "Read", types.NewSignatureType(
		nil, nil, nil, nil, nil, false,
	))
	readerIface := types.NewInterfaceType([]*types.Func{readMethod}, nil)
	readerIface.Complete()
	readerTypeName := types.NewTypeName(token.NoPos, pkg, "Reader", nil)
	readerNamed := types.NewNamed(readerTypeName, readerIface, nil)
	_ = readerNamed

	t.Run("TypeParamStops", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify TypeParam stops traversal (doesn't expand constraint)
		// Input: TypeParam with Reader constraint
		// Expected: false (we don't traverse into constraint)

		// Create a type parameter: T with constraint Reader
		tpn := types.NewTypeName(token.NoPos, pkg, "T", nil)
		tp := types.NewTypeParam(tpn, readerNamed)
		_ = tp
		// TypeParam itself should not match (spec says stop at TypeParam)
		assert.False(t, ContainsInterface(readerTypeName, tp))
	})
}
