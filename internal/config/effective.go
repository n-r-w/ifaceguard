package config

// EffectiveOwnershipConfig contains ownership configuration with resolved defaults.
// ContractScope is no longer a pointer - it has been resolved to an actual value.
type EffectiveOwnershipConfig struct {
	Enabled                bool
	ContractScope          ContractScope
	SkipIfUsedAsInput      bool
	ContractPackages       []string
	IgnoreInterfaces       []string
	IgnoreMarkerInterfaces bool
}

// EffectiveConfig contains configuration with all runtime defaults resolved.
type EffectiveConfig struct {
	Ownership  EffectiveOwnershipConfig
	Assertions AssertionsConfig // Assertions has no runtime-dependent defaults
	Exclude    ExcludeConfig
}

// ResolveEffective resolves runtime-dependent defaults in the config.
// ContractScope resolution:
//   - If unset (nil): defaults to exportedoutput.
//   - If explicitly set: honored as-is.
func (c Config) ResolveEffective() EffectiveConfig {
	resolvedScope := resolveContractScope(c.Ownership.ContractScope)

	return EffectiveConfig{
		Ownership: EffectiveOwnershipConfig{
			Enabled:                c.Ownership.Enabled,
			ContractScope:          resolvedScope,
			SkipIfUsedAsInput:      c.Ownership.SkipIfUsedAsInput,
			ContractPackages:       c.Ownership.ContractPackages,
			IgnoreInterfaces:       c.Ownership.IgnoreInterfaces,
			IgnoreMarkerInterfaces: c.Ownership.IgnoreMarkerInterfaces,
		},
		Assertions: c.Assertions,
		Exclude:    c.Exclude,
	}
}

// resolveContractScope resolves the ContractScope default.
func resolveContractScope(scope *ContractScope) ContractScope {
	if scope != nil {
		return *scope
	}
	return ContractScopeExportedOutput
}

// Compile compiles all regex patterns in the effective config.
// Unlike Config.Compile(), this returns a CompiledConfig with a non-nil ContractScope pointer
// pointing to the resolved scope value.
func (ec EffectiveConfig) Compile() (CompiledConfig, error) {
	compiledOwnership, err := compileEffectiveOwnership(ec.Ownership)
	if err != nil {
		return CompiledConfig{}, err
	}

	compiledAssertions, err := compileAssertions(ec.Assertions)
	if err != nil {
		return CompiledConfig{}, err
	}

	compiledExclude, err := compileExclude(ec.Exclude)
	if err != nil {
		return CompiledConfig{}, err
	}

	return CompiledConfig{
		Ownership:  compiledOwnership,
		Assertions: compiledAssertions,
		Exclude:    compiledExclude,
	}, nil
}

func compileEffectiveOwnership(ownership EffectiveOwnershipConfig) (CompiledOwnershipConfig, error) {
	contractPackages, err := compilePatterns(ownership.ContractPackages, "contractpackages")
	if err != nil {
		return CompiledOwnershipConfig{}, err
	}

	ignoreInterfaces, err := compilePatterns(ownership.IgnoreInterfaces, "ignoreinterfaces")
	if err != nil {
		return CompiledOwnershipConfig{}, err
	}

	scope := ownership.ContractScope // copy the resolved value

	return CompiledOwnershipConfig{
		Enabled:                ownership.Enabled,
		ContractScope:          &scope,
		SkipIfUsedAsInput:      ownership.SkipIfUsedAsInput,
		ContractPackages:       contractPackages,
		IgnoreInterfaces:       ignoreInterfaces,
		IgnoreMarkerInterfaces: ownership.IgnoreMarkerInterfaces,
	}, nil
}
