package spec

// License information for the exposed API.
//
// See: https://spec.openapis.org/oas/v3.0.4.html#license-object
type License struct {
	// REQUIRED. The license name used for the API.
	Name string `json:"name" yaml:"name"`
	// A URL for the license used for the API. This MUST be in the form of a URL.
	URL string `json:"url,omitempty" yaml:"url,omitempty"`
}
