// Package config provides configuration types and defaults for the ifaceguard linter.
package config

// ContractScope defines the mode for classifying interfaces as contractual in the ownership rule.
type ContractScope string

const (
	// ContractScopeExportedOutput classifies an interface as contractual if it appears in exported
	// function/method results or exported variable types (package-local heuristic).
	ContractScopeExportedOutput ContractScope = "exportedoutput"

	// ContractScopeAnyExported classifies any exported interface as contractual (strictest mode).
	ContractScopeAnyExported ContractScope = "anyexported"
)

// OwnershipConfig contains configuration for the ownership rule.
type OwnershipConfig struct {
	// Enabled controls whether the ownership rule is active.
	Enabled bool

	// ContractScope determines how interfaces are classified as contractual.
	// A nil value means "unset" and defaults to exportedoutput.
	ContractScope *ContractScope

	// SkipIfUsedAsInput suppresses contractual classification for interfaces that appear
	// only in exported function/method input parameters (not in results or vars).
	// Applies only when ContractScope is exportedoutput.
	SkipIfUsedAsInput bool

	// ContractPackages is a list of regex patterns matching package import paths
	// that are allowed to declare interfaces implemented in the same package.
	ContractPackages []string

	// IgnoreInterfaces is a list of regex patterns matching "pkgpath.InterfaceName"
	// that should be excluded from ownership checks.
	IgnoreInterfaces []string

	// IgnoreMarkerInterfaces excludes marker interfaces (no methods, no embedded interfaces,
	// and IsMethodSet==true) from ownership checks.
	IgnoreMarkerInterfaces bool
}

// AssertionsConfig contains configuration for the assertions rule.
type AssertionsConfig struct {
	// Enabled controls whether the assertions rule is active.
	Enabled bool

	// WiringPackages is a list of regex patterns matching package import paths
	// where assertions are allowed regardless of implementation type location.
	WiringPackages []string

	// AcceptConversionOnlyForm enables recognition of assertions without explicit
	// interface type in ValueSpec.Type, e.g., var _ = consumer.I((*T)(nil)).
	AcceptConversionOnlyForm bool

	// ScanFunctionBodies enables scanning var assertions inside function bodies,
	// not just package-level declarations.
	ScanFunctionBodies bool

	// RequireAssertions enables checking that each type implementing a contractual
	// interface from another package has at least one assertion in an allowed location.
	RequireAssertions bool

	// RequireAssertionsStrict disables relevance filtering for requireassertions.
	// When true, any matching contractual interface in the module can trigger IFG003.
	RequireAssertionsStrict bool
}

// ExcludeConfig contains global exclusion rules applied to all checks.
type ExcludeConfig struct {
	// Files is a list of regex patterns matching file paths
	// that should be excluded from analysis.
	Files []string

	// Types is a list of regex patterns matching "pkgpath.TypeName"
	// that should be excluded from analysis.
	Types []string
}

// Config contains all configuration for the ifaceguard linter.
type Config struct {
	Ownership  OwnershipConfig
	Assertions AssertionsConfig
	Exclude    ExcludeConfig
}
