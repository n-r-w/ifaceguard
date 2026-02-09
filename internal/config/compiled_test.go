package config_test

import (
	"testing"

	"github.com/n-r-w/ifaceguard/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompile_Success(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Ownership: config.OwnershipConfig{
			Enabled:                true,
			ContractScope:          nil,
			SkipIfUsedAsInput:      true,
			ContractPackages:       []string{"^example\\.com/.*$", "^internal/.*$"},
			IgnoreInterfaces:       []string{".*\\.Error$"},
			IgnoreMarkerInterfaces: true,
		},
		Assertions: config.AssertionsConfig{
			Enabled:                  true,
			WiringPackages:           []string{"^cmd/.*$"},
			AcceptConversionOnlyForm: false,
			ScanFunctionBodies:       false,
			RequireAssertions:        false,
			RequireAssertionsStrict:  false,
		},
		Exclude: config.ExcludeConfig{
			Files: []string{".*_mock\\.go$"},
			Types: []string{".*\\.Mock.*$"},
		},
	}

	compiled, err := cfg.Compile()
	require.NoError(t, err)

	// Verify ownership config compiled.
	assert.True(t, compiled.Ownership.Enabled)
	assert.Len(t, compiled.Ownership.ContractPackages, 2)
	assert.Len(t, compiled.Ownership.IgnoreInterfaces, 1)

	// Verify regexes match as expected.
	assert.True(t, compiled.Ownership.ContractPackages[0].MatchString("example.com/pkg"))
	assert.True(t, compiled.Ownership.ContractPackages[1].MatchString("internal/foo"))
	assert.True(t, compiled.Ownership.IgnoreInterfaces[0].MatchString("pkg.Error"))

	// Verify exclude patterns compiled.
	assert.Len(t, compiled.Exclude.Files, 1)
	assert.Len(t, compiled.Exclude.Types, 1)
	assert.True(t, compiled.Exclude.Files[0].MatchString("adapter/interface_mock.go"))
	assert.True(t, compiled.Exclude.Types[0].MatchString("pkg.MockService"))

	// Verify assertions config compiled.
	assert.True(t, compiled.Assertions.Enabled)
	assert.Len(t, compiled.Assertions.WiringPackages, 1)
	assert.True(t, compiled.Assertions.WiringPackages[0].MatchString("cmd/main"))
}

func TestCompile_InvalidOwnershipContractPackages(t *testing.T) {
	t.Parallel()

	cfg := config.Default()
	cfg.Ownership.ContractPackages = []string{"[invalid"}

	_, err := cfg.Compile()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ownership")
	assert.Contains(t, err.Error(), "contractpackages[0]")
	assert.Contains(t, err.Error(), "[invalid")
}

func TestCompile_InvalidOwnershipIgnoreInterfaces(t *testing.T) {
	t.Parallel()

	cfg := config.Default()
	cfg.Ownership.IgnoreInterfaces = []string{"valid", "(unclosed"}

	_, err := cfg.Compile()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ownership")
	assert.Contains(t, err.Error(), "ignoreinterfaces[1]")
	assert.Contains(t, err.Error(), "(unclosed")
}

func TestCompile_InvalidExcludeTypes(t *testing.T) {
	t.Parallel()

	cfg := config.Default()
	cfg.Exclude.Types = []string{"valid", "[invalid"}

	_, err := cfg.Compile()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exclude")
	assert.Contains(t, err.Error(), "types[1]")
	assert.Contains(t, err.Error(), "[invalid")
}

func TestCompile_InvalidExcludeFiles(t *testing.T) {
	t.Parallel()

	cfg := config.Default()
	cfg.Exclude.Files = []string{"[invalid"}

	_, err := cfg.Compile()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exclude")
	assert.Contains(t, err.Error(), "files[0]")
	assert.Contains(t, err.Error(), "[invalid")
}

func TestCompile_InvalidAssertionsWiringPackages(t *testing.T) {
	t.Parallel()

	cfg := config.Default()
	cfg.Assertions.WiringPackages = []string{"**invalid**"}

	_, err := cfg.Compile()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "assertions")
	assert.Contains(t, err.Error(), "wiringpackages[0]")
}

func TestCompile_EmptyPatterns(t *testing.T) {
	t.Parallel()

	cfg := config.Default()
	compiled, err := cfg.Compile()
	require.NoError(t, err)

	// Empty slices should compile to nil.
	assert.Nil(t, compiled.Ownership.ContractPackages)
	assert.Nil(t, compiled.Ownership.IgnoreInterfaces)
	assert.Nil(t, compiled.Assertions.WiringPackages)
	assert.Nil(t, compiled.Exclude.Files)
	assert.Nil(t, compiled.Exclude.Types)
}
