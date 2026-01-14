package examples

// This example shows how to document a real KYC document upload endpoint
// with query parameters and file upload

// KYBDocumentType represents the type of business document
//
// swagger:enum
type KYBDocumentType string

const (
	ArticlesOfIncorporation    KYBDocumentType = "articles_of_incorporation"
	CertificateOfIncorporation KYBDocumentType = "certificate_of_incorporation"
	DBA                        KYBDocumentType = "dba"
	EINConfirmation            KYBDocumentType = "ein_confirmation"
	Formation                  KYBDocumentType = "formation"
	Other                      KYBDocumentType = "other"
	ProofOfAddress             KYBDocumentType = "proof_of_address"
	OrganizationChart          KYBDocumentType = "organization_chart"
)

// UploadDocumentParams defines parameters for document upload
//
// swagger:parameters uploadDocument
type UploadDocumentParams struct {
	// The account ID to upload the document for
	// in: query
	// required: true
	// example: 00000000-0000-0000-0000-000000000000
	AccountID string `json:"accountId"`

	// The document type
	// in: query
	// required: true
	// example: certificate_of_incorporation
	DocumentType KYBDocumentType `json:"documentType"`

	// in:form
	Body struct {
		// The file to upload
		// required: true
		// format: binary
		File string `json:"file"`
	}
}

// Alternative approach: Using a reference to a DTO struct
// This is useful when you want to reuse the same struct in your actual handler code
//
// swagger:parameters uploadDocumentV2
type UploadDocumentV2Params struct {
	// The account ID to upload the document for
	// in: query
	// required: true
	AccountID string `json:"accountId"`

	// The document type
	// in: query
	// required: true
	DocumentType KYBDocumentType `json:"documentType"`

	// in:form
	Body UploadDocumentRequest
}

// UploadDocumentRequest represents the actual DTO used in your handler
// This can be your real struct with *multipart.FileHeader
type UploadDocumentRequest struct {
	// The file to upload
	// required: true
	// format: binary
	File string `json:"file"`
}

// swagger:route POST /api/v2/kyc/banking-verification/business/upload-document KYC uploadDocument
// Consumes:
// - multipart/form-data
//
// # Upload Document
//
// This endpoint allows to upload a document for a specific account.
//
// **Required permissions:**
// - `kyc:create` OR
// - `kyc:manage`
//
// Security:
// - bearer
//
// Responses:
//
//	200: UploadDocumentResponse
//	400: errorResponse
//	500: errorResponse
func UploadDocument() {}

// UploadDocumentResponse represents the response from document upload
//
// swagger:model
type UploadDocumentResponse struct {
	// Document ID
	// example: doc_123456
	DocumentID string `json:"documentId"`
	// Upload status
	// example: success
	Status string `json:"status"`
}
