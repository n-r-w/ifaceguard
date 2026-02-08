package driver

import (
	"io"
	"os"
	"path/filepath"
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
