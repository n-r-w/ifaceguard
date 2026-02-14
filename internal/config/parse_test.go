package config_test

import (
	"testing"

	"github.com/n-r-w/ifaceguard/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFromAny_NilSettings(t *testing.T) {
	t.Parallel()

	cfg, err := config.ParseFromAny(nil)
	require.NoError(t, err)

	// Should return defaults.
	expected := config.Default()
	assert.Equal(t, expected, cfg)
}

func TestParseFromAny_EmptyMap(t *testing.T) {
	t.Parallel()

	cfg, err := config.ParseFromAny(map[string]any{})
	require.NoError(t, err)

	// Should return defaults.
	expected := config.Default()
	assert.Equal(t, expected, cfg)
}

func TestParseFromAny_InvalidSettingsType(t *testing.T) {
	t.Parallel()

	_, err := config.ParseFromAny("invalid")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expected map[string]any")
}

func TestParseFromAny_OwnershipEnabled(t *testing.T) {
	t.Parallel()

	settings := map[string]any{
		"ownership": map[string]any{
			"enabled": false,
		},
	}

	cfg, err := config.ParseFromAny(settings)
	require.NoError(t, err)

	assert.False(t, cfg.Ownership.Enabled)
	// Other fields should be defaults.
	assert.True(t, cfg.Assertions.Enabled)
}

func TestParseFromAny_ContractScopeValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected config.ContractScope
	}{
		{"exportedoutput", "exportedoutput", config.ContractScopeExportedOutput},
		{"anyexported", "anyexported", config.ContractScopeAnyExported},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			settings := map[string]any{
				"ownership": map[string]any{
					"contractscope": tc.input,
				},
			}

			cfg, err := config.ParseFromAny(settings)
			require.NoError(t, err)
			require.NotNil(t, cfg.Ownership.ContractScope)
			assert.Equal(t, tc.expected, *cfg.Ownership.ContractScope)
		})
	}
}

func TestParseFromAny_ContractScopeInvalid(t *testing.T) {
	t.Parallel()

	settings := map[string]any{
		"ownership": map[string]any{
			"contractscope": "invalid",
		},
	}

	_, err := config.ParseFromAny(settings)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid contractscope")
	assert.Contains(t, err.Error(), "exportedoutput")
	assert.Contains(t, err.Error(), "anyexported")
}

func TestParseFromAny_CaseInsensitiveKeys(t *testing.T) {
	t.Parallel()

	// Viper lowercases all keys, so we test lowercase only.
	// But also verify case-insensitive matching for mixed case.
	settings := map[string]any{
		"OWNERSHIP": map[string]any{
			"ENABLED":       false,
			"CONTRACTSCOPE": "exportedoutput",
		},
	}

	cfg, err := config.ParseFromAny(settings)
	require.NoError(t, err)
	assert.False(t, cfg.Ownership.Enabled)
	require.NotNil(t, cfg.Ownership.ContractScope)
	assert.Equal(t, config.ContractScopeExportedOutput, *cfg.Ownership.ContractScope)
}

func TestParseFromAny_UnknownTopLevelKeys(t *testing.T) {
	t.Parallel()

	settings := map[string]any{
		"ruleb": map[string]any{
			"requireassertions": true,
		},
	}

	_, err := config.ParseFromAny(settings)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown keys in settings")
	assert.Contains(t, err.Error(), "ruleb")
	assert.Contains(t, err.Error(), "ownership")
	assert.Contains(t, err.Error(), "assertions")
	assert.Contains(t, err.Error(), "constructors")
	assert.Contains(t, err.Error(), "exclude")
}

func TestParseFromAny_UnknownAssertionsKeys(t *testing.T) {
	t.Parallel()

	settings := map[string]any{
		"assertions": map[string]any{
			"ruleb": true,
		},
	}

	_, err := config.ParseFromAny(settings)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown keys in assertions")
	assert.Contains(t, err.Error(), "ruleb")
	assert.Contains(t, err.Error(), "requireassertions")
	assert.Contains(t, err.Error(), "requireassertionsstrict")
	assert.Contains(t, err.Error(), "checkbypassassertions")
	assert.Contains(t, err.Error(), "scanfunctionbodies")
	assert.Contains(t, err.Error(), "acceptconversiononlyform")
}

func TestParseFromAny_StringSlice(t *testing.T) {
	t.Parallel()

	settings := map[string]any{
		"ownership": map[string]any{
			"contractpackages": []any{"^example\\.com/.*$", "^internal/.*$"},
			"ignoreinterfaces": []string{"example.com/pkg.Ignored"},
		},
	}

	cfg, err := config.ParseFromAny(settings)
	require.NoError(t, err)

	assert.Equal(t, []string{"^example\\.com/.*$", "^internal/.*$"}, cfg.Ownership.ContractPackages)
	assert.Equal(t, []string{"example.com/pkg.Ignored"}, cfg.Ownership.IgnoreInterfaces)
}

func TestParseFromAny_ExcludeSettings(t *testing.T) {
	t.Parallel()

	settings := map[string]any{
		"exclude": map[string]any{
			"files": []string{".*_mock\\.go$"},
			"types": []string{".*\\.Mock.*$"},
		},
	}

	cfg, err := config.ParseFromAny(settings)
	require.NoError(t, err)

	assert.Equal(t, []string{".*_mock\\.go$"}, cfg.Exclude.Files)
	assert.Equal(t, []string{".*\\.Mock.*$"}, cfg.Exclude.Types)
}

func TestParseFromAny_AssertionsSettings(t *testing.T) {
	t.Parallel()

	settings := map[string]any{
		"assertions": map[string]any{
			"enabled":                  false,
			"wiringpackages":           []string{"^cmd/.*$"},
			"acceptconversiononlyform": true,
			"scanfunctionbodies":       true,
			"requireassertions":        true,
			"requireassertionsstrict":  true,
			"checkbypassassertions":    false,
		},
	}

	cfg, err := config.ParseFromAny(settings)
	require.NoError(t, err)

	assert.False(t, cfg.Assertions.Enabled)
	assert.Equal(t, []string{"^cmd/.*$"}, cfg.Assertions.WiringPackages)
	assert.True(t, cfg.Assertions.AcceptConversionOnlyForm)
	assert.True(t, cfg.Assertions.ScanFunctionBodies)
	assert.True(t, cfg.Assertions.RequireAssertions)
	assert.True(t, cfg.Assertions.RequireAssertionsStrict)
	assert.False(t, cfg.Assertions.CheckBypassAssertions)
}

func TestParseFromAny_ConstructorsSettings(t *testing.T) {
	t.Parallel()

	settings := map[string]any{
		"constructors": map[string]any{
			"enabled":           true,
			"namepatterns":      []string{"^Factory[A-Z]", "^Build[A-Z]"},
			"exportedonly":      false,
			"ignoreinterfaces":  []string{"^example\\.com/project/iface\\.Allowed$"},
			"ignoreerrorreturn": false,
		},
	}

	cfg, err := config.ParseFromAny(settings)
	require.NoError(t, err)

	assert.True(t, cfg.Constructors.Enabled)
	assert.Equal(t, []string{"^Factory[A-Z]", "^Build[A-Z]"}, cfg.Constructors.NamePatterns)
	assert.False(t, cfg.Constructors.ExportedOnly)
	assert.Equal(t, []string{"^example\\.com/project/iface\\.Allowed$"}, cfg.Constructors.IgnoreInterfaces)
	assert.False(t, cfg.Constructors.IgnoreErrorReturn)
}

func TestParseFromAny_UnknownConstructorsKeys(t *testing.T) {
	t.Parallel()

	settings := map[string]any{
		"constructors": map[string]any{
			"unknown": true,
		},
	}

	_, err := config.ParseFromAny(settings)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown keys in constructors")
	assert.Contains(t, err.Error(), "unknown")
	assert.Contains(t, err.Error(), "namepatterns")
	assert.Contains(t, err.Error(), "ignoreerrorreturn")
}

func TestParseFromAny_AllSettings(t *testing.T) {
	t.Parallel()

	settings := map[string]any{
		"ownership": map[string]any{
			"enabled":                true,
			"contractscope":          "anyexported",
			"skipifusedasinput":      false,
			"contractpackages":       []string{"^pkg/contract/.*$"},
			"ignoreinterfaces":       []string{".*\\.Error$"},
			"ignoremarkerinterfaces": false,
		},
		"assertions": map[string]any{
			"enabled":                  true,
			"wiringpackages":           []string{"^cmd/.*$", "^internal/wiring$"},
			"acceptconversiononlyform": true,
			"scanfunctionbodies":       true,
			"requireassertions":        true,
			"requireassertionsstrict":  true,
			"checkbypassassertions":    true,
		},
		"constructors": map[string]any{
			"enabled":           true,
			"namepatterns":      []string{"^New[A-Z]", "^MustNew[A-Z]"},
			"exportedonly":      true,
			"ignoreinterfaces":  []string{"^example\\.com/project/iface\\.Allowed$"},
			"ignoreerrorreturn": true,
		},
		"exclude": map[string]any{
			"files": []string{".*_mock\\.go$"},
			"types": []string{".*\\.Mock.*$"},
		},
	}

	cfg, err := config.ParseFromAny(settings)
	require.NoError(t, err)

	// Verify ownership settings.
	assert.True(t, cfg.Ownership.Enabled)
	require.NotNil(t, cfg.Ownership.ContractScope)
	assert.Equal(t, config.ContractScopeAnyExported, *cfg.Ownership.ContractScope)
	assert.False(t, cfg.Ownership.SkipIfUsedAsInput)
	assert.Equal(t, []string{"^pkg/contract/.*$"}, cfg.Ownership.ContractPackages)
	assert.Equal(t, []string{".*\\.Error$"}, cfg.Ownership.IgnoreInterfaces)
	assert.False(t, cfg.Ownership.IgnoreMarkerInterfaces)

	// Verify assertions settings.
	assert.True(t, cfg.Assertions.Enabled)
	assert.Equal(t, []string{"^cmd/.*$", "^internal/wiring$"}, cfg.Assertions.WiringPackages)
	assert.True(t, cfg.Assertions.AcceptConversionOnlyForm)
	assert.True(t, cfg.Assertions.ScanFunctionBodies)
	assert.True(t, cfg.Assertions.RequireAssertions)
	assert.True(t, cfg.Assertions.RequireAssertionsStrict)
	assert.True(t, cfg.Assertions.CheckBypassAssertions)

	// Verify constructors settings.
	assert.True(t, cfg.Constructors.Enabled)
	assert.Equal(t, []string{"^New[A-Z]", "^MustNew[A-Z]"}, cfg.Constructors.NamePatterns)
	assert.True(t, cfg.Constructors.ExportedOnly)
	assert.Equal(t, []string{"^example\\.com/project/iface\\.Allowed$"}, cfg.Constructors.IgnoreInterfaces)
	assert.True(t, cfg.Constructors.IgnoreErrorReturn)

	// Verify Exclude.
	assert.Equal(t, []string{".*_mock\\.go$"}, cfg.Exclude.Files)
	assert.Equal(t, []string{".*\\.Mock.*$"}, cfg.Exclude.Types)
}
