package testhelpers

import (
	"path/filepath"
	"testing"

	"github.com/n-r-w/ifaceguard/internal/analyzer"
	"github.com/n-r-w/ifaceguard/internal/config"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/analysistest"
)

// BuildAnalyzer creates an analyzer instance for testing with the given config.
// This wraps analyzer.New and fails the test on error.
func BuildAnalyzer(t *testing.T, cfg config.Config) *analysis.Analyzer {
	t.Helper()

	a, err := analyzer.New(cfg)
	if err != nil {
		t.Fatalf("failed to build analyzer: %v", err)
	}
	return a
}

// RunAnalysisTest runs analysistest.Run with the given analyzer and patterns.
// testdataDir should be the absolute path to the testdata directory.
// patterns are module-qualified package patterns (e.g., "ifaceguard-testdata/baseline").
func RunAnalysisTest(t *testing.T, testdataDir string, a *analysis.Analyzer, patterns ...string) {
	t.Helper()
	analysistest.Run(t, testdataDir, a, patterns...)
}

// TestdataDir returns the absolute path to the testdata directory for the analyzer package.
//
//nolint:paralleltest // TestdataDir is a helper and does not define a test.
func TestdataDir(t *testing.T) string {
	t.Helper()

	// Return the module root for analyzer fixtures (testdata/src).
	return filepath.Join(analysistest.TestData(), "src")
}

// TestBuildAnalyzer_Success verifies that BuildAnalyzer creates a valid analyzer.
func TestBuildAnalyzer_Success(t *testing.T) {
	t.Parallel()

	cfg := config.Default()
	a := BuildAnalyzer(t, cfg)

	if a == nil {
		t.Fatal("expected analyzer to be created")
	}

	if a.Name != analyzer.AnalyzerName {
		t.Errorf("expected analyzer name %q, got %q", analyzer.AnalyzerName, a.Name)
	}
}
