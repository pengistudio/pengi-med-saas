package sri_document

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"pengi-med-saas/core/utils"
)

// HTTPGateway is the production Gateway: the sri-xml-signer service, which signs
// and proxies the SRI's reception and authorization SOAP services. It translates
// the service's error strings into the lifecycle's outcomes.
type HTTPGateway struct {
	client *utils.SriSignerClient
}

func NewHTTPGateway() *HTTPGateway {
	return &HTTPGateway{client: utils.NewSriSignerClient()}
}

func (g *HTTPGateway) Sign(p12 []byte, password string, xml string) (string, error) {
	resp, err := g.client.SignXML(p12, password, []byte(xml))
	if err != nil {
		if isTransport(err) {
			return "", fmt.Errorf("%w: %w", ErrUnreachable, err)
		}
		return "", err
	}
	return resp.XML, nil
}

func (g *HTTPGateway) SubmitForReception(signedXML string, sriEnv string) error {
	_, err := g.client.ValidateXMLWithSRI([]byte(signedXML), signerEnv(sriEnv))
	if err == nil {
		return nil
	}
	msg := err.Error()
	switch {
	case isTransport(err):
		return fmt.Errorf("%w: %w", ErrUnreachable, err)
	// The SRI already has this comprobante from an earlier attempt: same access
	// key (ID 43) or same sequential (ID 45). Both mean it was received.
	case isSriVerdict(msg) && (strings.Contains(msg, "(ID 43)") || strings.Contains(msg, "(ID 45)")):
		return nil
	case isSriVerdict(msg):
		return &Rejection{Message: msg}
	default:
		// No verdict from the SRI: its reception service could not be reached.
		return fmt.Errorf("%w: %w", ErrUnreachable, err)
	}
}

func (g *HTTPGateway) QueryAuthorization(accessKey string, sriEnv string) (Authorization, error) {
	resp, err := g.client.AuthorizeXMLWithSRI(accessKey, signerEnv(sriEnv))
	if err == nil {
		return Authorization{Status: Authorized, AuthorizedAt: parseFechaAutorizacion(resp.FechaAutorizacion)}, nil
	}
	msg := err.Error()
	upper := strings.ToUpper(msg)
	switch {
	case isTransport(err):
		return Authorization{}, fmt.Errorf("%w: %w", ErrUnreachable, err)
	case isSriVerdict(msg), strings.Contains(upper, "NO AUTORIZADO"), strings.Contains(upper, "RECHAZADA"):
		return Authorization{Status: NotAuthorized, Message: msg}, nil
	// "EN PROCESAMIENTO", or no <autorizacion> yet for this key.
	case strings.Contains(upper, "EN PROCESAMIENTO"), strings.Contains(upper, "AUTORIZACION"), strings.Contains(upper, "AUTORIZACIÓN"):
		return Authorization{Status: AuthorizationPending}, nil
	default:
		return Authorization{}, fmt.Errorf("%w: %w", ErrUnreachable, err)
	}
}

// isTransport reports whether the signer service itself could not be reached.
func isTransport(err error) bool {
	var urlErr *url.Error
	return errors.As(err, &urlErr)
}

// isSriVerdict reports whether a message is a structured SRI response
// ("[SRI-ERROR] ... (ID nn) - ...") rather than an infrastructure failure.
func isSriVerdict(msg string) bool {
	return strings.Contains(msg, "[SRI-")
}

// signerEnv translates the SRI ambiente code ("1"/"2") into the "test"/"prod"
// path segment of the sri-xml-signer routes.
func signerEnv(sriEnv string) string {
	if sriEnv == "2" {
		return "prod"
	}
	return "test"
}

func parseFechaAutorizacion(raw string) time.Time {
	if t, err := time.Parse("2006-01-02T15:04:05.999-07:00", raw); err == nil {
		return t
	}
	return time.Now()
}
