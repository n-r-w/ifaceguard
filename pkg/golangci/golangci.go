// Package golangci provides integration with golangci-lint's module plugin system.
//
// This package registers the ifaceguard analyzer as a golangci-lint plugin.
// When imported, it automatically registers the plugin via init().
//
// Usage in .custom-gcl.yml:
//
//	version: v2.8.0
//	plugins:
//	  - module: 'github.com/n-r-w/ifaceguard'
//	    import: 'github.com/n-r-w/ifaceguard/pkg/golangci'
//	    version: v0.1.0
//
// Usage in .golangci.yml:
//
//	version: "2"
//	linters:
//	  enable:
//	    - ifaceguard
//	  settings:
//	    custom:
//	      ifaceguard:
//	        type: "module"
//	        settings:
//	          ownership:
//	            enabled: true
//	            contractscope: exportedoutput
//	          assertions:
//	            enabled: true
package golangci

import (
	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"

	"github.com/n-r-w/ifaceguard/internal/analyzer"
	"github.com/n-r-w/ifaceguard/internal/config"
)

//nolint:gochecknoinits // init() is required for golangci-lint module plugin registration
func init() {
	register.Plugin(analyzer.AnalyzerName, newPlugin)
}

// plugin implements register.LinterPlugin interface for golangci-lint integration.
type plugin struct {
	cfg config.Config
}

// newPlugin creates a new ifaceguard plugin instance from golangci-lint settings.
func newPlugin(settings any) (register.LinterPlugin, error) {
	cfg, err := config.ParseFromAny(settings)
	if err != nil {
		return nil, err
	}

	return &plugin{cfg: cfg}, nil
}

// BuildAnalyzers returns the ifaceguard analyzer.
func (p *plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return buildAnalyzers(p.cfg)
}

// GetLoadMode returns the load mode required by the analyzer.
// ifaceguard requires type information for interface/implementation analysis.
func (p *plugin) GetLoadMode() string {
	return register.LoadModeTypesInfo
}

func buildAnalyzers(cfg config.Config) ([]*analysis.Analyzer, error) {
	// golangci-lint runs analyzers per-package without workspace context.
	a, err := analyzer.New(cfg)
	if err != nil {
		return nil, err
	}

	return []*analysis.Analyzer{a}, nil
}
