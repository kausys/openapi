package generator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kausys/openapi/cache"
	"github.com/kausys/openapi/scanner"
	"github.com/kausys/openapi/spec"
	"gopkg.in/yaml.v3"
)

// Generator orchestrates the OpenAPI spec generation process.
type Generator struct {
	config  *Config
	cache   *cache.Manager
	scanner *scanner.Scanner

	// referencedSchemas tracks which schemas are actually used in the spec
	referencedSchemas map[string]bool
}

// New creates a new Generator with the given options.
func New(opts ...Option) *Generator {
	cfg := DefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	scannerOpts := []scanner.Option{
		scanner.WithDir(cfg.Dir),
		scanner.WithPattern(cfg.Pattern),
		scanner.WithIgnorePaths(cfg.IgnorePaths...),
	}

	return &Generator{
		config:            cfg,
		cache:             cache.NewManager(cfg.Dir),
		scanner:           scanner.New(scannerOpts...),
		referencedSchemas: make(map[string]bool),
	}
}

// Generate runs the full generation pipeline.
func (g *Generator) Generate() (*spec.OpenAPI, error) {
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

	// Phase 4: Assemble OpenAPI spec
	openAPI, err := g.assemble()
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

// cacheScannedData saves scanned data to cache.
func (g *Generator) cacheScannedData() error {
	// Group schemas and routes by source file (using relative paths)
	fileSchemas := make(map[string][]string)
	fileRoutes := make(map[string][]string)

	// Cache schemas and track by source file
	for name, structInfo := range g.scanner.Structs {
		schema := g.structToSchema(structInfo)
		if err := g.cache.SaveSchema(name, schema); err != nil {
			return err
		}

		// Track which file this schema came from (convert to relative path)
		if sourceFile, ok := g.scanner.StructSources[name]; ok {
			relPath := g.toRelativePath(sourceFile)
			fileSchemas[relPath] = append(fileSchemas[relPath], name)
		}
	}

	// Cache routes and track by source file
	for opID, routeInfo := range g.scanner.Routes {
		route := g.routeToOperation(routeInfo)
		if err := g.cache.SaveRoute(opID, route); err != nil {
			return err
		}

		// Track which file this route came from (convert to relative path)
		if sourceFile, ok := g.scanner.RouteSources[opID]; ok {
			relPath := g.toRelativePath(sourceFile)
			fileRoutes[relPath] = append(fileRoutes[relPath], opID)
		}
	}

	// Update cache index with file entries
	allFiles := make(map[string]bool)
	for f := range fileSchemas {
		allFiles[f] = true
	}
	for f := range fileRoutes {
		allFiles[f] = true
	}

	for filePath := range allFiles {
		schemas := fileSchemas[filePath]
		routes := fileRoutes[filePath]
		if err := g.cache.UpdateEntry(filePath, schemas, routes, nil); err != nil {
			return err
		}
	}

	// Save cache index
	return g.cache.Save()
}

// toRelativePath converts an absolute path to a path relative to the config directory.
func (g *Generator) toRelativePath(absPath string) string {
	relPath, err := filepath.Rel(g.config.Dir, absPath)
	if err != nil {
		return absPath // fallback to absolute if conversion fails
	}
	return relPath
}

// assemble creates the OpenAPI spec from scanned data.
func (g *Generator) assemble() (*spec.OpenAPI, error) {
	// Reset referenced schemas for each generation
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

	// Set info from meta
	if g.scanner.Meta != nil {
		openAPI.Info = g.metaToInfo(g.scanner.Meta)

		// Add security schemes from meta
		for name, scheme := range g.scanner.Meta.SecuritySchemes {
			openAPI.Components.SecuritySchemes[name] = g.securitySchemeToSpec(scheme)
		}

		// Add tags from meta
		for _, tag := range g.scanner.Meta.Tags {
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

	// Add schemas (all models first)
	for name, structInfo := range g.scanner.Structs {
		if structInfo.IsModel {
			openAPI.Components.Schemas[name] = g.structToSchema(structInfo)
		}
	}

	// Add enums as schemas
	for name, enumInfo := range g.scanner.Enums {
		openAPI.Components.Schemas[name] = g.enumToSchema(enumInfo)
	}

	// Add paths (this will mark schemas as referenced)
	for _, routeInfo := range g.scanner.Routes {
		g.addRoute(openAPI, routeInfo)
	}

	// Clean unused schemas if enabled
	if g.config.CleanUnused {
		// First, recursively mark schemas referenced by other referenced schemas
		g.markNestedReferences(openAPI.Components)
		// Then clean unused schemas
		g.cleanUnusedSchemas(openAPI.Components)
	}

	return openAPI, nil
}

// markSchemaAsReferenced marks a schema as being used.
func (g *Generator) markSchemaAsReferenced(schemaName string) {
	if schemaName != "" {
		g.referencedSchemas[schemaName] = true
	}
}

// markNestedReferences recursively marks schemas that are referenced by already-referenced schemas.
func (g *Generator) markNestedReferences(components *spec.Components) {
	if components == nil || components.Schemas == nil {
		return
	}

	// Keep marking until no new references are found
	for {
		newRefs := false
		for schemaName := range g.referencedSchemas {
			schema, exists := components.Schemas[schemaName]
			if !exists || schema == nil {
				continue
			}

			// Check properties for references
			for _, propSchema := range schema.Properties {
				if ref := g.extractSchemaRef(propSchema); ref != "" {
					if !g.referencedSchemas[ref] {
						g.referencedSchemas[ref] = true
						newRefs = true
					}
				}
			}

			// Check array items
			if schema.Items != nil {
				if ref := g.extractSchemaRef(schema.Items); ref != "" {
					if !g.referencedSchemas[ref] {
						g.referencedSchemas[ref] = true
						newRefs = true
					}
				}
			}

			// Check additionalProperties
			if schema.AdditionalProperties != nil {
				if ref := g.extractSchemaRef(schema.AdditionalProperties); ref != "" {
					if !g.referencedSchemas[ref] {
						g.referencedSchemas[ref] = true
						newRefs = true
					}
				}
			}

			// Check composition schemas
			for _, s := range schema.AllOf {
				if ref := g.extractSchemaRef(s); ref != "" {
					if !g.referencedSchemas[ref] {
						g.referencedSchemas[ref] = true
						newRefs = true
					}
				}
			}
			for _, s := range schema.OneOf {
				if ref := g.extractSchemaRef(s); ref != "" {
					if !g.referencedSchemas[ref] {
						g.referencedSchemas[ref] = true
						newRefs = true
					}
				}
			}
			for _, s := range schema.AnyOf {
				if ref := g.extractSchemaRef(s); ref != "" {
					if !g.referencedSchemas[ref] {
						g.referencedSchemas[ref] = true
						newRefs = true
					}
				}
			}
		}

		if !newRefs {
			break
		}
	}
}

// extractSchemaRef extracts the schema name from a $ref if present.
func (g *Generator) extractSchemaRef(schema *spec.Schema) string {
	if schema == nil || schema.Ref == "" {
		return ""
	}
	// Extract schema name from "#/components/schemas/SchemaName"
	const prefix = "#/components/schemas/"
	if len(schema.Ref) > len(prefix) {
		return schema.Ref[len(prefix):]
	}
	return ""
}

// cleanUnusedSchemas removes schemas that are declared but not referenced.
func (g *Generator) cleanUnusedSchemas(components *spec.Components) {
	if components == nil || components.Schemas == nil {
		return
	}

	// Remove unreferenced schemas
	for schemaName := range components.Schemas {
		if !g.referencedSchemas[schemaName] {
			delete(components.Schemas, schemaName)
		}
	}
}

// writeOutput writes the spec to the output file.
func (g *Generator) writeOutput(openAPI *spec.OpenAPI) error {
	dir := filepath.Dir(g.config.OutputFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	var data []byte
	var err error

	switch g.config.OutputFormat {
	case "json":
		data, err = json.MarshalIndent(openAPI, "", "  ")
	default:
		data, err = yaml.Marshal(openAPI)
	}

	if err != nil {
		return err
	}

	return os.WriteFile(g.config.OutputFile, data, 0644)
}
