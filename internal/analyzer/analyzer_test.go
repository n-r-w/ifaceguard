package analyzer_test

import (
	"path/filepath"
	"testing"

	"github.com/n-r-w/ifaceguard/internal/analyzer"
	"github.com/n-r-w/ifaceguard/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis/analysistest"
)

func moduleTestdataDir(t *testing.T) string {
	t.Helper()

	return filepath.Join(analysistest.TestData(), "src")
}

func TestNew_DefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := config.Default()
	a, err := analyzer.New(cfg)
	require.NoError(t, err)
	require.NotNil(t, a)

	assert.Equal(t, analyzer.AnalyzerName, a.Name)
}

// TestAnalysistest_Baseline verifies the analysistest infrastructure works correctly.
// The baseline package contains no interfaces or implementations, so no diagnostics are expected.
func TestAnalysistest_Baseline(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	// Run analysistest with baseline package - expects no diagnostics
	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/baseline")
}

// =============================================================================
// Assertions Tests
// =============================================================================

// TestAssertions_Basic tests basic assertions rule violation detection.
// Assertion in iface package for type declared in impl package should trigger IFG002-ASSERTION-PLACEMENT.
//
// Purpose: Verify basic assertions rule detection works.
// Inputs: Default config with assertions enabled.
// Expected: IFG002-ASSERTION-PLACEMENT diagnostic on assertion in iface/iface.go.
// Edge cases: None (basic case).
// Dependencies: assertions_basic fixture packages.
func TestAssertions_Basic(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	// Test the iface package where violation occurs.
	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/assertions_basic/iface")
}

// TestAssertions_AllRHSForms tests all supported RHS assertion forms for the assertions rule.
// Forms: (*T)(nil), new(T), T{}, &T{}.
//
// Purpose: Verify all RHS forms are recognized.
// Inputs: Default config with assertions enabled.
// Expected: IFG002-ASSERTION-PLACEMENT diagnostic for each assertion form.
// Edge cases: Different RHS syntaxes for the same semantic meaning.
// Dependencies: assertions_forms fixture packages.
func TestAssertions_AllRHSForms(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/assertions_forms/iface")
}

// TestAssertions_UnnamedInterfaceLiteral tests unnamed interface literal as LHS for assertions.
// Form: var _ interface{ Method() } = (*T)(nil).
//
// Purpose: Verify assertions with unnamed interface literals are recognized.
// Inputs: Default config with assertions enabled.
// Expected: IFG002-ASSERTION-PLACEMENT diagnostic for unnamed interface assertions.
// Edge cases: Interface type is inline literal, not a named type.
// Dependencies: assertions_unnamed fixture packages.
func TestAssertions_UnnamedInterfaceLiteral(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/assertions_unnamed/assert")
}

// TestAssertions_CrossPackage tests assertion in a different package from implementation type.
// This is the primary use case for the assertions rule.
//
// Purpose: Verify cross-package assertion detection.
// Inputs: Default config with assertions enabled.
// Expected: IFG002-ASSERTION-PLACEMENT diagnostic when assertion is not in impl package.
// Edge cases: Consumer package imports provider and creates assertion.
// Dependencies: assertions_crosspackage fixture packages.
func TestAssertions_CrossPackage(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/assertions_crosspackage/consumer")
}

// TestAssertions_CorrectPlacement tests that assertions in the implementation package are NOT violations.
//
// Purpose: Verify no false positives for correct assertion placement.
// Inputs: Default config with assertions enabled.
// Expected: No diagnostics (all assertions are in correct package).
// Edge cases: Multiple assertion forms, all correct.
// Dependencies: assertions_correct fixture package.
func TestAssertions_CorrectPlacement(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	// No diagnostics expected - assertions are in the same package as implementation.
	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/assertions_correct")
}

// TestAssertions_TestFilesIncluded verifies that assertions in _test.go are analyzed by default.
//
// Purpose: Ensure test files participate in assertions analysis.
// Inputs: Default config.
// Expected: IFG002-ASSERTION-PLACEMENT diagnostic from a _test.go file.
// Edge cases: Diagnostics should not be silently skipped.
// Dependencies: assertions_tests_included fixture packages.
func TestAssertions_TestFilesIncluded(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/assertions_tests_included/...")
}

// TestAssertions_Disabled verifies that assertions diagnostics are suppressed when disabled.
func TestAssertions_Disabled(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	cfg.Assertions.Enabled = false

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/assertions_disabled/iface")
}

// TestAssertions_WiringPackages_Allowed tests that assertions in wiring packages are allowed.
// Config: assertions.wiringpackages includes the compose package path.
//
// Purpose: Verify wiring package allowlist works.
// Inputs: Config with compose package in wiringpackages.
// Expected: No diagnostics for assertions in compose package.
// Edge cases: Same impl type, different assertion locations.
// Dependencies: assertions_wiring fixture packages.
func TestAssertions_WiringPackages_Allowed(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	cfg.Assertions.WiringPackages = []string{"^ifaceguard-testdata/assertions_wiring/compose$"}

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	// Only test compose package - no diagnostics expected.
	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/assertions_wiring/compose")
}

// TestAssertions_WiringPackages_NotAllowed tests that assertions NOT in wiring packages are violations.
// Config: assertions.wiringpackages includes compose but NOT other package.
//
// Purpose: Verify non-wiring packages still trigger diagnostics.
// Inputs: Config with compose package in wiringpackages.
// Expected: IFG002-ASSERTION-PLACEMENT diagnostic for assertion in other package.
// Edge cases: Allowlist is selective.
// Dependencies: assertions_wiring fixture packages.
func TestAssertions_WiringPackages_NotAllowed(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	cfg.Assertions.WiringPackages = []string{"^ifaceguard-testdata/assertions_wiring/compose$"}

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	// Test other package - should have violation.
	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/assertions_wiring/other")
}

// --------------------------------------------------------------------------
// Assertions Extension Flags Tests
// --------------------------------------------------------------------------

// TestAssertions_AcceptConversionOnlyForm_Disabled tests that conversion-only form assertions
// are NOT recognized when acceptconversiononlyform=false (default).
// Config: assertions.acceptconversiononlyform=false.
//
// Purpose: Verify conversion-only forms are ignored when flag is disabled.
// Inputs: Fixture with conversion-only assertion, flag disabled.
// Expected: No diagnostics (assertion not recognized).
// Edge cases: Conversion-only form should be invisible to analyzer.
// Dependencies: assertions_conversiononly fixture.
func TestAssertions_AcceptConversionOnlyForm_Disabled(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	cfg.Assertions.AcceptConversionOnlyForm = false // explicit default

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	// With flag disabled, conversion-only form should NOT be recognized.
	// No diagnostics expected - uses _nodiag fixture.
	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/assertions_conversiononly_nodiag/iface")
}

// TestAssertions_AcceptConversionOnlyForm_Enabled tests that conversion-only form assertions
// ARE recognized when acceptconversiononlyform=true.
// Config: assertions.acceptconversiononlyform=true.
//
// Purpose: Verify conversion-only forms are recognized when flag is enabled.
// Inputs: Fixture with conversion-only assertion, flag enabled.
// Expected: IFG002-ASSERTION-PLACEMENT diagnostic (assertion in wrong package).
// Edge cases: var _ = I(expr) form without explicit type.
// Dependencies: assertions_conversiononly fixture.
func TestAssertions_AcceptConversionOnlyForm_Enabled(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	cfg.Assertions.AcceptConversionOnlyForm = true

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	// With flag enabled, conversion-only form SHOULD be recognized.
	// Expect IFG002-ASSERTION-PLACEMENT diagnostic.
	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/assertions_conversiononly/iface")
}

// TestAssertions_ScanFunctionBodies_Disabled tests that assertions inside function bodies
// are NOT recognized when scanfunctionbodies=false (default).
// Config: assertions.scanfunctionbodies=false.
//
// Purpose: Verify function-body assertions are ignored when flag is disabled.
// Inputs: Fixture with assertion inside function, flag disabled.
// Expected: No diagnostics (assertion not recognized).
// Edge cases: Package-level only scanning.
// Dependencies: assertions_funcbody fixture.
func TestAssertions_ScanFunctionBodies_Disabled(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	cfg.Assertions.ScanFunctionBodies = false // explicit default

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	// With flag disabled, function-body assertions should NOT be recognized.
	// No diagnostics expected - uses _nodiag fixture.
	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/assertions_funcbody_nodiag/iface")
}

// TestAssertions_ScanFunctionBodies_Enabled tests that assertions inside function bodies
// ARE recognized when scanfunctionbodies=true.
// Config: assertions.scanfunctionbodies=true.
//
// Purpose: Verify function-body assertions are recognized when flag is enabled.
// Inputs: Fixture with assertion inside function, flag enabled.
// Expected: IFG002-ASSERTION-PLACEMENT diagnostic (assertion in wrong package).
// Edge cases: Assertion inside FuncDecl body.
// Dependencies: assertions_funcbody fixture.
func TestAssertions_ScanFunctionBodies_Enabled(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	cfg.Assertions.ScanFunctionBodies = true

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	// With flag enabled, function-body assertion SHOULD be recognized.
	// Expect IFG002-ASSERTION-PLACEMENT diagnostic.
	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/assertions_funcbody/iface")
}

// TestAssertions_ConversionOnlyInFuncBody_BothFlagsDisabled tests that conversion-only form
// assertions inside function bodies are NOT recognized when both flags are disabled.
// Config: assertions.acceptconversiononlyform=false, assertions.scanfunctionbodies=false.
//
// Purpose: Verify flag interaction - neither flag enables recognition.
// Inputs: Fixture with conversion-only assertion inside function, both flags disabled.
// Expected: No diagnostics (assertion not recognized).
// Edge cases: Requires both flags to recognize.
// Dependencies: assertions_convfuncbody fixture.
func TestAssertions_ConversionOnlyInFuncBody_BothFlagsDisabled(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	cfg.Assertions.AcceptConversionOnlyForm = false
	cfg.Assertions.ScanFunctionBodies = false

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	// With both flags disabled, assertion should NOT be recognized.
	// No diagnostics expected - uses _nodiag fixture.
	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/assertions_convfuncbody_nodiag/iface")
}

// TestAssertions_ConversionOnlyInFuncBody_OnlyConversionEnabled tests that conversion-only form
// assertions inside function bodies are NOT recognized when only acceptconversiononlyform=true.
// Config: assertions.acceptconversiononlyform=true, assertions.scanfunctionbodies=false.
//
// Purpose: Verify flag interaction - acceptconversiononlyform alone is insufficient.
// Inputs: Fixture with conversion-only assertion inside function, only conversion flag enabled.
// Expected: No diagnostics (function body not scanned).
// Edge cases: Conversion recognition without body scanning.
// Dependencies: assertions_convfuncbody fixture.
func TestAssertions_ConversionOnlyInFuncBody_OnlyConversionEnabled(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	cfg.Assertions.AcceptConversionOnlyForm = true
	cfg.Assertions.ScanFunctionBodies = false

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	// With only conversion flag enabled, function body not scanned.
	// No diagnostics expected - uses _nodiag fixture.
	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/assertions_convfuncbody_nodiag/iface")
}

// TestAssertions_ConversionOnlyInFuncBody_OnlyScanEnabled tests that conversion-only form
// assertions inside function bodies are NOT recognized when only scanfunctionbodies=true.
// Config: assertions.acceptconversiononlyform=false, assertions.scanfunctionbodies=true.
//
// Purpose: Verify flag interaction - scanfunctionbodies alone is insufficient.
// Inputs: Fixture with conversion-only assertion inside function, only scan flag enabled.
// Expected: No diagnostics (conversion-only form not recognized).
// Edge cases: Body scanning without conversion recognition.
// Dependencies: assertions_convfuncbody fixture.
func TestAssertions_ConversionOnlyInFuncBody_OnlyScanEnabled(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	cfg.Assertions.AcceptConversionOnlyForm = false
	cfg.Assertions.ScanFunctionBodies = true

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	// With only scan flag enabled, conversion-only form not recognized.
	// No diagnostics expected - uses _nodiag fixture.
	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/assertions_convfuncbody_nodiag/iface")
}

// TestAssertions_ConversionOnlyInFuncBody_BothFlagsEnabled tests that conversion-only form
// assertions inside function bodies ARE recognized when both flags are enabled.
// Config: assertions.acceptconversiononlyform=true, assertions.scanfunctionbodies=true.
//
// Purpose: Verify flag interaction - both flags enable recognition.
// Inputs: Fixture with conversion-only assertion inside function, both flags enabled.
// Expected: IFG002-ASSERTION-PLACEMENT diagnostic (assertion in wrong package).
// Edge cases: Full recognition path.
// Dependencies: assertions_convfuncbody fixture.
func TestAssertions_ConversionOnlyInFuncBody_BothFlagsEnabled(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	cfg.Assertions.AcceptConversionOnlyForm = true
	cfg.Assertions.ScanFunctionBodies = true

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	// With both flags enabled, assertion SHOULD be recognized.
	// Expect IFG002-ASSERTION-PLACEMENT diagnostic.
	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/assertions_convfuncbody/iface")
}

// TestAssertions_ConversionToNonInterface_NotRecognized tests that conversion to a non-interface
// type is NOT treated as a compile-time assertion, even when acceptconversiononlyform=true.
// Conversion targets must type-check to an interface type.
func TestAssertions_ConversionToNonInterface_NotRecognized(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	cfg.Assertions.AcceptConversionOnlyForm = true

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	// Conversion to struct type (not interface) should NOT be recognized as assertion.
	// Expect NO diagnostic.
	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/assertions_convnoniface/iface")
}

// TestAssertions_ScanFunctionBodies_NestedFuncLiteral tests that assertions inside nested
// function literals ARE scanned when scanfunctionbodies=true.
// Implementation uses ast.Inspect which recursively visits all nodes including func literals.
func TestAssertions_ScanFunctionBodies_NestedFuncLiteral(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	cfg.Assertions.ScanFunctionBodies = true

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	// Assertions inside nested func literals SHOULD be recognized.
	// Expect IFG002-ASSERTION-PLACEMENT diagnostic.
	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/assertions_funcbody_nested/iface")
}

func TestAssertions_RequireAssertions_MissingAssertion(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	cfg.Assertions.RequireAssertions = true
	scope := config.ContractScopeAnyExported
	cfg.Ownership.ContractScope = &scope

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/assertions_requireassertions_missing/impl")
}

func TestAssertions_RequireAssertions_MissingAssertion_NoReference(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	cfg.Assertions.RequireAssertions = true
	scope := config.ContractScopeAnyExported
	cfg.Ownership.ContractScope = &scope

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/assertions_requireassertions_noref/impl")
}

func TestAssertions_RequireAssertions_AssertionPresent(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	cfg.Assertions.RequireAssertions = true
	scope := config.ContractScopeAnyExported
	cfg.Ownership.ContractScope = &scope

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/assertions_requireassertions_present/impl")
}

func TestAssertions_RequireAssertions_GenericReceiver(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	cfg.Assertions.RequireAssertions = true
	scope := config.ContractScopeAnyExported
	cfg.Ownership.ContractScope = &scope

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/assertions_requireassertions_generic/impl")
}

func TestAssertions_RequireAssertions_Strict_Disabled_NoCoImport(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	cfg.Assertions.RequireAssertions = true
	cfg.Assertions.RequireAssertionsStrict = false
	scope := config.ContractScopeAnyExported
	cfg.Ownership.ContractScope = &scope

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/assertions_requireassertions_strict_nodiag/impl")
}

func TestAssertions_RequireAssertions_Strict_Enabled_NoCoImport(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	cfg.Assertions.RequireAssertions = true
	cfg.Assertions.RequireAssertionsStrict = true
	scope := config.ContractScopeAnyExported
	cfg.Ownership.ContractScope = &scope

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/assertions_requireassertions_strict/impl")
}

func TestAssertions_RequireAssertions_BypassUnnamedInterface(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	cfg.Assertions.RequireAssertions = true
	scope := config.ContractScopeAnyExported
	cfg.Ownership.ContractScope = &scope

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/assertions_requireassertions_bypass_unnamed/impl")
}

func TestAssertions_RequireAssertions_BypassPrivateInterfaceOnlyAssertion(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	cfg.Assertions.RequireAssertions = true
	scope := config.ContractScopeAnyExported
	cfg.Ownership.ContractScope = &scope

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/assertions_requireassertions_bypass_private/impl")
}

func TestAssertions_RequireAssertions_BypassChecksDisabled(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	cfg.Assertions.RequireAssertions = true
	cfg.Assertions.CheckBypassAssertions = false
	scope := config.ContractScopeAnyExported
	cfg.Ownership.ContractScope = &scope

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/assertions_requireassertions_bypass_private_nodiag/impl")
}

func TestAssertions_RequireAssertions_BypassUnnamedChecksDisabled(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	cfg.Assertions.RequireAssertions = true
	cfg.Assertions.CheckBypassAssertions = false
	scope := config.ContractScopeAnyExported
	cfg.Ownership.ContractScope = &scope

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/assertions_requireassertions_bypass_unnamed_nodiag/impl")
}

func TestAssertions_RequireAssertions_PrivateInterfaceNotOnlyAssertion_NoBypassDiag(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	cfg.Assertions.RequireAssertions = true
	scope := config.ContractScopeAnyExported
	cfg.Ownership.ContractScope = &scope

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/assertions_requireassertions_private_reused/impl")
}

// =============================================================================
// Ownership Tests (exportedoutput mode)
// =============================================================================

// TestOwnership_ExportedOutput_FuncResult tests that interfaces appearing in exported function
// results are classified as contractual under exportedoutput mode.
//
// Purpose: Verify basic function result detection in exportedoutput mode.
// Inputs: Config with ownership.contractscope=exportedoutput.
// Expected: IFG001-OWNERSHIP diagnostic on Runner interface declaration.
// Edge cases: None (basic case).
// Dependencies: ownership_expout_func_result fixture.
func TestOwnership_ExportedOutput_FuncResult(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	scope := config.ContractScopeExportedOutput
	cfg.Ownership.ContractScope = &scope

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	// Expect IFG001-OWNERSHIP diagnostic - interface appears in exported function result.
	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/ownership_expout_func_result")
}

// TestOwnership_ExportedOutput_MethodResult tests that interfaces appearing in exported method
// results are classified as contractual under exportedoutput mode.
//
// Purpose: Verify method result detection in exportedoutput mode.
// Inputs: Config with ownership.contractscope=exportedoutput.
// Expected: IFG001-OWNERSHIP diagnostic on Worker interface declaration.
// Edge cases: Method on struct type (not standalone function).
// Dependencies: ownership_expout_method_result fixture.
func TestOwnership_ExportedOutput_MethodResult(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	scope := config.ContractScopeExportedOutput
	cfg.Ownership.ContractScope = &scope

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	// Expect IFG001-OWNERSHIP diagnostic - interface appears in exported method result.
	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/ownership_expout_method_result")
}

// TestOwnership_ExportedOutput_Var tests that interfaces appearing in exported variable types
// are classified as contractual under exportedoutput mode.
//
// Purpose: Verify exported variable type detection in exportedoutput mode.
// Inputs: Config with ownership.contractscope=exportedoutput.
// Expected: IFG001-OWNERSHIP diagnostic on Handler interface declaration.
// Edge cases: Exported var with direct interface type.
// Dependencies: ownership_expout_var fixture.
func TestOwnership_ExportedOutput_Var(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	scope := config.ContractScopeExportedOutput
	cfg.Ownership.ContractScope = &scope

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	// Expect IFG001-OWNERSHIP diagnostic - interface appears in exported variable type.
	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/ownership_expout_var")
}

// TestOwnership_ExportedOutput_Nested tests that interfaces nested in exported outputs
// (via Contains() predicate) are classified as contractual under exportedoutput mode.
//
// Purpose: Verify nested interface detection via Contains() predicate.
// Inputs: Config with ownership.contractscope=exportedoutput.
// Expected: IFG001-OWNERSHIP diagnostic on Processor interface declaration.
// Edge cases: Interface nested in struct field and slice element type.
// Dependencies: ownership_expout_nested fixture.
func TestOwnership_ExportedOutput_Nested(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	scope := config.ContractScopeExportedOutput
	cfg.Ownership.ContractScope = &scope

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	// Expect IFG001-OWNERSHIP diagnostic - interface nested in exported function results.
	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/ownership_expout_nested")
}

// TestOwnership_ExportedOutput_SkipInput_BothInputAndOutput tests that interfaces appearing
// in both input AND output are still flagged when skipifusedasinput=true (default).
//
// Purpose: Verify skipifusedasinput does not suppress when interface is in output.
// Inputs: Config with ownership.contractscope=exportedoutput, skipifusedasinput=true.
// Expected: IFG001-OWNERSHIP diagnostic - interface appears in output, so it's contractual.
// Edge cases: Interface in both input params and return type.
// Dependencies: ownership_expout_skipinput fixture.
func TestOwnership_ExportedOutput_SkipInput_BothInputAndOutput(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	scope := config.ContractScopeExportedOutput
	cfg.Ownership.ContractScope = &scope
	cfg.Ownership.SkipIfUsedAsInput = true // explicit default

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	// Expect IFG001-OWNERSHIP diagnostic - interface appears in output despite also in input.
	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/ownership_expout_skipinput")
}

// TestOwnership_ExportedOutput_SkipInput_InputOnly tests that interfaces appearing ONLY
// in input params are NOT flagged when skipifusedasinput=true (default).
//
// Purpose: Verify skipifusedasinput suppression for input-only usage.
// Inputs: Config with ownership.contractscope=exportedoutput, skipifusedasinput=true.
// Expected: No diagnostics - interface appears only in input params.
// Edge cases: Input-only usage should not trigger diagnostic.
// Dependencies: ownership_expout_inputonly_nodiag fixture.
func TestOwnership_ExportedOutput_SkipInput_InputOnly(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	scope := config.ContractScopeExportedOutput
	cfg.Ownership.ContractScope = &scope
	cfg.Ownership.SkipIfUsedAsInput = true // explicit default

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	// No diagnostics expected - interface appears only in input params.
	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/ownership_expout_inputonly_nodiag")
}

// TestOwnership_ExportedOutput_SkipInput_Disabled tests that interfaces appearing ONLY
// in input params ARE flagged when skipifusedasinput=false.
//
// Purpose: Verify skipifusedasinput=false treats input params as exported output.
// Inputs: Config with ownership.contractscope=exportedoutput, skipifusedasinput=false.
// Expected: IFG001-OWNERSHIP diagnostic - input params count as exported output.
// Edge cases: Override of default skipifusedasinput behavior.
// Dependencies: ownership_expout_inputonly_incl fixture.
func TestOwnership_ExportedOutput_SkipInput_Disabled(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	scope := config.ContractScopeExportedOutput
	cfg.Ownership.ContractScope = &scope
	cfg.Ownership.SkipIfUsedAsInput = false

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	// Expect IFG001-OWNERSHIP diagnostic - input params count as exported output when flag is false.
	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/ownership_expout_inputonly_incl")
}

// TestOwnership_ExcludeTypes tests that excluded type masks suppress ownership diagnostics.
//
// Purpose: Verify global exclude.types suppresses mock implementers.
// Inputs: Config with ownership.contractscope=exportedoutput and exclude.types mask.
// Expected: No diagnostics when the only implementer is excluded.
// Edge cases: Interface is contractual via exported result, but implementer is excluded.
// Dependencies: ownership_ignoreimpl fixture.
func TestOwnership_ExcludeTypes(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	scope := config.ContractScopeExportedOutput
	cfg.Ownership.ContractScope = &scope
	cfg.Exclude.Types = []string{"^ifaceguard-testdata/ownership_ignoreimpl\\.MockExternalAPI$"}

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	// No diagnostics expected - only local implementer is ignored by mask.
	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/ownership_ignoreimpl")
}

// TestOwnership_ExcludeFiles tests that excluded file masks suppress ownership diagnostics.
//
// Purpose: Verify global exclude.files suppresses mock implementations in *_mock.go.
// Inputs: Config with ownership.contractscope=exportedoutput and exclude.files mask.
// Expected: No diagnostics when the only implementer is in an excluded file.
// Edge cases: Interface is contractual via exported result, but mock file is excluded.
// Dependencies: ownership_ignorefile fixture.
func TestOwnership_ExcludeFiles(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	scope := config.ContractScopeExportedOutput
	cfg.Ownership.ContractScope = &scope
	cfg.Exclude.Files = []string{".*_mock\\.go$"}

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	// No diagnostics expected - implementer resides in excluded file.
	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/ownership_ignorefile")
}

// TestOwnership_ContractPackages_Excluded verifies contract package allowlist suppression.
func TestOwnership_ContractPackages_Excluded(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	scope := config.ContractScopeExportedOutput
	cfg.Ownership.ContractScope = &scope
	cfg.Ownership.ContractPackages = []string{"^ifaceguard-testdata/ownership_contractpkg$"}

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/ownership_contractpkg")
}

// TestOwnership_IgnoreInterfaces_Excluded verifies ignoreinterfaces suppression.
func TestOwnership_IgnoreInterfaces_Excluded(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	scope := config.ContractScopeExportedOutput
	cfg.Ownership.ContractScope = &scope
	cfg.Ownership.IgnoreInterfaces = []string{"^ifaceguard-testdata/ownership_ignoreinterfaces\\.Ignored$"}

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/ownership_ignoreinterfaces")
}

// TestOwnership_IgnoreMarkerInterfaces_Disabled verifies marker interfaces are checked when disabled.
func TestOwnership_IgnoreMarkerInterfaces_Disabled(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	scope := config.ContractScopeExportedOutput
	cfg.Ownership.ContractScope = &scope
	cfg.Ownership.IgnoreMarkerInterfaces = false

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/ownership_marker_included")
}

// TestOwnership_Disabled verifies that ownership diagnostics are suppressed when disabled.
func TestOwnership_Disabled(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	cfg.Ownership.Enabled = false

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/ownership_disabled")
}

// =============================================================================
// Ownership: contractscope=anyexported tests
// =============================================================================

// TestOwnership_AnyExported_Basic tests anyexported mode: any exported interface is contractual.
// - Exported interface with local implementer => IFG001-OWNERSHIP expected.
// - Unexported interface with local implementer => no diagnostic.
func TestOwnership_AnyExported_Basic(t *testing.T) {
	t.Parallel()

	testdataDir := moduleTestdataDir(t)
	cfg := config.Default()
	scope := config.ContractScopeAnyExported
	cfg.Ownership.ContractScope = &scope

	a, err := analyzer.New(cfg)
	require.NoError(t, err)

	// Expect IFG001-OWNERSHIP diagnostic on exported interface with local implementer.
	// Unexported interface should NOT produce diagnostic.
	analysistest.Run(t, testdataDir, a, "ifaceguard-testdata/ownership_anyexp_basic")
}
