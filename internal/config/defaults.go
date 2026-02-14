package config

// Default returns a Config with all default values as specified in the ifaceguard spec.
// Note: Ownership.ContractScope is nil (unset) by default and resolves to exportedoutput.
func Default() Config {
	return Config{
		Ownership: OwnershipConfig{
			Enabled:                true,
			ContractScope:          nil, // Unset: resolved to exportedoutput
			SkipIfUsedAsInput:      true,
			ContractPackages:       nil,
			IgnoreInterfaces:       nil,
			IgnoreMarkerInterfaces: true,
		},
		Assertions: AssertionsConfig{
			Enabled:                  true,
			WiringPackages:           nil,
			AcceptConversionOnlyForm: false,
			ScanFunctionBodies:       false,
			RequireAssertions:        false,
			RequireAssertionsStrict:  false,
			CheckBypassAssertions:    true,
		},
		Constructors: ConstructorsConfig{
			Enabled:           false,
			NamePatterns:      []string{"^New[A-Z]", "^MustNew[A-Z]"},
			ExportedOnly:      true,
			IgnoreInterfaces:  nil,
			IgnoreErrorReturn: true,
		},
		Exclude: ExcludeConfig{
			Files: nil,
			Types: nil,
		},
	}
}
