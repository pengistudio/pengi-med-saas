package sri_document

import (
	"time"

	"pengi-med-saas/core/pdfrender"
	"pengi-med-saas/core/tenantfiles"
)

// Gateway is everything the lifecycle needs from the outside SRI world: signing
// (done by the sri-xml-signer service) and the SRI's reception and authorization
// web services (proxied by the same service).
type Gateway interface {
	// Sign returns the XAdES-signed XML. Errors wrapping ErrUnreachable mean the
	// signer could not be reached.
	Sign(p12 []byte, password string, xml string) (string, error)
	// SubmitForReception sends a signed comprobante to the SRI. nil means the SRI
	// has it, including when it was already registered from an earlier attempt.
	// A *Rejection means the SRI returned it (DEVUELTA).
	SubmitForReception(signedXML string, sriEnv string) error
	// QueryAuthorization asks the SRI for the authorization of a received
	// comprobante without resending it.
	QueryAuthorization(accessKey string, sriEnv string) (Authorization, error)
}

// Documents are the tenant files and PDF rendering the lifecycle and its kinds
// use: the P12 certificate, signed XML and printed documents such as the RIDE.
type Documents struct {
	Files    tenantfiles.Store
	Renderer *pdfrender.Renderer
}

// Publisher enqueues a processing task for the lifecycle's consumers.
type Publisher interface {
	Publish(queue string, body []byte) error
}

type AuthorizationStatus int

const (
	// AuthorizationPending: the SRI has not decided yet ("EN PROCESAMIENTO").
	AuthorizationPending AuthorizationStatus = iota
	Authorized
	NotAuthorized
)

type Authorization struct {
	Status AuthorizationStatus
	// AuthorizedAt is the SRI's fechaAutorizacion; set only when Authorized.
	AuthorizedAt time.Time
	// Message is the SRI's reason; set only when NotAuthorized.
	Message string

	// The rest is set only when Authorized: what the SRI's <autorizacion>
	// carries, to rebuild the authorized XML (see buildAuthorizedXML).
	//
	// Number is the numeroAutorizacion (equal to the access key in the offline
	// scheme). RawAuthorizedAt is fechaAutorizacion as the SRI wrote it.
	// Environment is the ambiente text ("PRUEBAS" / "PRODUCCIÓN"). The
	// comprobante itself is not taken from the answer: the authorized XML wraps
	// our stored signed XML, which is byte-exact.
	Number          string
	RawAuthorizedAt string
	Environment     string
	Messages        []AuthorizationMessage
}

// AuthorizationMessage is one <mensaje> of the SRI's authorization answer.
type AuthorizationMessage struct {
	Identifier     string
	Message        string
	AdditionalInfo string
	Type           string
}
