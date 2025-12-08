package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/kausys/openapi/scanner"
	"github.com/kausys/openapi/spec"
	"gopkg.in/yaml.v3"
)

// GenerateMulti generates multiple OpenAPI specs based on spec: directives.
// Returns a map of spec name to OpenAPI spec.
func (g *Generator) GenerateMulti() (map[string]*spec.OpenAPI, error) {
	// Phase 1: Initialize cache
	if g.config.UseCache {
		if err := g.cache.Init(); err != nil {
			return nil, fmt.Errorf("failed to initialize cache: %w", err)
		}
		if err := g.cache.Load(); err != nil {
			return nil, fmt.Errorf("failed to load cache: %w", err)
		}
	}

	// Phase 2: Scan source files
	if err := g.scanner.Scan(); err != nil {
		return nil, fmt.Errorf("failed to scan source files: %w", err)
	}

	// Phase 3: Cache scanned data
	if g.config.UseCache {
		if err := g.cacheScannedData(); err != nil {
			return nil, fmt.Errorf("failed to cache scanned data: %w", err)
		}
	}

	// Phase 4: Assemble multiple OpenAPI specs
	specs, err := g.assembleMulti()
	if err != nil {
		return nil, fmt.Errorf("failed to assemble specs: %w", err)
	}

	// Phase 5: Write output files
	if g.config.OutputFile != "" {
		if err := g.writeMultiOutput(specs); err != nil {
			return nil, fmt.Errorf("failed to write output: %w", err)
		}
	}

	return specs, nil
}

// assembleMulti creates multiple OpenAPI specs from scanned data.
func (g *Generator) assembleMulti() (map[string]*spec.OpenAPI, error) {
	// Collect all spec names from routes
	specNames := g.collectSpecNames()

	// If no specs found, generate single default spec
	if len(specNames) == 0 {
		openAPI, err := g.assemble()
		if err != nil {
			return nil, err
		}
		return map[string]*spec.OpenAPI{scanner.DefaultSpec: openAPI}, nil
	}

	// Generate a spec for each name
	result := make(map[string]*spec.OpenAPI)
	for specName := range specNames {
		openAPI, err := g.assembleForSpec(specName)
		if err != nil {
			return nil, fmt.Errorf("failed to assemble spec %s: %w", specName, err)
		}
		// Only include specs that have routes
		if openAPI != nil && len(openAPI.Paths.PathItems) > 0 {
			result[specName] = openAPI
		}
	}

	return result, nil
}

// collectSpecNames collects all unique spec names from routes.
func (g *Generator) collectSpecNames() map[string]bool {
	specNames := make(map[string]bool)

	for _, route := range g.scanner.Routes {
		if len(route.Specs) == 0 {
			// Routes without spec: go to default
			specNames[scanner.DefaultSpec] = true
		} else {
			for _, s := range route.Specs {
				specNames[s] = true
			}
		}
	}

	return specNames
}

// assembleForSpec creates an OpenAPI spec for a specific spec name.
func (g *Generator) assembleForSpec(specName string) (*spec.OpenAPI, error) {
	// Reset referenced schemas for this spec
	g.referencedSchemas = make(map[string]bool)

	openAPI := &spec.OpenAPI{
		OpenAPI: "3.0.4",
		Paths: &spec.Paths{
			PathItems: make(map[string]*spec.PathItem),
		},
		Components: &spec.Components{
			Schemas:         make(map[string]*spec.Schema),
			SecuritySchemes: make(map[string]*spec.SecurityScheme),
		},
	}

	// Set info from meta (with inheritance)
	meta := g.getMetaForSpec(specName)
	generalMeta := g.scanner.Meta // Always get the general meta for inheritance

	if meta != nil {
		openAPI.Info = g.metaToInfo(meta)

		// Add security schemes from meta
		for name, scheme := range meta.SecuritySchemes {
			openAPI.Components.SecuritySchemes[name] = g.securitySchemeToSpec(scheme)
		}

		// Add tags from meta
		for _, tag := range meta.Tags {
			openAPI.Tags = append(openAPI.Tags, &spec.Tag{
				Name:        tag.Name,
				Description: tag.Description,
			})
		}
	} else if generalMeta != nil {
		// No specific meta, use general meta completely
		openAPI.Info = g.metaToInfo(generalMeta)

		for name, scheme := range generalMeta.SecuritySchemes {
			openAPI.Components.SecuritySchemes[name] = g.securitySchemeToSpec(scheme)
		}

		for _, tag := range generalMeta.Tags {
			openAPI.Tags = append(openAPI.Tags, &spec.Tag{
				Name:        tag.Name,
				Description: tag.Description,
			})
		}
	} else {
		openAPI.Info = &spec.Info{
			Title:   "API",
			Version: "1.0.0",
		}
	}

	// If specific meta doesn't have security schemes, inherit from general meta
	if len(openAPI.Components.SecuritySchemes) == 0 && generalMeta != nil {
		for name, scheme := range generalMeta.SecuritySchemes {
			openAPI.Components.SecuritySchemes[name] = g.securitySchemeToSpec(scheme)
		}
	}

	// Add routes that belong to this spec
	// This will mark schemas as referenced via markSchemaAsReferenced
	for _, routeInfo := range g.scanner.Routes {
		if g.routeBelongsToSpec(routeInfo, specName) {
			g.addRoute(openAPI, routeInfo)
		}
	}

	// In multi-spec mode, we need to carefully track which schemas are actually used.
	// The problem is that structToSchema marks references while converting, which
	// would mark ALL schemas if we convert them all upfront.
	//
	// Solution: Build schemas iteratively - only convert schemas that are referenced,
	// then check if those schemas reference more schemas, and repeat.
	g.buildReferencedSchemas(openAPI.Components, specName)

	return openAPI, nil
}

// buildReferencedSchemas iteratively builds only the schemas that are actually referenced.
// It starts with schemas marked as referenced by routes, then adds schemas referenced by those, and so on.
func (g *Generator) buildReferencedSchemas(components *spec.Components, specName string) {
	// Keep track of schemas we've already processed to avoid infinite loops
	processed := make(map[string]bool)

	// Keep iterating until no new schemas are added
	for {
		newSchemas := false

		// Find schemas that are referenced but not yet in components
		for schemaName := range g.referencedSchemas {
			if processed[schemaName] {
				continue
			}

			// Try to find this schema as a struct/model
			if structInfo, ok := g.scanner.Structs[schemaName]; ok && structInfo.IsModel {
				// Check if we should use a spec-specific version
				schema := g.getSchemaForSpec(schemaName, specName)
				if schema != nil {
					components.Schemas[schemaName] = schema
					newSchemas = true
				}
			}

			// Try to find as an enum
			if enumInfo, ok := g.scanner.Enums[schemaName]; ok {
				components.Schemas[schemaName] = g.enumToSchema(enumInfo)
				newSchemas = true
			}

			processed[schemaName] = true
		}

		if !newSchemas {
			break
		}
	}
}

// getMetaForSpec returns the meta for a specific spec, with inheritance from general meta.
func (g *Generator) getMetaForSpec(specName string) *scanner.MetaInfo {
	// First, look for a meta with this specific spec
	for _, meta := range g.scanner.Metas {
		if slices.Contains(meta.Specs, specName) {
			return meta
		}
	}

	// Fall back to general meta (meta without spec: directive)
	return g.scanner.Meta
}

// routeBelongsToSpec checks if a route belongs to a specific spec.
func (g *Generator) routeBelongsToSpec(route *scanner.RouteInfo, specName string) bool {
	// Routes without spec: go to default
	if len(route.Specs) == 0 {
		return specName == scanner.DefaultSpec
	}

	return slices.Contains(route.Specs, specName)
}

// getSchemaForSpec returns the appropriate schema for a spec, with override logic.
// If a model has spec: X, it overrides the general model for spec X.
func (g *Generator) getSchemaForSpec(modelName string, specName string) *spec.Schema {
	var generalModel *scanner.StructInfo
	var specificModel *scanner.StructInfo

	// Find models with this name
	for _, structInfo := range g.scanner.Structs {
		if structInfo.Name != modelName {
			continue
		}

		if len(structInfo.Specs) == 0 {
			// General model (no spec: directive)
			generalModel = structInfo
		} else if slices.Contains(structInfo.Specs, specName) {
			// Model is for our spec
			specificModel = structInfo
		}
	}

	// Priority: specific > general
	if specificModel != nil {
		return g.structToSchema(specificModel)
	}
	if generalModel != nil {
		return g.structToSchema(generalModel)
	}

	// Model not found for this spec
	return nil
}

// writeMultiOutput writes multiple specs to output files.
func (g *Generator) writeMultiOutput(specs map[string]*spec.OpenAPI) error {
	// Determine output directory and extension
	outputDir := filepath.Dir(g.config.OutputFile)
	ext := filepath.Ext(g.config.OutputFile)
	if ext == "" {
		ext = ".yaml"
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	for specName, openAPI := range specs {
		filename := filepath.Join(outputDir, specName+ext)

		var data []byte
		var err error

		switch g.config.OutputFormat {
		case "json":
			data, err = json.MarshalIndent(openAPI, "", "  ")
		default:
			data, err = yaml.Marshal(openAPI)
		}

		if err != nil {
			return fmt.Errorf("failed to marshal spec %s: %w", specName, err)
		}

		if err := os.WriteFile(filename, data, 0644); err != nil {
			return fmt.Errorf("failed to write spec %s: %w", specName, err)
		}
	}

	return nil
}

// GetSpecNames returns all spec names that would be generated.
func (g *Generator) GetSpecNames() ([]string, error) {
	// Scan if not already done
	if len(g.scanner.Routes) == 0 {
		if err := g.scanner.Scan(); err != nil {
			return nil, err
		}
	}

	specNames := g.collectSpecNames()
	names := make([]string, 0, len(specNames))
	for name := range specNames {
		names = append(names, name)
	}

	return names, nil
}

// GenerateSpec generates a single spec by name.
func (g *Generator) GenerateSpec(specName string) (*spec.OpenAPI, error) {
	// Phase 1: Initialize cache
	if g.config.UseCache {
		if err := g.cache.Init(); err != nil {
			return nil, fmt.Errorf("failed to initialize cache: %w", err)
		}
		if err := g.cache.Load(); err != nil {
			return nil, fmt.Errorf("failed to load cache: %w", err)
		}
	}

	// Phase 2: Scan source files
	if err := g.scanner.Scan(); err != nil {
		return nil, fmt.Errorf("failed to scan source files: %w", err)
	}

	// Phase 3: Cache scanned data
	if g.config.UseCache {
		if err := g.cacheScannedData(); err != nil {
			return nil, fmt.Errorf("failed to cache scanned data: %w", err)
		}
	}

	// Phase 4: Assemble specific spec
	openAPI, err := g.assembleForSpec(strings.ToLower(specName))
	if err != nil {
		return nil, fmt.Errorf("failed to assemble spec: %w", err)
	}

	// Phase 5: Write output
	if g.config.OutputFile != "" {
		if err := g.writeOutput(openAPI); err != nil {
			return nil, fmt.Errorf("failed to write output: %w", err)
		}
	}

	return openAPI, nil
}
