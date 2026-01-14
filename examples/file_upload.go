package examples

// This example demonstrates how to document file upload endpoints
// using multipart/form-data with the OpenAPI generator.

// swagger:meta
// Title: File Upload API
// Version: 1.0.0
// Description: API for uploading files with metadata
type Meta struct{}

// UploadSingleFileParams defines parameters for single file upload
//
// swagger:parameters uploadSingleFile
type UploadSingleFileParams struct {
	// in:body
	Body struct {
		// The file to upload
		// format: binary
		File string `json:"file"`
	}
}

// swagger:route POST /upload/single files uploadSingleFile
// Consumes:
// - multipart/form-data
//
// Upload a single file
//
// Responses:
//   200: uploadResponse
func UploadSingleFile() {}

// UploadMultipleFilesParams defines parameters for multiple file upload
//
// swagger:parameters uploadMultipleFiles
type UploadMultipleFilesParams struct {
	// in:body
	Body struct {
		// Profile picture
		// format: binary
		Avatar string `json:"avatar"`
		// Resume document
		// format: binary
		Resume string `json:"resume"`
	}
}

// swagger:route POST /upload/multiple files uploadMultipleFiles
// Consumes:
// - multipart/form-data
//
// Upload multiple files
//
// Responses:
//   200: uploadResponse
func UploadMultipleFiles() {}

// UploadWithMetadataParams defines parameters for file upload with metadata
//
// swagger:parameters uploadWithMetadata
type UploadWithMetadataParams struct {
	// in:body
	Body struct {
		// The file to upload
		// format: binary
		File string `json:"file"`
		// File title
		// required: true
		Title string `json:"title"`
		// File description
		Description string `json:"description"`
		// File tags
		Tags []string `json:"tags"`
		// Is file public
		IsPublic bool `json:"is_public"`
	}
}

// swagger:route POST /upload/metadata files uploadWithMetadata
// Consumes:
// - multipart/form-data
//
// Upload file with metadata
//
// Upload a file along with additional metadata fields.
// The file is uploaded as binary data, while other fields
// are sent as regular form fields.
//
// Responses:
//   200: uploadResponse
//   400: errorResponse
func UploadWithMetadata() {}

// UploadFlexibleParams defines parameters for flexible upload
//
// swagger:parameters uploadFlexible
type UploadFlexibleParams struct {
	// in:body
	Body struct {
		// The file to upload (can be binary or base64)
		// format: binary
		File string `json:"file"`
	}
}

// swagger:route POST /upload/flexible files uploadFlexible
// Consumes:
// - multipart/form-data
// - application/json
//
// Upload with multiple content types
//
// This endpoint accepts both multipart/form-data (for binary uploads)
// and application/json (for base64 encoded files).
//
// Responses:
//   200: uploadResponse
func UploadFlexible() {}

// UploadResponse represents a successful upload response
//
// swagger:model
type UploadResponse struct {
	// Upload ID
	// Example: abc123
	ID string `json:"id"`
	// File name
	// Example: document.pdf
	Filename string `json:"filename"`
	// File size in bytes
	// Example: 1024
	Size int64 `json:"size"`
	// Upload timestamp
	// Example: 2024-01-15T10:30:00Z
	UploadedAt string `json:"uploaded_at"`
}

// ErrorResponse represents an error response
//
// swagger:model
type ErrorResponse struct {
	// Error message
	// Example: Invalid file format
	Message string `json:"message"`
}

