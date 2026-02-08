package typeutil

import (
	"go/importer"
	"go/token"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestImplementsEither tests the T/*T implementation check.
func TestImplementsEither(t *testing.T) {
	t.Parallel()

	pkg := types.NewPackage("test/pkg", "pkg")

	// Create interface: type Reader interface { Read() []byte }
	byteSlice := types.NewSlice(types.Typ[types.Byte])
	resultVar := types.NewVar(token.NoPos, pkg, "", byteSlice)
	results := types.NewTuple(resultVar)
	readSig := types.NewSignatureType(nil, nil, nil, nil, results, false)
	readMethod := types.NewFunc(token.NoPos, pkg, "Read", readSig)
	readerIface := types.NewInterfaceType([]*types.Func{readMethod}, nil)
	readerIface.Complete()

	t.Run("NilTypeReturnsFalse", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify nil type returns false
		// Expected: false
		assert.False(t, ImplementsEither(nil, readerIface))
	})

	t.Run("NilInterfaceReturnsFalse", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify nil interface returns false
		// Expected: false
		structType := types.NewStruct(nil, nil)
		structTypeName := types.NewTypeName(token.NoPos, pkg, "MyStruct", nil)
		structNamed := types.NewNamed(structTypeName, structType, nil)
		assert.False(t, ImplementsEither(structNamed, nil))
	})

	t.Run("ValueReceiverImplements", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify type with value receiver method implements interface
		// Input: type Impl struct{}; func (Impl) Read() []byte
		// Expected: true (T implements)

		implStruct := types.NewStruct(nil, nil)
		implTypeName := types.NewTypeName(token.NoPos, pkg, "ValueImpl", nil)
		implNamed := types.NewNamed(implTypeName, implStruct, nil)

		// Add method with value receiver
		recv := types.NewVar(token.NoPos, pkg, "", implNamed)
		methodSig := types.NewSignatureType(recv, nil, nil, nil, results, false)
		method := types.NewFunc(token.NoPos, pkg, "Read", methodSig)
		implNamed.AddMethod(method)

		assert.True(t, ImplementsEither(implNamed, readerIface))
	})

	t.Run("PointerReceiverImplements", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify type with pointer receiver method implements interface via *T
		// Input: type Impl struct{}; func (*Impl) Read() []byte
		// Expected: true (*T implements)

		implStruct := types.NewStruct(nil, nil)
		implTypeName := types.NewTypeName(token.NoPos, pkg, "PtrImpl", nil)
		implNamed := types.NewNamed(implTypeName, implStruct, nil)

		// Add method with pointer receiver
		ptrRecv := types.NewVar(token.NoPos, pkg, "", types.NewPointer(implNamed))
		methodSig := types.NewSignatureType(ptrRecv, nil, nil, nil, results, false)
		method := types.NewFunc(token.NoPos, pkg, "Read", methodSig)
		implNamed.AddMethod(method)

		// T doesn't implement, but *T does
		assert.False(t, types.Implements(implNamed, readerIface))
		assert.True(t, types.Implements(types.NewPointer(implNamed), readerIface))
		assert.True(t, ImplementsEither(implNamed, readerIface))
	})

	t.Run("NoMethodDoesNotImplement", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify type without required method doesn't implement
		// Input: type Empty struct{}
		// Expected: false

		emptyStruct := types.NewStruct(nil, nil)
		emptyTypeName := types.NewTypeName(token.NoPos, pkg, "Empty", nil)
		emptyNamed := types.NewNamed(emptyTypeName, emptyStruct, nil)

		assert.False(t, ImplementsEither(emptyNamed, readerIface))
	})

	t.Run("WrongMethodSignatureDoesNotImplement", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify type with wrong method signature doesn't implement
		// Input: type Wrong struct{}; func (Wrong) Read() string (wrong return type)
		// Expected: false

		wrongStruct := types.NewStruct(nil, nil)
		wrongTypeName := types.NewTypeName(token.NoPos, pkg, "Wrong", nil)
		wrongNamed := types.NewNamed(wrongTypeName, wrongStruct, nil)

		// Add method with wrong return type
		wrongResultVar := types.NewVar(token.NoPos, pkg, "", types.Typ[types.String])
		wrongResults := types.NewTuple(wrongResultVar)
		recv := types.NewVar(token.NoPos, pkg, "", wrongNamed)
		wrongMethodSig := types.NewSignatureType(recv, nil, nil, nil, wrongResults, false)
		wrongMethod := types.NewFunc(token.NoPos, pkg, "Read", wrongMethodSig)
		wrongNamed.AddMethod(wrongMethod)

		assert.False(t, ImplementsEither(wrongNamed, readerIface))
	})

	t.Run("EmptyInterfaceAlwaysImplemented", func(t *testing.T) {
		t.Parallel()
		// Purpose: verify any type implements empty interface
		// Input: any type, interface{}
		// Expected: true

		emptyIface := types.NewInterfaceType(nil, nil)
		emptyIface.Complete()

		implStruct := types.NewStruct(nil, nil)
		implTypeName := types.NewTypeName(token.NoPos, pkg, "AnyType", nil)
		implNamed := types.NewNamed(implTypeName, implStruct, nil)

		assert.True(t, ImplementsEither(implNamed, emptyIface))
	})

	// Test with real types from standard library
	t.Run("RealTypesFromStdlib", func(t *testing.T) {
		t.Parallel()
		imp := importer.Default()

		// Load io package
		ioPkg, err := imp.Import("io")
		require.NoError(t, err)

		// Get io.Reader interface
		readerObj := ioPkg.Scope().Lookup("Reader")
		require.NotNil(t, readerObj)
		ioReaderIface := readerObj.Type().Underlying().(*types.Interface)

		// Load bytes package
		bytesPkg, err := imp.Import("bytes")
		require.NoError(t, err)

		// bytes.Buffer implements io.Reader (via pointer receiver)
		bufferObj := bytesPkg.Scope().Lookup("Buffer")
		require.NotNil(t, bufferObj)
		bufferNamed := bufferObj.Type().(*types.Named)

		assert.True(t, ImplementsEither(bufferNamed, ioReaderIface))

		// Load strings package
		stringsPkg, err := imp.Import("strings")
		require.NoError(t, err)

		// strings.Reader implements io.Reader
		strReaderObj := stringsPkg.Scope().Lookup("Reader")
		require.NotNil(t, strReaderObj)
		strReaderNamed := strReaderObj.Type().(*types.Named)

		assert.True(t, ImplementsEither(strReaderNamed, ioReaderIface))
	})
}
