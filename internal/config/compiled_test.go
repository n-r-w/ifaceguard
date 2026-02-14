package config_test

import (
	"testing"

	"github.com/n-r-w/ifaceguard/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompile_Success(t *testing.T) {
	t.Parallel()

	cfg := config.Default()
	cfg.Ownership.ContractPackages = []string{"^example\\.com/.*$", "^internal/.*$"}
	cfg.Ownership.IgnoreInterfaces = []string{".*\\.Error$"}
	cfg.Assertions.WiringPackages = []string{"^cmd/.*$"}
	cfg.Constructors.Enabled = true
	cfg.Constructors.IgnoreInterfaces = []string{".*\\.Allowed$"}
	cfg.Exclude.Files = []string{".*_mock\\.go$"}
	cfg.Exclude.Types = []string{".*\\.Mock.*$"}

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

	// Verify constructors config compiled.
	assert.True(t, compiled.Constructors.Enabled)
	assert.Len(t, compiled.Constructors.NamePatterns, 2)
	assert.True(t, compiled.Constructors.NamePatterns[0].MatchString("NewService"))
	assert.False(t, compiled.Constructors.NamePatterns[0].MatchString("BuildService"))
	assert.Len(t, compiled.Constructors.IgnoreInterfaces, 1)
	assert.True(t, compiled.Constructors.IgnoreInterfaces[0].MatchString("foo.Allowed"))
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

func TestCompile_InvalidConstructorsNamePatterns(t *testing.T) {
	t.Parallel()

	cfg := config.Default()
	cfg.Constructors.NamePatterns = []string{"[invalid"}

	_, err := cfg.Compile()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "constructors")
	assert.Contains(t, err.Error(), "namepatterns[0]")
	assert.Contains(t, err.Error(), "[invalid")
}

func TestCompile_EmptyPatterns(t *testing.T) {
	t.Parallel()

	cfg := config.Default()
	compiled, err := cfg.Compile()
	require.NoError(t, err)

	// Empty slices should compile to nil, except constructor namepatterns
	// which come from defaults.
	assert.Nil(t, compiled.Ownership.ContractPackages)
	assert.Nil(t, compiled.Ownership.IgnoreInterfaces)
	assert.Nil(t, compiled.Assertions.WiringPackages)
	assert.NotNil(t, compiled.Constructors.NamePatterns)
	assert.Len(t, compiled.Constructors.NamePatterns, 2)
	assert.Nil(t, compiled.Constructors.IgnoreInterfaces)
	assert.Nil(t, compiled.Exclude.Files)
	assert.Nil(t, compiled.Exclude.Types)
}
