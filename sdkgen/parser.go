package sdkgen

import (
	"fmt"
	"os"

	"github.com/kausys/openapi/spec"
	"gopkg.in/yaml.v3"
)

// parseSpec reads and parses an OpenAPI YAML spec file.
func parseSpec(path string) (*spec.OpenAPI, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read spec file %s: %w", path, err)
	}

	var openAPI spec.OpenAPI
	if err := yaml.Unmarshal(data, &openAPI); err != nil {
		return nil, fmt.Errorf("failed to parse spec file %s: %w", path, err)
	}

	if openAPI.Paths == nil {
		openAPI.Paths = &spec.Paths{PathItems: make(map[string]*spec.PathItem)}
	}
	if openAPI.Components == nil {
		openAPI.Components = &spec.Components{Schemas: make(map[string]*spec.Schema)}
	}
	if openAPI.Components.Schemas == nil {
		openAPI.Components.Schemas = make(map[string]*spec.Schema)
	}

	return &openAPI, nil
}
