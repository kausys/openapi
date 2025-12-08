package spec

// The Schema Object allows the definition of input and output data types. These types can be
// objects, but also primitives and arrays. This object is an extended subset of the JSON Schema
// Specification Draft Wright-00.
//
// See: https://spec.openapis.org/oas/v3.0.4.html#schema-object
type Schema struct {
	// A reference to another schema. If present, other properties are ignored.
	Ref string `json:"$ref,omitempty" yaml:"$ref,omitempty"`

	// JSON Schema keywords taken directly from JSON Schema
	// See: https://spec.openapis.org/oas/v3.0.4.html#json-schema-keywords

	// The title of the schema.
	Title string `json:"title,omitempty" yaml:"title,omitempty"`
	// A number that the value must be a multiple of.
	MultipleOf *float64 `json:"multipleOf,omitempty" yaml:"multipleOf,omitempty"`
	// The maximum value for a number.
	Maximum *float64 `json:"maximum,omitempty" yaml:"maximum,omitempty"`
	// When true, indicates that the maximum value is exclusive.
	ExclusiveMaximum bool `json:"exclusiveMaximum,omitempty" yaml:"exclusiveMaximum,omitempty"`
	// The minimum value for a number.
	Minimum *float64 `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	// When true, indicates that the minimum value is exclusive.
	ExclusiveMinimum bool `json:"exclusiveMinimum,omitempty" yaml:"exclusiveMinimum,omitempty"`
	// The maximum length for a string.
	MaxLength *uint64 `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
	// The minimum length for a string.
	MinLength uint64 `json:"minLength,omitempty" yaml:"minLength,omitempty"`
	// A regular expression that a string value must match. This string SHOULD be a valid regular
	// expression, according to the Ecma-262 Edition 5.1 regular expression dialect.
	Pattern string `json:"pattern,omitempty" yaml:"pattern,omitempty"`
	// The maximum number of items in an array.
	MaxItems *uint64 `json:"maxItems,omitempty" yaml:"maxItems,omitempty"`
	// The minimum number of items in an array.
	MinItems *uint64 `json:"minItems,omitempty" yaml:"minItems,omitempty"`
	// Indicates whether items in an array must be unique.
	UniqueItems bool `json:"uniqueItems,omitempty" yaml:"uniqueItems,omitempty"`
	// The maximum number of properties in an object.
	MaxProperties *uint64 `json:"maxProperties,omitempty" yaml:"maxProperties,omitempty"`
	// The minimum number of properties in an object.
	MinProperties *uint64 `json:"minProperties,omitempty" yaml:"minProperties,omitempty"`
	// A list of required properties when type is "object".
	Required []string `json:"required,omitempty" yaml:"required,omitempty"`
	// The enumeration of possible values.
	Enum []any `json:"enum,omitempty" yaml:"enum,omitempty"`

	// JSON Schema keywords with adjusted definitions for OpenAPI

	// Value MUST be a string. Multiple types via an array are not supported.
	Type string `json:"type,omitempty" yaml:"type,omitempty"`
	// Inline or referenced schema MUST be of a Schema Object and not a standard JSON Schema.
	AllOf []*Schema `json:"allOf,omitempty" yaml:"allOf,omitempty"`
	// Inline or referenced schema MUST be of a Schema Object and not a standard JSON Schema.
	OneOf []*Schema `json:"oneOf,omitempty" yaml:"oneOf,omitempty"`
	// Inline or referenced schema MUST be of a Schema Object and not a standard JSON Schema.
	AnyOf []*Schema `json:"anyOf,omitempty" yaml:"anyOf,omitempty"`
	// Inline or referenced schema MUST be of a Schema Object and not a standard JSON Schema.
	Not *Schema `json:"not,omitempty" yaml:"not,omitempty"`
	// Value MUST be an object and not an array. Inline or referenced schema MUST be of a Schema
	// Object and not a standard JSON Schema. items MUST be present if type is "array".
	Items *Schema `json:"items,omitempty" yaml:"items,omitempty"`
	// Property definitions MUST be a Schema Object and not a standard JSON Schema (inline or
	// referenced).
	Properties map[string]*Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
	// Value can be boolean or object. Inline or referenced schema MUST be of a Schema Object and not
	// a standard JSON Schema. Consistent with JSON Schema, additionalProperties defaults to true.
	AdditionalProperties *Schema `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`
	// CommonMark syntax MAY be used for rich text representation.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	// See Data Type Formats for further details. While relying on JSON Schema's defined formats, the
	// OAS offers a few additional predefined formats.
	Format string `json:"format,omitempty" yaml:"format,omitempty"`
	// The default value represents what would be assumed by the consumer of the input as the value of
	// the schema if one is not provided. Unlike JSON Schema, the value MUST conform to the defined
	// type for the Schema Object defined at the same level.
	Default any `json:"default,omitempty" yaml:"default,omitempty"`

	// OpenAPI-specific fixed fields

	// This keyword only takes effect if type is explicitly defined within the same Schema Object. A
	// true value indicates that both null values and values of the type specified by type are
	// allowed. Other Schema Object constraints retain their defined behavior, and therefore may
	// disallow the use of null as a value. A false value leaves the specified or default type
	// unmodified. The default value is false.
	Nullable bool `json:"nullable,omitempty" yaml:"nullable,omitempty"`
	// Adds support for polymorphism. The discriminator is used to determine which of a set of schemas
	// a payload is expected to satisfy. See Composition and Inheritance for more details.
	Discriminator *Discriminator `json:"discriminator,omitempty" yaml:"discriminator,omitempty"`
	// Relevant only for Schema Object properties definitions. Declares the property as "read only".
	// This means that it MAY be sent as part of a response but SHOULD NOT be sent as part of the
	// request. If the property is marked as readOnly being true and is in the required list, the
	// required will take effect on the response only. A property MUST NOT be marked as both readOnly
	// and writeOnly being true. Default value is false.
	ReadOnly bool `json:"readOnly,omitempty" yaml:"readOnly,omitempty"`
	// Relevant only for Schema Object properties definitions. Declares the property as "write only".
	// Therefore, it MAY be sent as part of a request but SHOULD NOT be sent as part of the response.
	// If the property is marked as writeOnly being true and is in the required list, the required
	// will take effect on the request only. A property MUST NOT be marked as both readOnly and
	// writeOnly being true. Default value is false.
	WriteOnly bool `json:"writeOnly,omitempty" yaml:"writeOnly,omitempty"`
	// This MAY be used only on property schemas. It has no effect on root schemas. Adds additional
	// metadata to describe the XML representation of this property.
	XML *XML `json:"xml,omitempty" yaml:"xml,omitempty"`
	// Additional external documentation for this schema.
	ExternalDocs *ExternalDocs `json:"externalDocs,omitempty" yaml:"externalDocs,omitempty"`
	// A free-form field to include an example of an instance for this schema. To represent examples
	// that cannot be naturally represented in JSON or YAML, a string value can be used to contain the
	// example with escaping where necessary.
	Example any `json:"example,omitempty" yaml:"example,omitempty"`
	// Specifies that a schema is deprecated and SHOULD be transitioned out of usage. Default value is
	// false.
	Deprecated bool `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
}
