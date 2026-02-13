package driver

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/n-r-w/ifaceguard/internal/analyzer"
	"github.com/n-r-w/ifaceguard/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_Default(t *testing.T) {
	t.Parallel()

	got, err := loadConfig("")
	require.NoError(t, err)

	assert.Equal(t, config.Default(), got)
}

func TestLoadConfig_FromFile(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "ifaceguard.yml")
	cleanPath := filepath.Clean(path)
	yaml := `ownership:
  enabled: false
assertions:
  enabled: false
exclude:
  files:
    - ".*_mock\\.go$"
`

	require.NoError(t, os.WriteFile(cleanPath, []byte(yaml), 0o600))

	got, err := loadConfig(cleanPath)
	require.NoError(t, err)

	expected := config.Default()
	expected.Ownership.Enabled = false
	expected.Assertions.Enabled = false
	expected.Exclude.Files = []string{".*_mock\\.go$"}

	assert.Equal(t, expected, got)
}

func TestNewFlagSet_ConfigFlagName(t *testing.T) {
	t.Parallel()

	a, err := analyzer.New(config.Default())
	require.NoError(t, err)

	fs, _ := newFlagSet(a, Options{Args: nil, Stdout: io.Discard, Stderr: io.Discard})

	assert.NotNil(t, fs.Lookup("config"))
	assert.Nil(t, fs.Lookup("c"))
}

func TestRun_FailFastOnPackageErrors(t *testing.T) {
	tempDir := t.TempDir()

	goModPath := filepath.Join(tempDir, "go.mod")
	goMod := `module example.com/broken

go 1.25
`
	require.NoError(t, os.WriteFile(goModPath, []byte(goMod), 0o600))

	sourcePath := filepath.Join(tempDir, "broken.go")
	source := `package broken

func f() {
	_ = missingSymbol
}
`
	require.NoError(t, os.WriteFile(sourcePath, []byte(source), 0o600))

	t.Chdir(tempDir)

	a, err := analyzer.New(config.Default())
	require.NoError(t, err)

	var stderr bytes.Buffer
	exitCode := Run(a, Options{
		Args:   []string{"./..."},
		Stdout: io.Discard,
		Stderr: &stderr,
	})

	assert.Equal(t, 1, exitCode)

	output := stderr.String()
	assert.Equal(t, 1, strings.Count(output, "undefined: missingSymbol"))
	assert.NotContains(t, output, "analysis skipped due to errors in package")
}

func TestRun_PrintNoErrorsMessage(t *testing.T) {
	tempDir := t.TempDir()

	goModPath := filepath.Join(tempDir, "go.mod")
	goMod := `module example.com/clean

go 1.25
`
	require.NoError(t, os.WriteFile(goModPath, []byte(goMod), 0o600))

	sourcePath := filepath.Join(tempDir, "clean.go")
	source := `package clean

func Value() int {
	return 42
}
`
	require.NoError(t, os.WriteFile(sourcePath, []byte(source), 0o600))

	t.Chdir(tempDir)

	a, err := analyzer.New(config.Default())
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run(a, Options{
		Args:   []string{"./..."},
		Stdout: &stdout,
		Stderr: &stderr,
	})

	assert.Equal(t, 0, exitCode)
	assert.Equal(t, noErrorsFoundMessage+"\n", stdout.String())
	assert.Empty(t, stderr.String())
}

func TestRun_JSONOutputDoesNotPrintNoErrorsMessage(t *testing.T) {
	tempDir := t.TempDir()

	goModPath := filepath.Join(tempDir, "go.mod")
	goMod := `module example.com/cleanjson

go 1.25
`
	require.NoError(t, os.WriteFile(goModPath, []byte(goMod), 0o600))

	sourcePath := filepath.Join(tempDir, "clean.go")
	source := `package cleanjson

func Value() int {
	return 7
}
`
	require.NoError(t, os.WriteFile(sourcePath, []byte(source), 0o600))

	t.Chdir(tempDir)

	a, err := analyzer.New(config.Default())
	require.NoError(t, err)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := Run(a, Options{
		Args:   []string{"-json", "./..."},
		Stdout: &stdout,
		Stderr: &stderr,
	})

	assert.Equal(t, 0, exitCode)
	assert.NotContains(t, stdout.String(), noErrorsFoundMessage)
	assert.Empty(t, stderr.String())
}
