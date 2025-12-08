package scanner

import (
	"go/ast"
	"go/token"
	"strings"
)

// processSchemas processes swagger:model and swagger:parameters directives.
func (s *Scanner) processSchemas(filePath string, file *ast.File) error {
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			if genDecl.Doc == nil {
				continue
			}

			var name string
			var isParameter, isModel bool

			if hasDirective(genDecl.Doc, ModelDirective) {
				name = extractDirectiveValue(genDecl.Doc, ModelDirective)
				isModel = true
			}

			if hasDirective(genDecl.Doc, ParameterDirective) {
				name = extractDirectiveValue(genDecl.Doc, ParameterDirective)
				isParameter = true
			}

			if !isModel && !isParameter {
				continue
			}

			if name == "" {
				name = typeSpec.Name.Name
			}

			structInfo := &StructInfo{
				Name:        name,
				Fields:      []*FieldInfo{},
				Description: extractDescription(genDecl.Doc, []string{SwaggerPrefix, OneOfDirective, AllOfDirective, AnyOfDirective, SpecDirective}),
				IsParameter: isParameter,
				IsModel:     isModel,
				SourceFile:  filePath,
				OneOf:       extractCompositionSchemas(genDecl.Doc, OneOfDirective),
				AllOf:       extractCompositionSchemas(genDecl.Doc, AllOfDirective),
				AnyOf:       extractCompositionSchemas(genDecl.Doc, AnyOfDirective),
				Specs:       extractSpecs(genDecl.Doc),
			}

			// Process struct fields
			if structType, ok := typeSpec.Type.(*ast.StructType); ok {
				processStructFields(structInfo, structType)
			}

			s.Structs[name] = structInfo
			s.TypeToStruct[typeSpec.Name.Name] = name
			s.StructSources[name] = filePath
		}
	}
	return nil
}

// extractCompositionSchemas extracts schema names from composition directives.
func extractCompositionSchemas(doc *ast.CommentGroup, directive string) []string {
	if doc == nil {
		return nil
	}

	value := extractDirectiveValue(doc, directive)
	if value == "" {
		return nil
	}

	var schemas []string
	for _, s := range strings.Split(value, ",") {
		s = strings.TrimSpace(s)
		if s != "" {
			schemas = append(schemas, s)
		}
	}
	return schemas
}

// processStructFields processes all fields in a struct.
func processStructFields(structInfo *StructInfo, structType *ast.StructType) {
	if structType.Fields == nil {
		return
	}

	for _, field := range structType.Fields.List {
		// Check if this is an embedded field (no names)
		if len(field.Names) == 0 {
			embeddedType := extractTypeName(field.Type)
			if embeddedType != "" {
				structInfo.EmbeddedTypes = append(structInfo.EmbeddedTypes, embeddedType)
			}
			continue
		}

		fieldInfo := parseField(field)
		if fieldInfo != nil {
			structInfo.Fields = append(structInfo.Fields, fieldInfo)
		}
	}
}

// extractTypeName extracts the type name from an AST expression.
func extractTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return extractTypeName(t.X)
	case *ast.SelectorExpr:
		if x, ok := t.X.(*ast.Ident); ok {
			return x.Name + "." + t.Sel.Name
		}
	}
	return ""
}

// parseField parses a single struct field.
func parseField(field *ast.Field) *FieldInfo {
	// Skip fields without names (embedded)
	if len(field.Names) == 0 {
		return nil
	}

	// Skip unexported fields
	if !ast.IsExported(field.Names[0].Name) {
		return nil
	}

	// Check if field has swagger:ignore directive BEFORE creating FieldInfo
	if hasFieldIgnoreDirective(field) {
		return nil
	}

	fieldInfo := &FieldInfo{
		Name:        field.Names[0].Name,
		Tags:        make(map[string]string),
		Validations: make(map[string]string),
	}

	// Determine field type
	determineFieldType(fieldInfo, field.Type)

	// Parse struct tags
	if field.Tag != nil {
		parseFieldTags(fieldInfo, field.Tag.Value)
	}

	// Parse field doc comments
	if field.Doc != nil {
		parseFieldDoc(fieldInfo, field.Doc)
	}

	// Also check inline comments
	if field.Comment != nil {
		parseFieldDoc(fieldInfo, field.Comment)
	}

	return fieldInfo
}

// hasFieldIgnoreDirective checks if a field has the swagger:ignore directive.
func hasFieldIgnoreDirective(field *ast.Field) bool {
	// Check field doc comments
	if field.Doc != nil && hasDirective(field.Doc, IgnoreDirective) {
		return true
	}
	// Check inline comments
	if field.Comment != nil && hasDirective(field.Comment, IgnoreDirective) {
		return true
	}
	return false
}

// determineFieldType determines the type information for a field.
func determineFieldType(fieldInfo *FieldInfo, expr ast.Expr) {
	switch t := expr.(type) {
	case *ast.Ident:
		fieldInfo.Type = t.Name
	case *ast.StarExpr:
		fieldInfo.IsPointer = true
		determineFieldType(fieldInfo, t.X)
	case *ast.ArrayType:
		fieldInfo.IsArray = true
		determineFieldType(fieldInfo, t.Elt)
	case *ast.MapType:
		fieldInfo.IsMap = true
		if keyIdent, ok := t.Key.(*ast.Ident); ok {
			fieldInfo.MapKeyType = keyIdent.Name
		}
		determineFieldType(fieldInfo, t.Value)
	case *ast.SelectorExpr:
		if x, ok := t.X.(*ast.Ident); ok {
			fieldInfo.Type = x.Name + "." + t.Sel.Name
		}
	case *ast.StructType:
		// Inline struct type
		fieldInfo.Type = "object"
		fieldInfo.IsInlineStruct = true
		fieldInfo.InlineStruct = processInlineStruct(fieldInfo.Name, t)
	}
}

// processInlineStruct processes an inline struct and returns a StructInfo.
func processInlineStruct(name string, structType *ast.StructType) *StructInfo {
	structInfo := &StructInfo{
		Name:   name,
		Fields: []*FieldInfo{},
	}

	for _, field := range structType.Fields.List {
		// Skip embedded fields
		if len(field.Names) == 0 {
			continue
		}

		fieldInfo := parseField(field)
		if fieldInfo != nil {
			structInfo.Fields = append(structInfo.Fields, fieldInfo)
		}
	}

	return structInfo
}

// parseFieldTags parses struct tags (json, yaml, validate, etc.).
func parseFieldTags(fieldInfo *FieldInfo, tagValue string) {
	// Remove backticks
	tagValue = strings.Trim(tagValue, "`")

	// Parse json tag
	if jsonTag := getTagValue(tagValue, "json"); jsonTag != "" {
		parts := strings.Split(jsonTag, ",")
		if len(parts) > 0 && parts[0] != "-" {
			fieldInfo.Tags["json"] = parts[0]
		}
		for _, opt := range parts[1:] {
			if opt == "omitempty" || opt == "omitzero" {
				fieldInfo.HasOmitempty = true
			}
		}
	}

	// Parse yaml tag
	if yamlTag := getTagValue(tagValue, "yaml"); yamlTag != "" {
		parts := strings.Split(yamlTag, ",")
		if len(parts) > 0 && parts[0] != "-" {
			fieldInfo.Tags["yaml"] = parts[0]
		}
	}

	// Parse example tag
	if example := getTagValue(tagValue, "example"); example != "" {
		fieldInfo.Example = example
	}

	// Parse default tag
	if def := getTagValue(tagValue, "default"); def != "" {
		fieldInfo.Default = def
	}

	// Parse validate tag for validations
	if validate := getTagValue(tagValue, "validate"); validate != "" {
		parseValidateTag(fieldInfo, validate)
	}
}

// getTagValue extracts the value for a specific tag key.
func getTagValue(tagValue, key string) string {
	// Look for key:"value" pattern
	keyPattern := key + `:"([^"]*)"`
	re := getCompiledRegex(keyPattern)
	matches := re.FindStringSubmatch(tagValue)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// parseValidateTag parses validation rules from the validate tag.
func parseValidateTag(fieldInfo *FieldInfo, validate string) {
	rules := strings.Split(validate, ",")
	for _, rule := range rules {
		rule = strings.TrimSpace(rule)
		if rule == "required" {
			fieldInfo.Required = true
			fieldInfo.ExplicitRequired = true
		} else if strings.HasPrefix(rule, "min=") {
			fieldInfo.Validations["min"] = strings.TrimPrefix(rule, "min=")
		} else if strings.HasPrefix(rule, "max=") {
			fieldInfo.Validations["max"] = strings.TrimPrefix(rule, "max=")
		} else if strings.HasPrefix(rule, "len=") {
			fieldInfo.Validations["len"] = strings.TrimPrefix(rule, "len=")
		} else if rule == "email" {
			fieldInfo.Validations["format"] = "email"
		} else if rule == "uuid" {
			fieldInfo.Validations["format"] = "uuid"
		} else if rule == "url" {
			fieldInfo.Validations["format"] = "uri"
		}
	}
}

// knownFieldDirectives lists all known field-level directives
var knownFieldDirectives = []string{
	SwaggerPrefix,
	ExampleDirective,
	DefaultDirective,
	RequiredDirective,
	NullableDirective,
	FormatDirective,
	InDirective,
	MinimumDirective,
	MaximumDirective,
	MinLengthDirective,
	MaxLengthDirective,
	PatternDirective,
	MinItemsDirective,
	MaxItemsDirective,
	UniqueItemsDirective,
	ReadOnlyDirective,
	WriteOnlyDirective,
	IgnoreDirective,
	OneOfDirective,
	AllOfDirective,
	AnyOfDirective,
}

// parseFieldDoc parses documentation comments for a field.
func parseFieldDoc(fieldInfo *FieldInfo, doc *ast.CommentGroup) {
	if doc == nil {
		return
	}

	// Check for ignore directive first
	if hasDirective(doc, IgnoreDirective) {
		return
	}

	comments := trimComments(doc)

	// Extract single-line directive values
	fieldInfo.Example = extractSingleLineValue(doc, ExampleDirective)
	fieldInfo.Default = extractSingleLineValue(doc, DefaultDirective)

	// Handle required directive
	if requiredValue := extractSingleLineValue(doc, RequiredDirective); requiredValue != "" {
		switch requiredValue {
		case "true":
			fieldInfo.Required = true
			fieldInfo.ExplicitRequired = true
		case "false":
			fieldInfo.Required = false
			fieldInfo.ExplicitOptional = true
		}
	}

	// Extract nullable directive
	if nullableVal := extractSingleLineValue(doc, NullableDirective); nullableVal == "true" {
		fieldInfo.Nullable = true
	}

	// Extract format directive
	if format := extractSingleLineValue(doc, FormatDirective); format != "" {
		fieldInfo.Validations["format"] = format
	}

	// Extract in directive (for parameter location)
	// Handle format like "in:header 'Origin'" - only take the first word (header)
	if inValue := extractSingleLineValue(doc, InDirective); inValue != "" {
		// Take only the first word (e.g., "header" from "header 'Origin'")
		inValue = strings.Fields(inValue)[0]
		fieldInfo.In = inValue
		if inValue == "body" {
			fieldInfo.IsRequestBody = true
		}
	}

	// Extract numeric constraints
	if minVal := extractSingleLineValue(doc, MinimumDirective); minVal != "" {
		fieldInfo.Validations["min"] = minVal
	}
	if maxVal := extractSingleLineValue(doc, MaximumDirective); maxVal != "" {
		fieldInfo.Validations["max"] = maxVal
	}

	// Extract string length constraints
	if minLenVal := extractSingleLineValue(doc, MinLengthDirective); minLenVal != "" {
		fieldInfo.Validations["minLength"] = minLenVal
	}
	if maxLenVal := extractSingleLineValue(doc, MaxLengthDirective); maxLenVal != "" {
		fieldInfo.Validations["maxLength"] = maxLenVal
	}

	// Extract pattern
	if pattern := extractSingleLineValue(doc, PatternDirective); pattern != "" {
		fieldInfo.Validations["pattern"] = pattern
	}

	// Extract array constraints
	if minItemsVal := extractSingleLineValue(doc, MinItemsDirective); minItemsVal != "" {
		fieldInfo.Validations["minItems"] = minItemsVal
	}
	if maxItemsVal := extractSingleLineValue(doc, MaxItemsDirective); maxItemsVal != "" {
		fieldInfo.Validations["maxItems"] = maxItemsVal
	}
	if hasDirective(doc, UniqueItemsDirective) {
		fieldInfo.Validations["uniqueItems"] = "true"
	}

	// Extract read/write constraints
	if hasDirective(doc, ReadOnlyDirective) {
		fieldInfo.Validations["readOnly"] = "true"
	}
	if hasDirective(doc, WriteOnlyDirective) {
		fieldInfo.Validations["writeOnly"] = "true"
	}

	// Extract description (non-directive lines)
	fieldInfo.Description = extractFieldDescription(comments, knownFieldDirectives)
}

// extractSingleLineValue extracts a single-line directive value.
func extractSingleLineValue(doc *ast.CommentGroup, directive string) string {
	if doc == nil {
		return ""
	}

	for _, comment := range trimComments(doc) {
		if strings.HasPrefix(comment, directive) {
			return strings.TrimSpace(strings.TrimPrefix(comment, directive))
		}
	}
	return ""
}

// extractFieldDescription extracts description lines, excluding directives.
func extractFieldDescription(comments []string, knownDirectives []string) string {
	var lines []string

	for _, comment := range comments {
		// Skip directives
		isDirective := false
		for _, directive := range knownDirectives {
			if strings.HasPrefix(comment, directive) {
				isDirective = true
				break
			}
		}

		if !isDirective && comment != "" {
			lines = append(lines, comment)
		}
	}

	if len(lines) == 0 {
		return ""
	}

	return strings.Join(lines, "\n")
}
