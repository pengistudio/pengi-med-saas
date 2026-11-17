package sri_document

import (
	"encoding/json"
	"encoding/xml"
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
		return authorizedFrom(resp), nil
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

// authorizedFrom maps the signer's AUTORIZADO answer. The scalar texts come from
// the raw <autorizacion> (comprobanteCrudo) when it parses, since the signer's
// SOAP client re-serializes fechaAutorizacion. The comprobante is not used (see
// buildAuthorizedXML).
func authorizedFrom(resp *utils.SriAuthorizationResponse) Authorization {
	auth := Authorization{
		Status:          Authorized,
		AuthorizedAt:    parseFechaAutorizacion(resp.FechaAutorizacion),
		Number:          string(resp.NumeroAutorizacion),
		RawAuthorizedAt: resp.FechaAutorizacion,
		Environment:     resp.Ambiente,
		Messages:        parseMensajes(resp.Mensajes),
	}
	crudo := string(resp.ComprobanteCrudo)
	var raw struct {
		NumeroAutorizacion string `xml:"numeroAutorizacion"`
		FechaAutorizacion  string `xml:"fechaAutorizacion"`
		Ambiente           string `xml:"ambiente"`
	}
	if crudo != "" && xml.Unmarshal([]byte(crudo), &raw) == nil {
		if v := strings.TrimSpace(raw.NumeroAutorizacion); v != "" {
			auth.Number = v
		}
		if v := strings.TrimSpace(raw.FechaAutorizacion); v != "" {
			auth.RawAuthorizedAt = v
		}
		if v := strings.TrimSpace(raw.Ambiente); v != "" {
			auth.Environment = v
		}
	}
	return auth
}

// parseMensajes reads the signer's mensajes (null or an array of objects);
// anything else yields none. Values are stringified whatever their JSON type.
func parseMensajes(raw json.RawMessage) []AuthorizationMessage {
	var items []map[string]any
	if len(raw) == 0 || json.Unmarshal(raw, &items) != nil {
		return nil
	}
	text := func(m map[string]any, key string) string {
		if v, ok := m[key]; ok && v != nil {
			return fmt.Sprint(v)
		}
		return ""
	}
	var out []AuthorizationMessage
	for _, m := range items {
		out = append(out, AuthorizationMessage{
			Identifier:     text(m, "identificador"),
			Message:        text(m, "mensaje"),
			AdditionalInfo: text(m, "informacionAdicional"),
			Type:           text(m, "tipo"),
		})
	}
	return out
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

// parseFechaAutorizacion reads the SRI's fechaAutorizacion: its own offset form
// first, then RFC 3339 (the signer's SOAP client turns it into a JS Date, so it
// arrives as UTC "…Z", with or without milliseconds). Unparseable means now.
func parseFechaAutorizacion(raw string) time.Time {
	for _, layout := range []string{"2006-01-02T15:04:05.999-07:00", time.RFC3339Nano} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t
		}
	}
	return time.Now()
}
