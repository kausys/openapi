package generator

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ConfigFile represents the .openapi.yaml configuration file.
type ConfigFile struct {
	CustomTypes map[string]TypeConfig `yaml:"custom_types"`
}

// TypeConfig represents a custom type configuration in the config file.
type TypeConfig struct {
	Type    string `yaml:"type"`
	Format  string `yaml:"format"`
	Example any    `yaml:"example"`
	Default any    `yaml:"default"`
}

// LoadConfigFile loads custom types from .openapi.yaml in the given directory.
// It searches for .openapi.yaml, .openapi.yml, or openapi.config.yaml.
func LoadConfigFile(dir string) error {
	configNames := []string{
		".openapi.yaml",
		".openapi.yml",
		"openapi.config.yaml",
		"openapi.config.yml",
	}

	var configPath string
	for _, name := range configNames {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			configPath = path
			break
		}
	}

	if configPath == "" {
		return nil // No config file, not an error
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var config ConfigFile
	if err := yaml.Unmarshal(data, &config); err != nil {
		return err
	}

	// Register custom types
	for typeName, typeConfig := range config.CustomTypes {
		RegisterTypeInfo(typeName, &TypeInfo{
			Type:    typeConfig.Type,
			Format:  typeConfig.Format,
			Example: typeConfig.Example,
			Default: typeConfig.Default,
		})
	}

	return nil
}
