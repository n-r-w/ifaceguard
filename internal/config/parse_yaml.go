package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ParseFromYAMLFile parses ifaceguard settings from a YAML file that matches the
// content of the README "settings" block (ownership/assertions/exclude).
func ParseFromYAMLFile(path string) (Config, error) {
	cleanPath := filepath.Clean(path)
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return Config{}, fmt.Errorf("config: read file: %w", err)
	}

	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return Config{}, fmt.Errorf("config: parse yaml: %w", err)
	}

	return ParseFromAny(raw)
}
