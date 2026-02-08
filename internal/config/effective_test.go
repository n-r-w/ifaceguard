package config_test

import (
	"testing"

	"github.com/n-r-w/ifaceguard/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	cfg := config.Config{
		Ownership: config.OwnershipConfig{
			Enabled:                false,
			ContractScope:          nil,
			SkipIfUsedAsInput:      false,
			ContractPackages:       []string{"pkg1", "pkg2"},
			IgnoreInterfaces:       []string{"iface1"},
			IgnoreMarkerInterfaces: false,
		},
		Assertions: config.AssertionsConfig{
			Enabled:                  false,
			WiringPackages:           []string{"wiring"},
			AcceptConversionOnlyForm: true,
			ScanFunctionBodies:       true,
			RequireAssertions:        true,
		},
		Exclude: config.ExcludeConfig{
			Files: []string{".*_mock\\.go$"},
			Types: []string{".*\\.Mock.*$"},
		},
	}

	effective := cfg.ResolveEffective()

	// Verify ownership fields.
	assert.False(t, effective.Ownership.Enabled)
	assert.False(t, effective.Ownership.SkipIfUsedAsInput)
	assert.Equal(t, []string{"pkg1", "pkg2"}, effective.Ownership.ContractPackages)
	assert.Equal(t, []string{"iface1"}, effective.Ownership.IgnoreInterfaces)
	assert.False(t, effective.Ownership.IgnoreMarkerInterfaces)

	// Verify assertions config is copied as-is.
	assert.Equal(t, cfg.Assertions, effective.Assertions)

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

	cfg := config.Config{
		Ownership: config.OwnershipConfig{
			Enabled:                true,
			ContractScope:          nil,
			SkipIfUsedAsInput:      false,
			ContractPackages:       []string{`^pkg/contract$`},
			IgnoreInterfaces:       []string{`\.Internal$`},
			IgnoreMarkerInterfaces: false,
		},
		Assertions: config.AssertionsConfig{
			Enabled:                  true,
			WiringPackages:           []string{`^pkg/wiring$`},
			AcceptConversionOnlyForm: false,
			ScanFunctionBodies:       false,
			RequireAssertions:        false,
		},
		Exclude: config.ExcludeConfig{
			Files: []string{`.*_mock\.go$`},
			Types: []string{`\.Mock.*$`},
		},
	}

	effective := cfg.ResolveEffective()
	compiled, err := effective.Compile()
	require.NoError(t, err)

	// Verify regex patterns are compiled.
	assert.Len(t, compiled.Ownership.ContractPackages, 1)
	assert.Len(t, compiled.Ownership.IgnoreInterfaces, 1)
	assert.Len(t, compiled.Assertions.WiringPackages, 1)
	assert.Len(t, compiled.Exclude.Files, 1)
	assert.Len(t, compiled.Exclude.Types, 1)

	// Verify patterns match as expected.
	assert.True(t, compiled.Ownership.ContractPackages[0].MatchString("pkg/contract"))
	assert.False(t, compiled.Ownership.ContractPackages[0].MatchString("other/pkg"))

	assert.True(t, compiled.Ownership.IgnoreInterfaces[0].MatchString("pkg.Internal"))
	assert.False(t, compiled.Ownership.IgnoreInterfaces[0].MatchString("pkg.Public"))
	assert.True(t, compiled.Exclude.Files[0].MatchString("adapter/interface_mock.go"))
	assert.True(t, compiled.Exclude.Types[0].MatchString("pkg.MockService"))
}

func TestEffectiveConfig_Compile_InvalidRegex_ReturnsError(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Ownership: config.OwnershipConfig{
			Enabled:                true,
			ContractScope:          nil,
			SkipIfUsedAsInput:      false,
			ContractPackages:       []string{`[invalid`}, // invalid regex
			IgnoreInterfaces:       nil,
			IgnoreMarkerInterfaces: false,
		},
		Assertions: config.AssertionsConfig{
			Enabled:                  false,
			WiringPackages:           nil,
			AcceptConversionOnlyForm: false,
			ScanFunctionBodies:       false,
			RequireAssertions:        false,
		},
		Exclude: config.ExcludeConfig{
			Files: nil,
			Types: nil,
		},
	}

	effective := cfg.ResolveEffective()
	_, err := effective.Compile()
	assert.Error(t, err)
}
