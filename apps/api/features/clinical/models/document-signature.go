package clinical_models

import "time"

// DocumentSignature records who electronically signed a document and where
// its signed PDF lives (tenantfiles). A signed PDF is immutable: download and
// email serve it instead of re-rendering.
type DocumentSignature struct {
	SignedByID *uint      `json:"signed_by_id,omitempty"`
	SignedAt   *time.Time `json:"signed_at,omitempty"`
	SignerName string     `json:"signer_name,omitempty"`
	SignedFile string     `json:"-"`
}

// IsSigned reports whether the document has a signed PDF.
func (s DocumentSignature) IsSigned() bool { return s.SignedFile != "" }
