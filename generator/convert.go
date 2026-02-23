package generator

import (
	"slices"
	"strconv"
	"strings"

	"github.com/kausys/openapi/scanner"
	"github.com/kausys/openapi/spec"
)

// metaToInfo converts MetaInfo to spec.Info.
func (g *Generator) metaToInfo(meta *scanner.MetaInfo) *spec.Info {
	info := &spec.Info{
		Title:          meta.Title,
		Description:    meta.Description,
		TermsOfService: meta.TermsOfService,
		Version:        meta.Version,
	}

	if meta.Contact != nil {
		info.Contact = &spec.Contact{
			Name:  meta.Contact.Name,
			URL:   meta.Contact.URL,
			Email: meta.Contact.Email,
		}
	}

	if meta.License != nil {
		info.License = &spec.License{
			Name: meta.License.Name,
			URL:  meta.License.URL,
		}
	}

	return info
}

// enumToSchema converts EnumInfo to spec.Schema.
func (g *Generator) enumToSchema(e *scanner.EnumInfo) *spec.Schema {
	schema := &spec.Schema{
		Description: e.Description,
	}

	// Set the type based on BaseType
	g.setSchemaType(schema, e.BaseType)

	// Add enum values (sorted by key for consistent output)
	if len(e.Values) > 0 {
		schema.Enum = sortEnumValues(e.Values)
	}

	// Add example if available
	if e.Example != nil {
		schema.Example = e.Example
	}

	return schema
}

// sortEnumValues sorts enum values by key and returns a slice of values.
func sortEnumValues(values map[string]any) []any {
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	result := make([]any, 0, len(values))
	for _, k := range keys {
		result = append(result, values[k])
	}
	return result
}

// structToSchema converts StructInfo to spec.Schema.
func (g *Generator) structToSchema(s *scanner.StructInfo) *spec.Schema {
	// Handle oneOf/anyOf model schemas (pure composition, no type/properties)
	if s.IsOneOfModel {
		return g.compositionModelToSchema(s, s.OneOfOptions, s.OneOf)
	}
	if s.IsAnyOfModel {
		return g.compositionModelToSchema(s, s.AnyOfOptions, s.AnyOf)
	}

	// Handle non-struct underlying types (arrays, maps, primitives)
	switch s.UnderlyingKind {
	case scanner.KindArray:
		return &spec.Schema{
			Type:        scanner.TypeArray,
			Description: s.Description,
			Items:       g.typeToSchema(s.ElementType),
		}
	case scanner.KindMap:
		return &spec.Schema{
			Type:                 scanner.TypeObject,
			Description:          s.Description,
			AdditionalProperties: g.typeToSchema(s.ElementType),
		}
	case scanner.KindPrimitive:
		schema := &spec.Schema{
			Description: s.Description,
		}
		g.setSchemaType(schema, s.ElementType)
		return schema
	}

	schema := &spec.Schema{
		Type:        scanner.TypeObject,
		Description: s.Description,
		Properties:  make(map[string]*spec.Schema),
	}

	var required []string

	for _, field := range s.Fields {
		propName := g.getPropertyName(field)
		if propName == "" || propName == "-" {
			continue
		}

		propSchema := g.fieldToSchema(field)
		schema.Properties[propName] = propSchema

		if field.Required || field.ExplicitRequired {
			required = append(required, propName)
		}
	}

	if len(required) > 0 {
		schema.Required = required
	}

	// Handle legacy composition (mark references)
	if len(s.AllOf) > 0 {
		for _, ref := range s.AllOf {
			g.markSchemaAsReferenced(ref)
			schema.AllOf = append(schema.AllOf, &spec.Schema{
				Ref: "#/components/schemas/" + ref,
			})
		}
	}
	if len(s.OneOf) > 0 {
		for _, ref := range s.OneOf {
			g.markSchemaAsReferenced(ref)
			schema.OneOf = append(schema.OneOf, &spec.Schema{
				Ref: "#/components/schemas/" + ref,
			})
		}
	}
	if len(s.AnyOf) > 0 {
		for _, ref := range s.AnyOf {
			g.markSchemaAsReferenced(ref)
			schema.AnyOf = append(schema.AnyOf, &spec.Schema{
				Ref: "#/components/schemas/" + ref,
			})
		}
	}

	return schema
}

// compositionModelToSchema converts a swagger:oneOf or swagger:anyOf model to a composition schema.
func (g *Generator) compositionModelToSchema(s *scanner.StructInfo, options []string, refs []string) *spec.Schema {
	schema := &spec.Schema{
		Description: s.Description,
	}

	var schemas []*spec.Schema

	// Add options from embedded fields marked with swagger:oneOfOption/anyOfOption
	for _, typeName := range options {
		refName := g.resolveSchemaRef(typeName)
		g.markSchemaAsReferenced(refName)
		schemas = append(schemas, &spec.Schema{
			Ref: "#/components/schemas/" + refName,
		})
	}

	// Also support legacy inline oneOf:/anyOf: directive
	for _, ref := range refs {
		g.markSchemaAsReferenced(ref)
		schemas = append(schemas, &spec.Schema{
			Ref: "#/components/schemas/" + ref,
		})
	}

	if s.IsOneOfModel {
		schema.OneOf = schemas
	} else {
		schema.AnyOf = schemas
	}

	// Add discriminator if present
	if s.Discriminator != nil {
		schema.Discriminator = g.discriminatorToSpec(s.Discriminator)
	}

	return schema
}

// resolveSchemaRef resolves a type name to a schema reference name.
func (g *Generator) resolveSchemaRef(typeName string) string {
	// Check if there's a type mapping (Go type name -> model name)
	if modelName, ok := g.scanner.TypeToStruct[typeName]; ok {
		return modelName
	}

	// Check for package-qualified types (e.g., "dto.Agent")
	if strings.Contains(typeName, ".") {
		parts := strings.Split(typeName, ".")
		shortName := parts[len(parts)-1]

		// Check type mapping by short name
		if modelName, ok := g.scanner.TypeToStruct[shortName]; ok {
			return modelName
		}

		// Return short name if model exists
		if _, ok := g.scanner.Structs[shortName]; ok {
			return shortName
		}
	}

	return typeName
}

// discriminatorToSpec converts DiscriminatorInfo to spec.Discriminator.
func (g *Generator) discriminatorToSpec(d *scanner.DiscriminatorInfo) *spec.Discriminator {
	discriminator := &spec.Discriminator{
		PropertyName: d.PropertyName,
	}

	if len(d.Mapping) > 0 {
		discriminator.Mapping = make(map[string]string)
		for key, schemaName := range d.Mapping {
			// Resolve schema name to full reference
			refName := g.resolveSchemaRef(schemaName)
			discriminator.Mapping[key] = "#/components/schemas/" + refName
		}
	}

	return discriminator
}

// getPropertyName returns the JSON property name for a field.
func (g *Generator) getPropertyName(f *scanner.FieldInfo) string {
	if jsonName, ok := f.Tags["json"]; ok && jsonName != "" {
		return jsonName
	}
	return f.Name
}

// fieldToSchema converts FieldInfo to spec.Schema.
func (g *Generator) fieldToSchema(f *scanner.FieldInfo) *spec.Schema {
	// Handle arrays
	if f.IsArray {
		schema := &spec.Schema{
			Type:        scanner.TypeArray,
			Description: f.Description,
			Items:       g.typeToSchema(f.Type),
		}
		return schema
	}

	// Handle maps
	if f.IsMap {
		schema := &spec.Schema{
			Type:                 scanner.TypeObject,
			Description:          f.Description,
			AdditionalProperties: g.typeToSchema(f.Type),
		}
		return schema
	}

	// Check if this is a reference type (model or enum)
	// In OpenAPI 3.0.x, $ref cannot have other properties
	if g.isReferenceType(f.Type) {
		schema := g.typeToSchema(f.Type)
		// For enums, apply field-level overrides (description, example)
		if schema.Ref == "" && schema.Enum != nil {
			if f.Description != "" {
				schema.Description = f.Description
			}
			if f.Example != "" {
				schema.Example = f.Example
			}
		}
		return schema
	}

	// Handle basic/primitive types
	schema := &spec.Schema{
		Description: f.Description,
		Nullable:    f.Nullable,
	}

	g.setSchemaType(schema, f.Type)
	g.applyValidations(schema, f)

	// Set example (cast to schema type)
	if f.Example != "" {
		schema.Example = castToSchemaType(f.Example, schema.Type)
	}

	// Set default (cast to schema type)
	if f.Default != "" {
		schema.Default = castToSchemaType(f.Default, schema.Type)
	}

	return schema
}

// isReferenceType checks if a type should be a $ref (model or enum).
func (g *Generator) isReferenceType(typeName string) bool {
	// Check if it's a known model
	if _, ok := g.scanner.Structs[typeName]; ok {
		return true
	}
	// Check if it's an enum (using alias resolution)
	if g.scanner.GetEnumForType(typeName) != nil {
		return true
	}
	// Check for package-qualified types
	if strings.Contains(typeName, ".") {
		parts := strings.Split(typeName, ".")
		shortName := parts[len(parts)-1]
		if _, ok := g.scanner.Structs[shortName]; ok {
			return true
		}
	}
	return false
}

// typeToSchema converts a Go type name to a schema.
func (g *Generator) typeToSchema(typeName string) *spec.Schema {
	schema := &spec.Schema{}

	// Check if it's a known model (by model name)
	if _, ok := g.scanner.Structs[typeName]; ok {
		g.markSchemaAsReferenced(typeName)
		schema.Ref = "#/components/schemas/" + typeName
		return schema
	}

	// Check if there's a type mapping (Go type name -> model name)
	if modelName, ok := g.scanner.TypeToStruct[typeName]; ok {
		if _, exists := g.scanner.Structs[modelName]; exists {
			g.markSchemaAsReferenced(modelName)
			schema.Ref = "#/components/schemas/" + modelName
			return schema
		}
	}

	// Check for package-qualified types (e.g., "dto.Agent")
	if strings.Contains(typeName, ".") {
		parts := strings.Split(typeName, ".")
		shortName := parts[len(parts)-1]

		// Check by short name
		if _, ok := g.scanner.Structs[shortName]; ok {
			g.markSchemaAsReferenced(shortName)
			schema.Ref = "#/components/schemas/" + shortName
			return schema
		}

		// Check type mapping by short name
		if modelName, ok := g.scanner.TypeToStruct[shortName]; ok {
			if _, exists := g.scanner.Structs[modelName]; exists {
				g.markSchemaAsReferenced(modelName)
				schema.Ref = "#/components/schemas/" + modelName
				return schema
			}
		}
	}

	// Check if it's an enum - generate inline (using alias resolution)
	if enumInfo := g.scanner.GetEnumForType(typeName); enumInfo != nil {
		return g.createInlineEnumSchema(enumInfo)
	}

	g.setSchemaType(schema, typeName)
	return schema
}

// createInlineEnumSchema creates an inline schema for an enum.
func (g *Generator) createInlineEnumSchema(e *scanner.EnumInfo) *spec.Schema {
	schema := &spec.Schema{
		Description: e.Description,
	}
	g.setSchemaType(schema, e.BaseType)

	if len(e.Values) > 0 {
		schema.Enum = sortEnumValues(e.Values)
	}

	if e.Example != nil {
		schema.Example = e.Example
	}

	return schema
}

// setSchemaType sets the type and format for a schema based on Go type.
func (g *Generator) setSchemaType(schema *spec.Schema, goType string) {
	// Check for registered custom types first
	if typeInfo := GetCustomType(goType); typeInfo != nil {
		schema.Type = typeInfo.Type
		schema.Format = typeInfo.Format
		if typeInfo.Example != nil && schema.Example == nil {
			schema.Example = typeInfo.Example
		}
		if typeInfo.Default != nil && schema.Default == nil {
			schema.Default = typeInfo.Default
		}
		for k, v := range typeInfo.Validations {
			if schema.Format == "" && k == "format" {
				schema.Format = v
			}
		}
		return
	}

	switch goType {
	case "string":
		schema.Type = scanner.TypeString
	case "int", "int8", "int16", "int32":
		schema.Type = scanner.TypeInteger
		schema.Format = scanner.FormatInt32
	case "int64":
		schema.Type = scanner.TypeInteger
		schema.Format = scanner.FormatInt64
	case "uint", "uint8", "uint16", "uint32":
		schema.Type = scanner.TypeInteger
		schema.Format = scanner.FormatInt32
	case "uint64":
		schema.Type = scanner.TypeInteger
		schema.Format = scanner.FormatInt64
	case "float32":
		schema.Type = scanner.TypeNumber
		schema.Format = scanner.FormatFloat
	case "float64":
		schema.Type = scanner.TypeNumber
		schema.Format = scanner.FormatDouble
	case "bool":
		schema.Type = scanner.TypeBoolean
	default:
		// Check for package-qualified types
		if strings.Contains(goType, ".") {
			parts := strings.Split(goType, ".")
			typeName := parts[len(parts)-1]
			if _, ok := g.scanner.Structs[typeName]; ok {
				schema.Ref = "#/components/schemas/" + typeName
				return
			}
		}
		schema.Type = scanner.TypeString
	}
}

// castToSchemaType converts a string value to the appropriate Go type
// based on the OpenAPI schema type, so YAML serialization produces the correct type.
func castToSchemaType(value, schemaType string) any {
	switch schemaType {
	case scanner.TypeInteger:
		if v, err := strconv.ParseInt(value, 10, 64); err == nil {
			return v
		}
	case scanner.TypeNumber:
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			return v
		}
	case scanner.TypeBoolean:
		if v, err := strconv.ParseBool(value); err == nil {
			return v
		}
	}
	return value
}

// applyValidations applies validation rules to a schema.
func (g *Generator) applyValidations(schema *spec.Schema, f *scanner.FieldInfo) {
	if format, ok := f.Validations["format"]; ok {
		schema.Format = format
	}

	if min, ok := f.Validations["min"]; ok {
		if v, err := strconv.ParseFloat(min, 64); err == nil {
			schema.Minimum = new(v)
		}
	}

	if max, ok := f.Validations["max"]; ok {
		if v, err := strconv.ParseFloat(max, 64); err == nil {
			schema.Maximum = new(v)
		}
	}

	if minLen, ok := f.Validations["minLength"]; ok {
		if v, err := strconv.ParseUint(minLen, 10, 64); err == nil {
			schema.MinLength = v
		}
	}

	if maxLen, ok := f.Validations["maxLength"]; ok {
		if v, err := strconv.ParseUint(maxLen, 10, 64); err == nil {
			schema.MaxLength = new(v)
		}
	}

	if pattern, ok := f.Validations["pattern"]; ok {
		schema.Pattern = pattern
	}
}

// routeToOperation converts RouteInfo to spec.Operation.
func (g *Generator) routeToOperation(r *scanner.RouteInfo) *spec.Operation {
	responses := &spec.Responses{
		StatusCodes: make(map[string]*spec.Response),
	}

	// Add responses
	for _, resp := range r.Responses {
		response := &spec.Response{
			Description: resp.Description,
		}

		if resp.Type != "" {
			schema := g.typeToSchema(resp.Type)
			if resp.IsArray {
				schema = &spec.Schema{
					Type:  scanner.TypeArray,
					Items: schema,
				}
			}
			response.Content = map[string]*spec.MediaType{
				"application/json": {
					Schema: schema,
				},
			}
		}

		if resp.StatusCode == "default" {
			responses.Default = response
		} else {
			responses.StatusCodes[resp.StatusCode] = response
		}
	}

	op := &spec.Operation{
		OperationID: r.OperationID,
		Summary:     r.Summary,
		Description: r.Description,
		Tags:        r.Tags,
		Deprecated:  r.Deprecated,
		Responses:   responses,
	}

	// Add parameters and request body from swagger:parameters struct matching operationID
	params, requestBody := g.getOperationParameters(r)
	if len(params) > 0 {
		op.Parameters = params
	}

	// Only add requestBody for methods that support it (POST, PUT, PATCH)
	// GET, HEAD, DELETE do not have well-defined semantics for request body
	method := strings.ToUpper(r.Method)
	if requestBody != nil && (method == "POST" || method == "PUT" || method == "PATCH") {
		op.RequestBody = requestBody
	}

	// Add security
	if len(r.Security) > 0 {
		for _, scheme := range r.Security {
			op.Security = append(op.Security, &spec.SecurityRequirement{
				Requirements: map[string][]string{
					scheme: {},
				},
			})
		}
	}

	return op
}

// getOperationParameters finds and converts parameters for an operation.
func (g *Generator) getOperationParameters(r *scanner.RouteInfo) ([]*spec.Parameter, *spec.RequestBody) {
	// Look for a struct marked as swagger:parameters with matching operationID
	paramStruct, ok := g.scanner.Structs[r.OperationID]
	if !ok || !paramStruct.IsParameter {
		return nil, nil
	}

	var params []*spec.Parameter
	var requestBody *spec.RequestBody
	ignoredParams := make(map[string]bool)
	for _, ignored := range r.IgnoredParameters {
		ignoredParams[ignored] = true
	}

	for _, field := range paramStruct.Fields {
		// Skip ignored parameters
		paramName := g.getPropertyName(field)
		if ignoredParams[paramName] {
			continue
		}

		// Handle request body (in:body)
		if field.IsRequestBody || field.In == "body" {
			requestBody = g.fieldToRequestBody(field, r.Consumes)
			continue
		}

		param := g.fieldToParameter(field, r.Path)
		if param != nil {
			params = append(params, param)
		}
	}

	return params, requestBody
}

// fieldToRequestBody converts a FieldInfo to spec.RequestBody.
// consumes specifies the content types from the route's Consumes directive.
func (g *Generator) fieldToRequestBody(f *scanner.FieldInfo, consumes []string) *spec.RequestBody {
	var schema *spec.Schema

	// Handle inline structs
	if f.IsInlineStruct && f.InlineStruct != nil {
		schema = g.inlineStructToSchema(f.InlineStruct)
	} else {
		schema = g.typeToSchema(f.Type)
	}

	// Determine content types to use
	contentTypes := consumes
	if len(contentTypes) == 0 {
		// Default to application/json if no consumes specified
		contentTypes = []string{"application/json"}
	}

	// Build content map for each content type
	content := make(map[string]*spec.MediaType)
	for _, contentType := range contentTypes {
		mediaType := &spec.MediaType{}

		// Handle multipart/form-data specially
		if contentType == scanner.ContentTypeMultipart {
			mediaType.Schema = g.createMultipartSchema(f, schema)
			mediaType.Encoding = g.createMultipartEncoding(f, schema)
		} else {
			// For other content types (JSON, XML, etc.), use the schema as-is
			mediaType.Schema = schema
		}

		content[contentType] = mediaType
	}

	return &spec.RequestBody{
		Description: f.Description,
		Required:    f.Required,
		Content:     content,
	}
}

// inlineStructToSchema converts an inline StructInfo to spec.Schema.
func (g *Generator) inlineStructToSchema(s *scanner.StructInfo) *spec.Schema {
	schema := &spec.Schema{
		Type:        scanner.TypeObject,
		Description: s.Description,
		Properties:  make(map[string]*spec.Schema),
	}

	var required []string

	for _, field := range s.Fields {
		propName := g.getPropertyName(field)
		if propName == "" || propName == "-" {
			continue
		}

		propSchema := g.fieldToSchema(field)
		schema.Properties[propName] = propSchema

		if field.Required || field.ExplicitRequired {
			required = append(required, propName)
		}
	}

	if len(required) > 0 {
		schema.Required = required
	}

	return schema
}

// fieldToParameter converts a FieldInfo to spec.Parameter.
func (g *Generator) fieldToParameter(f *scanner.FieldInfo, path string) *spec.Parameter {
	paramName := g.getPropertyName(f)
	if paramName == "" || paramName == "-" {
		return nil
	}

	// Determine parameter location (in)
	in := f.In
	if in == "" {
		in = "query" // default
	}
	// Check if it's a path parameter
	if strings.Contains(path, "{"+paramName+"}") {
		in = "path"
	}

	// Create schema for the parameter
	var schema *spec.Schema

	// Check if the field type is an enum
	if enumInfo := g.findEnumInfo(f.Type); enumInfo != nil {
		schema = g.createInlineEnumSchema(enumInfo)
		// Override example if field has its own
		if f.Example != "" {
			schema.Example = castToSchemaType(f.Example, schema.Type)
		}
	} else {
		schema = &spec.Schema{}
		g.setSchemaType(schema, f.Type)
		g.applyValidations(schema, f)

		if f.Example != "" {
			schema.Example = castToSchemaType(f.Example, schema.Type)
		}
		if f.Default != "" {
			schema.Default = castToSchemaType(f.Default, schema.Type)
		}
	}

	param := &spec.Parameter{
		Name:        paramName,
		In:          in,
		Description: f.Description,
		Required:    f.Required || in == "path", // path parameters are always required
		Schema:      schema,
	}

	return param
}

// findEnumInfo finds EnumInfo for a given type name.
func (g *Generator) findEnumInfo(typeName string) *scanner.EnumInfo {
	// Use scanner's GetEnumForType which handles aliases
	return g.scanner.GetEnumForType(typeName)
}

// addRoute adds a route to the OpenAPI spec.
func (g *Generator) addRoute(openAPI *spec.OpenAPI, r *scanner.RouteInfo) {
	if openAPI.Paths == nil {
		openAPI.Paths = &spec.Paths{
			PathItems: make(map[string]*spec.PathItem),
		}
	}

	pathItem, exists := openAPI.Paths.PathItems[r.Path]
	if !exists {
		pathItem = &spec.PathItem{}
		openAPI.Paths.PathItems[r.Path] = pathItem
	}

	op := g.routeToOperation(r)

	switch strings.ToUpper(r.Method) {
	case "GET":
		pathItem.Get = op
	case "POST":
		pathItem.Post = op
	case "PUT":
		pathItem.Put = op
	case "DELETE":
		pathItem.Delete = op
	case "PATCH":
		pathItem.Patch = op
	case "HEAD":
		pathItem.Head = op
	case "OPTIONS":
		pathItem.Options = op
	}
}

// securitySchemeToSpec converts SecuritySchemeInfo to spec.SecurityScheme.
func (g *Generator) securitySchemeToSpec(s *scanner.SecuritySchemeInfo) *spec.SecurityScheme {
	return &spec.SecurityScheme{
		Type:         s.Type,
		Description:  s.Description,
		Name:         s.Name,
		In:           s.In,
		Scheme:       s.Scheme,
		BearerFormat: s.BearerFormat,
	}
}

// createMultipartSchema creates a schema for multipart/form-data requests.
// For multipart, the schema must be type: object with properties for each part.
func (g *Generator) createMultipartSchema(f *scanner.FieldInfo, originalSchema *spec.Schema) *spec.Schema {
	// If the field is an inline struct, use it directly as it already has properties
	if f.IsInlineStruct && f.InlineStruct != nil {
		return originalSchema
	}

	// If it's a reference to a model, use it as-is
	// The schema should already be type: object with properties
	return originalSchema
}

// createMultipartEncoding creates encoding information for multipart/form-data.
// This is used to specify content types for file upload fields.
func (g *Generator) createMultipartEncoding(f *scanner.FieldInfo, schema *spec.Schema) map[string]*spec.Encoding {
	// Only create encoding if we have an inline struct with fields
	if !f.IsInlineStruct || f.InlineStruct == nil {
		return nil
	}

	encoding := make(map[string]*spec.Encoding)

	// Check each field in the struct for file upload fields (format: binary)
	for _, field := range f.InlineStruct.Fields {
		propName := g.getPropertyName(field)
		if propName == "" || propName == "-" {
			continue
		}

		// Check if this field is a file upload (format: binary)
		if format, ok := field.Validations["format"]; ok && format == scanner.FormatBinary {
			// Add encoding for this field
			encoding[propName] = &spec.Encoding{
				ContentType: "application/octet-stream",
			}
		}
	}

	if len(encoding) == 0 {
		return nil
	}

	return encoding
}
