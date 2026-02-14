// Package config provides configuration types and parsing for the ifaceguard linter.
package config

import (
	"fmt"
	"sort"
	"strings"
)

// ParseFromAny decodes a settings object (typically from golangci-lint) into a Config.
// It starts with Default() and overlays any values present in the input.
// Keys are matched case-insensitively to account for Viper's lowercasing behavior.
func ParseFromAny(settings any) (Config, error) {
	cfg := Default()

	if settings == nil {
		return cfg, nil
	}

	m, ok := settings.(map[string]any)
	if !ok {
		return Config{}, fmt.Errorf("config: expected map[string]any, got %T", settings)
	}

	if err := validateUnknownKeys("settings", m, allowedTopLevelKeys()); err != nil {
		return Config{}, fmt.Errorf("config: %w", err)
	}

	if err := parseOwnershipSettings(&cfg.Ownership, m); err != nil {
		return Config{}, fmt.Errorf("config: ownership: %w", err)
	}

	if err := parseAssertionsSettings(&cfg.Assertions, m); err != nil {
		return Config{}, fmt.Errorf("config: assertions: %w", err)
	}

	if err := parseConstructorsSettings(&cfg.Constructors, m); err != nil {
		return Config{}, fmt.Errorf("config: constructors: %w", err)
	}

	if err := parseExcludeSettings(&cfg.Exclude, m); err != nil {
		return Config{}, fmt.Errorf("config: exclude: %w", err)
	}

	return cfg, nil
}

// parseOwnershipSettings parses the "ownership" section from a settings map.
func parseOwnershipSettings(ownership *OwnershipConfig, m map[string]any) error {
	section := getSection(m, "ownership")
	if section == nil {
		return nil
	}

	if err := validateUnknownKeys("ownership", section, allowedOwnershipKeys()); err != nil {
		return err
	}

	if v, ok := getBool(section, "enabled"); ok {
		ownership.Enabled = v
	}

	if v, ok := getString(section, "contractscope"); ok {
		scope := ContractScope(v)
		if err := validateContractScope(scope); err != nil {
			return err
		}
		ownership.ContractScope = &scope
	}

	if v, ok := getBool(section, "skipifusedasinput"); ok {
		ownership.SkipIfUsedAsInput = v
	}

	if v, ok := getStringSlice(section, "contractpackages"); ok {
		ownership.ContractPackages = v
	}

	if v, ok := getStringSlice(section, "ignoreinterfaces"); ok {
		ownership.IgnoreInterfaces = v
	}

	if v, ok := getBool(section, "ignoremarkerinterfaces"); ok {
		ownership.IgnoreMarkerInterfaces = v
	}

	return nil
}

// parseAssertionsSettings parses the "assertions" section from a settings map.
func parseAssertionsSettings(assertions *AssertionsConfig, m map[string]any) error {
	section := getSection(m, "assertions")
	if section == nil {
		return nil
	}

	if err := validateUnknownKeys("assertions", section, allowedAssertionsKeys()); err != nil {
		return err
	}

	if v, ok := getBool(section, "enabled"); ok {
		assertions.Enabled = v
	}

	if v, ok := getStringSlice(section, "wiringpackages"); ok {
		assertions.WiringPackages = v
	}

	if v, ok := getBool(section, "acceptconversiononlyform"); ok {
		assertions.AcceptConversionOnlyForm = v
	}

	if v, ok := getBool(section, "scanfunctionbodies"); ok {
		assertions.ScanFunctionBodies = v
	}

	if v, ok := getBool(section, "requireassertions"); ok {
		assertions.RequireAssertions = v
	}

	if v, ok := getBool(section, "requireassertionsstrict"); ok {
		assertions.RequireAssertionsStrict = v
	}

	if v, ok := getBool(section, "checkbypassassertions"); ok {
		assertions.CheckBypassAssertions = v
	}

	return nil
}

// parseExcludeSettings parses the "exclude" section from a settings map.
func parseExcludeSettings(exclude *ExcludeConfig, m map[string]any) error {
	section := getSection(m, "exclude")
	if section == nil {
		return nil
	}

	if err := validateUnknownKeys("exclude", section, allowedExcludeKeys()); err != nil {
		return err
	}

	if v, ok := getStringSlice(section, "files"); ok {
		exclude.Files = v
	}

	if v, ok := getStringSlice(section, "types"); ok {
		exclude.Types = v
	}

	return nil
}

// parseConstructorsSettings parses the "constructors" section from a settings map.
func parseConstructorsSettings(constructors *ConstructorsConfig, m map[string]any) error {
	section := getSection(m, "constructors")
	if section == nil {
		return nil
	}

	if err := validateUnknownKeys("constructors", section, allowedConstructorsKeys()); err != nil {
		return err
	}

	if v, ok := getBool(section, "enabled"); ok {
		constructors.Enabled = v
	}

	if v, ok := getStringSlice(section, "namepatterns"); ok {
		constructors.NamePatterns = v
	}

	if v, ok := getBool(section, "exportedonly"); ok {
		constructors.ExportedOnly = v
	}

	if v, ok := getStringSlice(section, "ignoreinterfaces"); ok {
		constructors.IgnoreInterfaces = v
	}

	if v, ok := getBool(section, "ignoreerrorreturn"); ok {
		constructors.IgnoreErrorReturn = v
	}

	return nil
}

// validateContractScope checks if the given scope is a valid enum value.
func validateContractScope(scope ContractScope) error {
	switch scope {
	case ContractScopeExportedOutput, ContractScopeAnyExported:
		return nil
	default:
		return fmt.Errorf(
			"invalid contractscope %q: allowed values are %s, %s",
			scope,
			ContractScopeExportedOutput,
			ContractScopeAnyExported,
		)
	}
}

func allowedTopLevelKeys() []string {
	return []string{
		"ownership",
		"assertions",
		"constructors",
		"exclude",
	}
}

func allowedOwnershipKeys() []string {
	return []string{
		"enabled",
		"contractscope",
		"skipifusedasinput",
		"contractpackages",
		"ignoreinterfaces",
		"ignoremarkerinterfaces",
	}
}

func allowedAssertionsKeys() []string {
	return []string{
		"enabled",
		"wiringpackages",
		"acceptconversiononlyform",
		"scanfunctionbodies",
		"requireassertions",
		"requireassertionsstrict",
		"checkbypassassertions",
	}
}

func allowedConstructorsKeys() []string {
	return []string{
		"enabled",
		"namepatterns",
		"exportedonly",
		"ignoreinterfaces",
		"ignoreerrorreturn",
	}
}

func allowedExcludeKeys() []string {
	return []string{
		"files",
		"types",
	}
}

func validateUnknownKeys(sectionName string, m map[string]any, allowed []string) error {
	if len(m) == 0 {
		return nil
	}

	allowedSet := make(map[string]struct{}, len(allowed))
	for _, key := range allowed {
		allowedSet[strings.ToLower(key)] = struct{}{}
	}

	unknown := make([]string, 0)
	for key := range m {
		if _, ok := allowedSet[strings.ToLower(key)]; !ok {
			unknown = append(unknown, key)
		}
	}

	if len(unknown) == 0 {
		return nil
	}

	sort.Strings(unknown)
	allowedSorted := make([]string, 0, len(allowed))
	for _, key := range allowed {
		allowedSorted = append(allowedSorted, strings.ToLower(key))
	}
	sort.Strings(allowedSorted)

	return fmt.Errorf(
		"unknown keys in %s: %s (allowed: %s)",
		sectionName,
		strings.Join(unknown, ", "),
		strings.Join(allowedSorted, ", "),
	)
}

// getSection returns a nested map for a given key (case-insensitive).
func getSection(m map[string]any, key string) map[string]any {
	for k, v := range m {
		if strings.EqualFold(k, key) {
			if section, ok := v.(map[string]any); ok {
				return section
			}
		}
	}
	return nil
}

// getBool returns a boolean value for a given key (case-insensitive).
func getBool(m map[string]any, key string) (bool, bool) {
	for k, v := range m {
		if strings.EqualFold(k, key) {
			if b, ok := v.(bool); ok {
				return b, true
			}
		}
	}
	return false, false
}

// getString returns a string value for a given key (case-insensitive).
func getString(m map[string]any, key string) (string, bool) {
	for k, v := range m {
		if strings.EqualFold(k, key) {
			if s, ok := v.(string); ok {
				return s, true
			}
		}
	}
	return "", false
}

// getStringSlice returns a string slice value for a given key (case-insensitive).
// It handles both []string and []any containing strings.
func getStringSlice(m map[string]any, key string) ([]string, bool) {
	for k, v := range m {
		if strings.EqualFold(k, key) {
			switch s := v.(type) {
			case []string:
				return s, true
			case []any:
				result := make([]string, 0, len(s))
				for _, item := range s {
					if str, ok := item.(string); ok {
						result = append(result, str)
					}
				}
				return result, len(result) == len(s)
			}
		}
	}
	return nil, false
}
