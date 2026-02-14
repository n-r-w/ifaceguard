package config

import (
	"fmt"
	"regexp"
)

// CompiledOwnershipConfig contains ownership configuration with compiled regexes.
type CompiledOwnershipConfig struct {
	Enabled                bool
	ContractScope          *ContractScope
	SkipIfUsedAsInput      bool
	ContractPackages       []*regexp.Regexp
	IgnoreInterfaces       []*regexp.Regexp
	IgnoreMarkerInterfaces bool
}

// CompiledAssertionsConfig contains assertions configuration with compiled regexes.
type CompiledAssertionsConfig struct {
	Enabled                  bool
	WiringPackages           []*regexp.Regexp
	AcceptConversionOnlyForm bool
	ScanFunctionBodies       bool
	RequireAssertions        bool
	RequireAssertionsStrict  bool
	CheckBypassAssertions    bool
}

// CompiledExcludeConfig contains compiled exclusion patterns.
type CompiledExcludeConfig struct {
	Files []*regexp.Regexp
	Types []*regexp.Regexp
}

// CompiledConfig contains configuration with all regex patterns compiled.
type CompiledConfig struct {
	Ownership  CompiledOwnershipConfig
	Assertions CompiledAssertionsConfig
	Exclude    CompiledExcludeConfig
}

// Compile compiles all regex patterns in the config and returns a CompiledConfig.
// Returns an error if any regex pattern fails to compile.
func (c Config) Compile() (CompiledConfig, error) {
	compiledOwnership, err := compileOwnership(c.Ownership)
	if err != nil {
		return CompiledConfig{}, fmt.Errorf("ownership: %w", err)
	}

	compiledAssertions, err := compileAssertions(c.Assertions)
	if err != nil {
		return CompiledConfig{}, fmt.Errorf("assertions: %w", err)
	}

	compiledExclude, err := compileExclude(c.Exclude)
	if err != nil {
		return CompiledConfig{}, fmt.Errorf("exclude: %w", err)
	}

	return CompiledConfig{
		Ownership:  compiledOwnership,
		Assertions: compiledAssertions,
		Exclude:    compiledExclude,
	}, nil
}

func compileOwnership(ownership OwnershipConfig) (CompiledOwnershipConfig, error) {
	contractPackages, err := compilePatterns(ownership.ContractPackages, "contractpackages")
	if err != nil {
		return CompiledOwnershipConfig{}, err
	}

	ignoreInterfaces, err := compilePatterns(ownership.IgnoreInterfaces, "ignoreinterfaces")
	if err != nil {
		return CompiledOwnershipConfig{}, err
	}

	return CompiledOwnershipConfig{
		Enabled:                ownership.Enabled,
		ContractScope:          ownership.ContractScope,
		SkipIfUsedAsInput:      ownership.SkipIfUsedAsInput,
		ContractPackages:       contractPackages,
		IgnoreInterfaces:       ignoreInterfaces,
		IgnoreMarkerInterfaces: ownership.IgnoreMarkerInterfaces,
	}, nil
}

func compileAssertions(assertions AssertionsConfig) (CompiledAssertionsConfig, error) {
	wiringPackages, err := compilePatterns(assertions.WiringPackages, "wiringpackages")
	if err != nil {
		return CompiledAssertionsConfig{}, err
	}

	return CompiledAssertionsConfig{
		Enabled:                  assertions.Enabled,
		WiringPackages:           wiringPackages,
		AcceptConversionOnlyForm: assertions.AcceptConversionOnlyForm,
		ScanFunctionBodies:       assertions.ScanFunctionBodies,
		RequireAssertions:        assertions.RequireAssertions,
		RequireAssertionsStrict:  assertions.RequireAssertionsStrict,
		CheckBypassAssertions:    assertions.CheckBypassAssertions,
	}, nil
}

func compileExclude(exclude ExcludeConfig) (CompiledExcludeConfig, error) {
	files, err := compilePatterns(exclude.Files, "files")
	if err != nil {
		return CompiledExcludeConfig{}, err
	}

	types, err := compilePatterns(exclude.Types, "types")
	if err != nil {
		return CompiledExcludeConfig{}, err
	}

	return CompiledExcludeConfig{
		Files: files,
		Types: types,
	}, nil
}

// compilePatterns compiles a slice of regex patterns.
// Returns a descriptive error if any pattern fails to compile.
func compilePatterns(patterns []string, fieldName string) ([]*regexp.Regexp, error) {
	if len(patterns) == 0 {
		return nil, nil
	}

	result := make([]*regexp.Regexp, len(patterns))
	for i, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("%s[%d]: invalid regex %q: %w", fieldName, i, pattern, err)
		}
		result[i] = re
	}
	return result, nil
}
