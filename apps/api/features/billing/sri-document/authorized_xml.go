package sri_document

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"
)

// ErrAuthorizedXMLUnavailable: the authorized XML of a document is not stored
// and the SRI did not answer with its authorization when asked again.
var ErrAuthorizedXMLUnavailable = errors.New("authorized XML unavailable")

// authorizedXMLName is the tenant file holding a document's authorized XML (the
// SRI's <autorizacion> wrapping the signed comprobante), next to its signed XML.
func authorizedXMLName(kind Kind, accessKey string) string {
	return kind.Folder + "/" + accessKey + ".autorizado.xml"
}

// AuthorizedXMLName is the tenant file holding a document's authorized XML.
func AuthorizedXMLName(kind Kind, accessKey string) string {
	return authorizedXMLName(kind, accessKey)
}

// AuthorizedXML returns the authorized XML of a document the SRI authorized.
// Documents authorized before it was stored have none: it is rebuilt from a new
// authorization query (the SRI answers it for any past access key) and stored.
// The query only reads the SRI's decision: the document is never resent and its
// status and files are left as they are (ADR 0001). The caller must have
// checked that the document is authorized and belongs to tenantID.
func (l *Lifecycle) AuthorizedXML(kind Kind, tenantID uint, accessKey string) ([]byte, error) {
	name := authorizedXMLName(kind, accessKey)
	if data, err := l.docs.Files.Read(tenantID, name); err == nil {
		return data, nil
	}
	// The comprobante is our stored signed XML; without it there is nothing to
	// wrap, so the SRI is not even asked.
	signed, err := l.docs.Files.Read(tenantID, signedXMLName(kind, accessKey))
	if err != nil {
		return nil, fmt.Errorf("%w: read signed XML: %w", ErrAuthorizedXMLUnavailable, err)
	}
	if l.gateway == nil {
		return nil, ErrAuthorizedXMLUnavailable
	}
	// The key says which ambiente issued it: a document from pruebas is
	// queried there even after the tenant moved to producción.
	sriEnv := accessKeyEnv(accessKey, l.sriEnv)
	auth, err := l.gateway.QueryAuthorization(accessKey, sriEnv)
	if err != nil {
		return nil, fmt.Errorf("query SRI authorization: %w", err)
	}
	if auth.Status != Authorized {
		return nil, fmt.Errorf("%w: SRI answered status %d for %s", ErrAuthorizedXMLUnavailable, auth.Status, accessKey)
	}
	data, err := buildAuthorizedXML(auth, string(signed), accessKey, sriEnv)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAuthorizedXMLUnavailable, err)
	}
	if err := l.docs.Files.Write(tenantID, name, data); err != nil {
		// Served anyway; the next download queries the SRI again.
		l.logger.Error("failed to store backfilled authorized XML",
			zap.String("kind", kind.Name), zap.Uint("tenant_id", tenantID), zap.String("access_key", accessKey), zap.Error(err))
	}
	return data, nil
}

// storeAuthorizedXML builds and stores the authorized XML of a document the SRI
// just authorized.
func (l *Lifecycle) storeAuthorizedXML(kind Kind, tenantID uint, accessKey string, auth Authorization) error {
	signed, err := l.docs.Files.Read(tenantID, signedXMLName(kind, accessKey))
	if err != nil {
		return fmt.Errorf("read signed XML: %w", err)
	}
	data, err := buildAuthorizedXML(auth, string(signed), accessKey, l.sriEnv)
	if err != nil {
		return err
	}
	return l.docs.Files.Write(tenantID, authorizedXMLName(kind, accessKey), data)
}

// accessKeyEnv is the ambiente code ("1" pruebas, "2" producción) in digit 24
// of a 49-digit access key, or fallback when the key does not carry one.
func accessKeyEnv(accessKey, fallback string) string {
	if len(accessKey) == 49 && (accessKey[23] == '1' || accessKey[23] == '2') {
		return string(accessKey[23])
	}
	return fallback
}

// buildAuthorizedXML renders the SRI's authorized XML: the <autorizacion>
// envelope of the SRI's answer wrapping signedXML, the comprobante exactly as
// we sent it and the SRI authorized it, inside a CDATA section. (The signer
// cannot give back the SRI's own copy byte for byte: its SOAP client parses a
// CDATA comprobante into an object.) Missing number and ambiente fall back to
// the access key (offline scheme: they are equal) and the ambiente of sriEnv.
func buildAuthorizedXML(auth Authorization, signedXML, accessKey, sriEnv string) ([]byte, error) {
	if auth.Status != Authorized {
		return nil, errors.New("authorization is not AUTORIZADO")
	}
	if strings.TrimSpace(signedXML) == "" {
		return nil, errors.New("no signed comprobante to wrap")
	}
	// The signer's field for the number is misnamed (rucEmisor); only trust it
	// when it looks like an authorization number (49 digits), else use the key.
	number := auth.Number
	if len(number) != 49 || strings.Trim(number, "0123456789") != "" {
		number = accessKey
	}
	environment := auth.Environment
	if environment == "" {
		environment = "PRUEBAS"
		if sriEnv == "2" {
			environment = "PRODUCCIÓN"
		}
	}

	var b bytes.Buffer
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString("<autorizacion>")
	element(&b, "estado", "AUTORIZADO")
	element(&b, "numeroAutorizacion", number)
	b.WriteString(`<fechaAutorizacion class="fechaAutorizacion">`)
	escape(&b, auth.RawAuthorizedAt)
	b.WriteString("</fechaAutorizacion>")
	element(&b, "ambiente", environment)
	b.WriteString("<comprobante><![CDATA[")
	// "]]>" cannot appear inside a CDATA section: close it after "]]" and
	// reopen it before ">", so the parsed text is unchanged.
	b.WriteString(strings.ReplaceAll(signedXML, "]]>", "]]]]><![CDATA[>"))
	b.WriteString("]]></comprobante>")
	b.WriteString("<mensajes>")
	for _, m := range auth.Messages {
		b.WriteString("<mensaje>")
		element(&b, "identificador", m.Identifier)
		element(&b, "mensaje", m.Message)
		if m.AdditionalInfo != "" {
			element(&b, "informacionAdicional", m.AdditionalInfo)
		}
		element(&b, "tipo", m.Type)
		b.WriteString("</mensaje>")
	}
	b.WriteString("</mensajes>")
	b.WriteString("</autorizacion>\n")
	return b.Bytes(), nil
}

func element(b *bytes.Buffer, name, text string) {
	b.WriteString("<" + name + ">")
	escape(b, text)
	b.WriteString("</" + name + ">")
}

func escape(b *bytes.Buffer, text string) {
	_ = xml.EscapeText(b, []byte(text))
}
