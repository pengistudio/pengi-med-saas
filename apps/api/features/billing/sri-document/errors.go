package sri_document

import "errors"

// ErrUnreachable marks a failure to reach the signer or the SRI (timeout,
// connection refused, SOAP fault) as opposed to a verdict on the document.
var ErrUnreachable = errors.New("sri gateway unreachable")

// errTransient marks infrastructure failures on our side (e.g. the database)
// worth retrying.
var errTransient = errors.New("transient lifecycle error")

// IsRetryable reports whether a Process error should be retried automatically.
// Retrying is always safe: the access key never changes (ADR 0001) and the SRI
// answers "already registered" for a comprobante it already received.
func IsRetryable(err error) bool {
	return errors.Is(err, ErrUnreachable) || errors.Is(err, errTransient)
}

// Error codes are i18n keys persisted in a document's error_code so the UI can
// explain a failure in the viewer's language; error_message keeps the raw detail.
const (
	ErrorCodeConnection       = "billing.sri_document.error.connection"
	ErrorCodeReturned         = "billing.sri_document.error.returned"
	ErrorCodeNotAuthorized    = "billing.sri_document.error.not_authorized"
	ErrorCodeMissingSignature = "billing.sri_document.error.missing_signature"
	// ErrorCodeSignatureExpired reuses the key the */sri/process handlers answer
	// with when refusing to queue a document for the same reason.
	ErrorCodeSignatureExpired = "billing.sri.error.signature_expired"
	ErrorCodeInternal         = "billing.sri_document.error.internal"
)

// Rejection is the SRI's verdict against a comprobante: DEVUELTA at reception or
// NO AUTORIZADO at authorization. Message carries the SRI's reason.
type Rejection struct {
	Message string
}

func (r *Rejection) Error() string { return r.Message }

// Enqueue refusals.
var (
	ErrNotFound          = errors.New("sri document not found")
	ErrAlreadyAuthorized = errors.New("sri document already authorized")
	// ErrSignatureExpired: the tenant's P12 certificate has expired, so the
	// document cannot be signed until a valid one is uploaded.
	ErrSignatureExpired = errors.New("sri signature certificate expired")
)
