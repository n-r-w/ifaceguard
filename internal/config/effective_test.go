package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/n-r-w/ifaceguard/internal/config"
)

func TestResolveEffective_UnsetScope_DefaultsToExportedOutput(t *testing.T) {
	t.Parallel()

	cfg := config.Default()
	// ContractScope is nil (unset).

	effective := cfg.ResolveEffective()

	// Unset scope => exportedoutput.
	assert.Equal(t, config.ContractScopeExportedOutput, effective.Ownership.ContractScope)
}

func TestResolveEffective_ExplicitScope_Honored(t *testing.T) {
	t.Parallel()

	// Test each explicit scope value.
	scopes := []config.ContractScope{
		config.ContractScopeExportedOutput,
		config.ContractScopeAnyExported,
	}

	for _, scope := range scopes {
		t.Run(string(scope), func(t *testing.T) {
			t.Parallel()

			cfg := config.Default()
			s := scope
			cfg.Ownership.ContractScope = &s

			// Explicit scope should be honored.
			effective := cfg.ResolveEffective()
			assert.Equal(t, scope, effective.Ownership.ContractScope)
		})
	}
}

func TestResolveEffective_OtherFieldsCopied(t *testing.T) {
	t.Parallel()

	cfg := config.Default()
	cfg.Ownership.Enabled = false
	cfg.Ownership.SkipIfUsedAsInput = false
	cfg.Ownership.ContractPackages = []string{"pkg1", "pkg2"}
	cfg.Ownership.IgnoreInterfaces = []string{"iface1"}
	cfg.Ownership.IgnoreMarkerInterfaces = false
	cfg.Assertions.Enabled = false
	cfg.Assertions.WiringPackages = []string{"wiring"}
	cfg.Assertions.AcceptConversionOnlyForm = true
	cfg.Assertions.ScanFunctionBodies = true
	cfg.Assertions.RequireAssertions = true
	cfg.Assertions.RequireAssertionsStrict = true
	cfg.Assertions.CheckBypassAssertions = true
	cfg.Constructors.Enabled = true
	cfg.Constructors.NamePatterns = []string{"^Factory[A-Z]"}
	cfg.Constructors.ExportedOnly = false
	cfg.Constructors.IgnoreInterfaces = []string{"ifaceguard-testdata/pkg.Iface"}
	cfg.Constructors.IgnoreErrorReturn = false
	cfg.Exclude.Files = []string{".*_mock\\.go$"}
	cfg.Exclude.Types = []string{".*\\.Mock.*$"}

	effective := cfg.ResolveEffective()

	// Verify ownership fields.
	assert.False(t, effective.Ownership.Enabled)
	assert.False(t, effective.Ownership.SkipIfUsedAsInput)
	assert.Equal(t, []string{"pkg1", "pkg2"}, effective.Ownership.ContractPackages)
	assert.Equal(t, []string{"iface1"}, effective.Ownership.IgnoreInterfaces)
	assert.False(t, effective.Ownership.IgnoreMarkerInterfaces)

	// Verify assertions config is copied as-is.
	assert.Equal(t, cfg.Assertions, effective.Assertions)

	// Verify constructors config is copied as-is.
	assert.Equal(t, cfg.Constructors, effective.Constructors)

	// Verify Exclude is copied as-is.
	assert.Equal(t, cfg.Exclude, effective.Exclude)
}

func TestEffectiveConfig_Compile_ScopeResolved(t *testing.T) {
	t.Parallel()

	cfg := config.Default()
	// ContractScope is nil (unset).

	effective := cfg.ResolveEffective()

	compiled, err := effective.Compile()
	require.NoError(t, err)

	// Compiled config should have non-nil ContractScope pointing to exportedoutput.
	assert.NotNil(t, compiled.Ownership.ContractScope)
	assert.Equal(t, config.ContractScopeExportedOutput, *compiled.Ownership.ContractScope)
}

func TestEffectiveConfig_Compile_RegexPatterns(t *testing.T) {
	t.Parallel()

	cfg := config.Default()
	cfg.Ownership.SkipIfUsedAsInput = false
	cfg.Ownership.ContractPackages = []string{`^pkg/contract$`}
	cfg.Ownership.IgnoreInterfaces = []string{`\.Internal$`}
	cfg.Ownership.IgnoreMarkerInterfaces = false
	cfg.Assertions.WiringPackages = []string{`^pkg/wiring$`}
	cfg.Constructors.Enabled = true
	cfg.Constructors.NamePatterns = []string{`^New[A-Z]`}
	cfg.Constructors.IgnoreInterfaces = []string{`\.Allowed$`}
	cfg.Exclude.Files = []string{`.*_mock\.go$`}
	cfg.Exclude.Types = []string{`\.Mock.*$`}

	effective := cfg.ResolveEffective()
	compiled, err := effective.Compile()
	require.NoError(t, err)

	// Verify regex patterns are compiled.
	assert.Len(t, compiled.Ownership.ContractPackages, 1)
	assert.Len(t, compiled.Ownership.IgnoreInterfaces, 1)
	assert.Len(t, compiled.Assertions.WiringPackages, 1)
	assert.Len(t, compiled.Constructors.NamePatterns, 1)
	assert.Len(t, compiled.Constructors.IgnoreInterfaces, 1)
	assert.Len(t, compiled.Exclude.Files, 1)
	assert.Len(t, compiled.Exclude.Types, 1)

	// Verify patterns match as expected.
	assert.True(t, compiled.Ownership.ContractPackages[0].MatchString("pkg/contract"))
	assert.False(t, compiled.Ownership.ContractPackages[0].MatchString("other/pkg"))

	assert.True(t, compiled.Ownership.IgnoreInterfaces[0].MatchString("pkg.Internal"))
	assert.False(t, compiled.Ownership.IgnoreInterfaces[0].MatchString("pkg.Public"))
	assert.True(t, compiled.Constructors.NamePatterns[0].MatchString("NewService"))
	assert.False(t, compiled.Constructors.NamePatterns[0].MatchString("BuildService"))
	assert.True(t, compiled.Constructors.IgnoreInterfaces[0].MatchString("foo.Allowed"))
	assert.True(t, compiled.Exclude.Files[0].MatchString("adapter/interface_mock.go"))
	assert.True(t, compiled.Exclude.Types[0].MatchString("pkg.MockService"))
}

func TestEffectiveConfig_Compile_InvalidRegex_ReturnsError(t *testing.T) {
	t.Parallel()

	cfg := config.Default()
	cfg.Ownership.SkipIfUsedAsInput = false
	cfg.Ownership.ContractPackages = []string{`[invalid`} // invalid regex
	cfg.Ownership.IgnoreMarkerInterfaces = false
	cfg.Assertions.Enabled = false
	cfg.Constructors.Enabled = false

	effective := cfg.ResolveEffective()
	_, err := effective.Compile()
	assert.Error(t, err)
}
