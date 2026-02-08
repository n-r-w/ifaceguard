//go:build plugin

// Package main provides integration with golangci-lint's Go plugin system (.so plugins).
//
// Build this file as a Go plugin:
//
//	CGO_ENABLED=1 go build -tags plugin -buildmode=plugin -o ifaceguard.so ./plugin/
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
//	        path: ./ifaceguard.so
//	        settings:
//	          ownership:
//	            enabled: true
//	            contractscope: exportedoutput
//	          assertions:
//	            enabled: true
//
// Note: The plugin and golangci-lint binary must be built with the same Go version
// and matching dependency versions. CGO_ENABLED=1 is required.
package main

import (
	"github.com/n-r-w/ifaceguard/internal/analyzer"
	"github.com/n-r-w/ifaceguard/internal/config"
	"golang.org/x/tools/go/analysis"
)

// New creates ifaceguard analyzers from golangci-lint configuration.
// This function is required by golangci-lint's Go plugin system.
func New(settings any) ([]*analysis.Analyzer, error) {
	cfg, err := config.ParseFromAny(settings)
	if err != nil {
		return nil, err
	}

	return buildAnalyzers(cfg)
}

func buildAnalyzers(cfg config.Config) ([]*analysis.Analyzer, error) {
	// golangci-lint runs analyzers per-package without workspace context.
	a, err := analyzer.New(cfg)
	if err != nil {
		return nil, err
	}

	return []*analysis.Analyzer{a}, nil
}
