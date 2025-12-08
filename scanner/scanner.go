package scanner

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"
)

// Config holds configuration options for the scanner.
type Config struct {
	// Pattern is the package pattern to scan (e.g., "./...", "./api/...")
	Pattern string
	// Dir is the directory path to scan from
	Dir string
	// IgnorePaths contains path patterns to exclude during scanning
	IgnorePaths []string
}

// Option is a function type for configuring the Scanner.
type Option func(*Config)

// WithPattern sets the package pattern to scan.
func WithPattern(pattern string) Option {
	return func(c *Config) {
		c.Pattern = pattern
	}
}

// WithDir sets the directory path to scan from.
func WithDir(dir string) Option {
	return func(c *Config) {
		c.Dir = dir
	}
}

// WithIgnorePaths sets the path patterns to ignore during scanning.
func WithIgnorePaths(paths ...string) Option {
	return func(c *Config) {
		c.IgnorePaths = append(c.IgnorePaths, paths...)
	}
}

// Scanner scans Go source code for OpenAPI directives.
type Scanner struct {
	config *Config
	fset   *token.FileSet

	// Extracted data
	Meta    *MetaInfo   // General meta (first meta without spec: directive)
	Metas   []*MetaInfo // All metas including spec-specific ones
	Enums   map[string]*EnumInfo
	Structs map[string]*StructInfo
	Routes  map[string]*RouteInfo

	// Type mappings
	TypeToEnum   map[string]string // Go type name -> enum name
	TypeToStruct map[string]string // Go type name -> struct name

	// Source file mappings
	EnumSources   map[string]string // enum name -> source file
	StructSources map[string]string // struct name -> source file
	RouteSources  map[string]string // operation ID -> source file

	// Type info for resolving embedded types
	typeInfo map[string]types.Object // Fully qualified type name -> types.Object
	pkgInfo  map[*ast.File]*packages.Package
}

// New creates a new Scanner with the given options.
func New(options ...Option) *Scanner {
	config := &Config{
		Pattern:     "./...",
		Dir:         ".",
		IgnorePaths: []string{},
	}

	for _, opt := range options {
		opt(config)
	}

	return &Scanner{
		config:        config,
		fset:          token.NewFileSet(),
		Meta:          nil, // Will be set to first general meta
		Metas:         []*MetaInfo{},
		Enums:         make(map[string]*EnumInfo),
		Structs:       make(map[string]*StructInfo),
		Routes:        make(map[string]*RouteInfo),
		TypeToEnum:    make(map[string]string),
		TypeToStruct:  make(map[string]string),
		EnumSources:   make(map[string]string),
		StructSources: make(map[string]string),
		RouteSources:  make(map[string]string),
		typeInfo:      make(map[string]types.Object),
		pkgInfo:       make(map[*ast.File]*packages.Package),
	}
}

// Scan scans all packages matching the configured pattern.
func (s *Scanner) Scan() error {
	cfg := &packages.Config{
		Mode:  packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo,
		Dir:   s.config.Dir,
		Fset:  s.fset,
		Tests: false,
	}

	pkgs, err := packages.Load(cfg, s.config.Pattern)
	if err != nil {
		return err
	}

	// First pass: collect type information (skip packages with errors)
	for _, pkg := range pkgs {
		if packages.PrintErrors([]*packages.Package{pkg}) > 0 {
			continue
		}
		s.collectTypeInfo(pkg)
	}

	// Second pass: process files
	for _, pkg := range pkgs {
		hasErrors := len(pkg.Errors) > 0

		if shouldIgnorePath(pkg.PkgPath, s.config.IgnorePaths) {
			continue
		}

		for i, file := range pkg.Syntax {
			if i >= len(pkg.GoFiles) {
				continue
			}
			filePath := pkg.GoFiles[i]

			if shouldIgnorePath(filePath, s.config.IgnorePaths) {
				continue
			}

			s.pkgInfo[file] = pkg

			// For packages with errors, only process meta (comments are still available)
			if hasErrors {
				// Still try to extract meta from packages with errors
				if err := s.processMeta(filePath, file); err != nil {
					// Ignore errors from packages with compilation issues
					continue
				}
				continue
			}

			if err := s.processFile(filePath, file, pkg); err != nil {
				return err
			}
		}
	}

	// Third pass: resolve embedded types
	s.resolveEmbeddedTypes()

	return nil
}

// collectTypeInfo collects type information from a package.
func (s *Scanner) collectTypeInfo(pkg *packages.Package) {
	if pkg.TypesInfo == nil {
		return
	}
	scope := pkg.Types.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if obj == nil {
			continue
		}
		// Store with package path prefix
		fullName := pkg.PkgPath + "." + name
		s.typeInfo[fullName] = obj
		// Also store short name for local package lookups
		s.typeInfo[name] = obj
	}
}

// ScanFile scans a single Go source file.
func (s *Scanner) ScanFile(filePath string) error {
	cfg := &packages.Config{
		Mode:  packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo,
		Dir:   s.config.Dir,
		Fset:  s.fset,
		Tests: false,
	}

	pkgs, err := packages.Load(cfg, "file="+filePath)
	if err != nil {
		return err
	}

	for _, pkg := range pkgs {
		s.collectTypeInfo(pkg)
		for i, file := range pkg.Syntax {
			if i >= len(pkg.GoFiles) {
				continue
			}
			if pkg.GoFiles[i] == filePath {
				s.pkgInfo[file] = pkg
				return s.processFile(filePath, file, pkg)
			}
		}
	}

	return nil
}

// processFile processes a single AST file.
func (s *Scanner) processFile(filePath string, file *ast.File, _ *packages.Package) error {
	// Process meta information
	if err := s.processMeta(filePath, file); err != nil {
		return err
	}

	// Process enums
	if err := s.processEnums(filePath, file); err != nil {
		return err
	}

	// Process schemas (models and parameters)
	if err := s.processSchemas(filePath, file); err != nil {
		return err
	}

	// Process routes
	if err := s.processRoutes(filePath, file); err != nil {
		return err
	}

	return nil
}

// resolveEmbeddedTypes resolves embedded types in structs by expanding their fields.
func (s *Scanner) resolveEmbeddedTypes() {
	resolved := make(map[string]bool)
	for _, structInfo := range s.Structs {
		s.resolveEmbeddedTypesRecursive(structInfo, resolved)
	}
}

// resolveEmbeddedTypesRecursive resolves embedded types recursively to handle nested embeds.
func (s *Scanner) resolveEmbeddedTypesRecursive(structInfo *StructInfo, resolved map[string]bool) {
	if resolved[structInfo.Name] {
		return
	}
	resolved[structInfo.Name] = true

	if len(structInfo.EmbeddedTypes) == 0 {
		return
	}

	for _, embeddedType := range structInfo.EmbeddedTypes {
		s.resolveEmbeddedType(structInfo, embeddedType, resolved)
	}
}

// resolveEmbeddedType resolves a single embedded type and adds its fields to the struct.
func (s *Scanner) resolveEmbeddedType(structInfo *StructInfo, embeddedTypeName string, resolved map[string]bool) {
	// First, check if it's a known struct in our scanner
	if embedded, ok := s.Structs[embeddedTypeName]; ok {
		// Recursively resolve the embedded struct first
		s.resolveEmbeddedTypesRecursive(embedded, resolved)
		structInfo.Fields = append(structInfo.Fields, embedded.Fields...)
		return
	}

	// Check by short name (without package prefix)
	shortName := embeddedTypeName
	if idx := lastIndex(embeddedTypeName, "."); idx >= 0 {
		shortName = embeddedTypeName[idx+1:]
	}

	if embedded, ok := s.Structs[shortName]; ok {
		// Recursively resolve the embedded struct first
		s.resolveEmbeddedTypesRecursive(embedded, resolved)
		structInfo.Fields = append(structInfo.Fields, embedded.Fields...)
		return
	}

	// Try to resolve using type information
	obj := s.typeInfo[embeddedTypeName]
	if obj == nil {
		obj = s.typeInfo[shortName]
	}
	if obj == nil {
		return
	}

	// Get the underlying type
	typeName, ok := obj.(*types.TypeName)
	if !ok {
		return
	}

	underlying := typeName.Type().Underlying()
	structType, ok := underlying.(*types.Struct)
	if !ok {
		return
	}

	// Extract fields from the types.Struct
	fields := s.extractFieldsFromTypesStruct(structType)
	structInfo.Fields = append(structInfo.Fields, fields...)
}

// extractFieldsFromTypesStruct extracts FieldInfo from a types.Struct.
func (s *Scanner) extractFieldsFromTypesStruct(st *types.Struct) []*FieldInfo {
	var fields []*FieldInfo

	for i := 0; i < st.NumFields(); i++ {
		field := st.Field(i)
		tag := st.Tag(i)

		// Skip unexported fields
		if !field.Exported() {
			continue
		}

		// Handle embedded fields recursively
		if field.Embedded() {
			if named, ok := field.Type().(*types.Named); ok {
				if underlying, ok := named.Underlying().(*types.Struct); ok {
					embeddedFields := s.extractFieldsFromTypesStruct(underlying)
					fields = append(fields, embeddedFields...)
				}
			}
			continue
		}

		fieldInfo := &FieldInfo{
			Name:        field.Name(),
			Tags:        make(map[string]string),
			Validations: make(map[string]string),
		}

		// Determine type
		s.setFieldTypeFromTypesType(fieldInfo, field.Type())

		// Parse struct tags
		if tag != "" {
			parseFieldTags(fieldInfo, "`"+tag+"`")
		}

		fields = append(fields, fieldInfo)
	}

	return fields
}

// setFieldTypeFromTypesType sets field type information from types.Type.
func (s *Scanner) setFieldTypeFromTypesType(fieldInfo *FieldInfo, t types.Type) {
	switch typ := t.(type) {
	case *types.Basic:
		fieldInfo.Type = typ.Name()
	case *types.Named:
		fieldInfo.Type = typ.Obj().Name()
	case *types.Pointer:
		fieldInfo.IsPointer = true
		s.setFieldTypeFromTypesType(fieldInfo, typ.Elem())
	case *types.Slice:
		fieldInfo.IsArray = true
		s.setFieldTypeFromTypesType(fieldInfo, typ.Elem())
	case *types.Map:
		fieldInfo.IsMap = true
		if basic, ok := typ.Key().(*types.Basic); ok {
			fieldInfo.MapKeyType = basic.Name()
		}
		s.setFieldTypeFromTypesType(fieldInfo, typ.Elem())
	}
}

// lastIndex returns the index of the last occurrence of sep in s, or -1 if not found.
func lastIndex(s, sep string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == sep[0] {
			return i
		}
	}
	return -1
}
