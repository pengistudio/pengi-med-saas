package sri_document

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"pengi-med-saas/core/tenantfiles"
	billing_models "pengi-med-saas/features/billing/models"

	"go.uber.org/zap"
)

// signedComprobante is a stored signed comprobante: an XML declaration,
// attributes, entities and a "]]>" a naive CDATA would break on.
const signedComprobante = `<?xml version="1.0" encoding="UTF-8"?>
<factura id="comprobante" version="1.1.0"><infoTributaria><razonSocial>A &amp; B</razonSocial></infoTributaria><campoAdicional nombre="nota">x ]]> y</campoAdicional><ds:Signature xmlns:ds="http://www.w3.org/2000/09/xmldsig#">sig</ds:Signature></factura>`

func authorizedAnswer() Authorization {
	return Authorization{
		Status:          Authorized,
		AuthorizedAt:    time.Date(2026, 9, 1, 15, 0, 0, 0, time.UTC),
		Number:          "0109202601179001122300110010010000001231234567810",
		RawAuthorizedAt: "2026-09-01T10:00:00-05:00",
		Environment:     "PRUEBAS",
	}
}

// parsedAuthorization is the <autorizacion> envelope as a reader parses it.
type parsedAuthorization struct {
	XMLName            xml.Name `xml:"autorizacion"`
	Estado             string   `xml:"estado"`
	NumeroAutorizacion string   `xml:"numeroAutorizacion"`
	Fecha              struct {
		Class string `xml:"class,attr"`
		Text  string `xml:",chardata"`
	} `xml:"fechaAutorizacion"`
	Ambiente    string `xml:"ambiente"`
	Comprobante string `xml:"comprobante"`
	Mensajes    []struct {
		Identificador        string `xml:"identificador"`
		Mensaje              string `xml:"mensaje"`
		InformacionAdicional string `xml:"informacionAdicional"`
		Tipo                 string `xml:"tipo"`
	} `xml:"mensajes>mensaje"`
}

// parseAuthorized parses data strictly, failing on anything not well-formed.
func parseAuthorized(t *testing.T, data []byte) parsedAuthorization {
	t.Helper()
	dec := xml.NewDecoder(bytes.NewReader(data))
	for {
		if _, err := dec.Token(); err == io.EOF {
			break
		} else if err != nil {
			t.Fatalf("authorized XML is not well-formed: %v\n%s", err, data)
		}
	}
	var got parsedAuthorization
	if err := xml.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal authorized XML: %v", err)
	}
	return got
}

// ─── buildAuthorizedXML ──────────────────────────────────────────────────────

func TestBuildAuthorizedXML_Envelope(t *testing.T) {
	auth := authorizedAnswer()
	auth.Environment = "PRUEBAS <&>"
	auth.Messages = []AuthorizationMessage{
		{Identifier: "60", Message: "ESTE PROCESO FUE REALIZADO EN EL AMBIENTE DE PRUEBAS", Type: "INFORMATIVO"},
		{Identifier: "70", Message: "A < B & C", AdditionalInfo: `"quoted"`, Type: "ADVERTENCIA"},
	}
	data, err := buildAuthorizedXML(auth, signedComprobante, "key", "1")
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if !bytes.HasPrefix(data, []byte(`<?xml version="1.0" encoding="UTF-8"?>`)) {
		t.Errorf("missing XML declaration: %.60s", data)
	}
	if !bytes.Contains(data, []byte(`<fechaAutorizacion class="fechaAutorizacion">2026-09-01T10:00:00-05:00</fechaAutorizacion>`)) {
		t.Errorf("fechaAutorizacion not in the SRI shape: %s", data)
	}
	if !bytes.Contains(data, []byte("<ambiente>PRUEBAS &lt;&amp;&gt;</ambiente>")) {
		t.Errorf("ambiente not escaped: %s", data)
	}

	got := parseAuthorized(t, data)
	if got.Estado != "AUTORIZADO" || got.NumeroAutorizacion != auth.Number || got.Fecha.Class != "fechaAutorizacion" ||
		got.Fecha.Text != auth.RawAuthorizedAt || got.Ambiente != "PRUEBAS <&>" {
		t.Errorf("envelope fields = %+v", got)
	}
	if got.Comprobante != signedComprobante {
		t.Errorf("comprobante is not the signed XML byte for byte:\n got %q\nwant %q", got.Comprobante, signedComprobante)
	}
	if len(got.Mensajes) != 2 || got.Mensajes[1].Mensaje != "A < B & C" || got.Mensajes[1].InformacionAdicional != `"quoted"` ||
		got.Mensajes[0].Identificador != "60" || got.Mensajes[0].Tipo != "INFORMATIVO" {
		t.Errorf("mensajes = %+v", got.Mensajes)
	}
}

func TestBuildAuthorizedXML_SplitsCDATATerminator(t *testing.T) {
	data, err := buildAuthorizedXML(authorizedAnswer(), signedComprobante, "key", "1")
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if !bytes.Contains(data, []byte("x ]]]]><![CDATA[> y")) {
		t.Errorf(`"]]>" not split across CDATA sections: %s`, data)
	}
	if got := parseAuthorized(t, data); got.Comprobante != signedComprobante {
		t.Errorf("comprobante changed by the split: %q", got.Comprobante)
	}
}

func TestBuildAuthorizedXML_DefaultsAndEmptyMensajes(t *testing.T) {
	auth := authorizedAnswer()
	auth.Number, auth.Environment = "", ""
	data, err := buildAuthorizedXML(auth, signedComprobante, "the-access-key", "2")
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	got := parseAuthorized(t, data)
	if got.NumeroAutorizacion != "the-access-key" || got.Ambiente != "PRODUCCIÓN" {
		t.Errorf("defaults: number %q ambiente %q", got.NumeroAutorizacion, got.Ambiente)
	}
	if !bytes.Contains(data, []byte("<mensajes></mensajes>")) || len(got.Mensajes) != 0 {
		t.Errorf("mensajes should be present and empty: %s", data)
	}
}

func TestBuildAuthorizedXML_RequiresAnAuthorizedComprobante(t *testing.T) {
	if _, err := buildAuthorizedXML(authorizedAnswer(), "  ", "key", "1"); err == nil {
		t.Error("built an envelope without a comprobante")
	}
	pending := authorizedAnswer()
	pending.Status = AuthorizationPending
	if _, err := buildAuthorizedXML(pending, signedComprobante, "key", "1"); err == nil {
		t.Error("built an envelope for a pending authorization")
	}
}

func TestAccessKeyEnv(t *testing.T) {
	key := func(env byte) string {
		k := []byte(strings.Repeat("0", 49))
		k[23] = env
		return string(k)
	}
	for _, tc := range []struct{ key, fallback, want string }{
		{key('1'), "2", "1"},
		{key('2'), "1", "2"},
		{key('9'), "1", "1"},
		{"short", "2", "2"},
	} {
		if got := accessKeyEnv(tc.key, tc.fallback); got != tc.want {
			t.Errorf("accessKeyEnv(%q, %q) = %q, want %q", tc.key, tc.fallback, got, tc.want)
		}
	}
}

// ─── On authorization ────────────────────────────────────────────────────────

func (f *fixture) authorizedXML(inv billing_models.Invoice) ([]byte, bool) {
	if inv.AccessKey == nil {
		return nil, false
	}
	data, err := f.files.Read(f.tenant.ID, "invoices/"+*inv.AccessKey+".autorizado.xml")
	return data, err == nil
}

func TestProcess_AuthorizedStoresAuthorizedXMLNextToSignedXML(t *testing.T) {
	f := newFixture(t)
	f.gateway.auth = authorizedAnswer()
	inv := f.invoice(billing_models.InvoiceStatusPending)
	if err := f.process(inv.ID); err != nil {
		t.Fatalf("process: %v", err)
	}
	got := f.reload(inv.ID)
	if got.Status != billing_models.InvoiceStatusAuthorized {
		t.Fatalf("status = %q", got.Status)
	}
	data, ok := f.authorizedXML(got)
	if !ok {
		t.Fatal("authorized XML not stored")
	}
	want, _ := buildAuthorizedXML(authorizedAnswer(), f.signedXML(got), *got.AccessKey, "1")
	if !bytes.Equal(data, want) {
		t.Errorf("stored authorized XML = %s\nwant %s", data, want)
	}
	if c := parseAuthorized(t, data).Comprobante; c != "<signed><factura>"+*got.AccessKey+"</factura></signed>" {
		t.Errorf("comprobante = %q, want the stored signed XML", c)
	}
	if f.signedXML(got) == "" {
		t.Error("signed XML was removed or overwritten")
	}
	if AuthorizedXMLName(f.kind, *got.AccessKey) != "invoices/"+*got.AccessKey+".autorizado.xml" {
		t.Errorf("AuthorizedXMLName = %q", AuthorizedXMLName(f.kind, *got.AccessKey))
	}
}

// The SRI wraps the comprobante in CDATA with an XML declaration, so the
// signer's SOAP client hands it back as an object: the authorized XML is still
// stored, built around our signed XML.
func TestProcess_CDATAComprobanteAnswerStillStoresAuthorizedXML(t *testing.T) {
	f := newFixture(t)
	gateway := signerStub(t, map[string]stubResponse{
		"/sign":          {200, map[string]any{"data": map[string]any{"xml": signedComprobante}}},
		"/validate/test": {200, map[string]any{"data": map[string]any{"status": "RECIBIDA"}}},
		"/authorization/test": {200, map[string]any{"data": map[string]any{
			"estadoAutorizacion": "AUTORIZADO", "fechaAutorizacion": "2026-09-01T15:00:00.000Z", "ambiente": "PRUEBAS",
			"comprobante": map[string]any{"factura": map[string]any{}}, "comprobanteCrudo": nil, "mensajes": nil}}},
	})
	f.docs = New(f.db, zap.NewNop(), gateway, f.publisher, Documents{Files: f.files}, "1")
	inv := f.invoice(billing_models.InvoiceStatusPending)
	if err := f.process(inv.ID); err != nil {
		t.Fatalf("process: %v", err)
	}
	got := f.reload(inv.ID)
	if got.Status != billing_models.InvoiceStatusAuthorized {
		t.Fatalf("status = %q, want authorized", got.Status)
	}
	if got.AuthorizedAt == nil || !got.AuthorizedAt.Equal(time.Date(2026, 9, 1, 15, 0, 0, 0, time.UTC)) {
		t.Errorf("authorized_at = %v, want the SRI's fecha", got.AuthorizedAt)
	}
	data, ok := f.authorizedXML(got)
	if !ok {
		t.Fatal("authorized XML not stored")
	}
	parsed := parseAuthorized(t, data)
	if parsed.Comprobante != signedComprobante || parsed.Fecha.Text != "2026-09-01T15:00:00.000Z" || parsed.NumeroAutorizacion != *got.AccessKey {
		t.Errorf("envelope = %+v", parsed)
	}
}

// failingAuthorizedStore fails only the authorized XML writes.
type failingAuthorizedStore struct{ tenantfiles.Store }

func (s failingAuthorizedStore) Write(tenantID uint, name string, data []byte) error {
	if strings.HasSuffix(name, ".autorizado.xml") {
		return errors.New("disk full")
	}
	return s.Store.Write(tenantID, name, data)
}

func TestProcess_AuthorizedXMLStoreFailureDoesNotFailTheDocument(t *testing.T) {
	f := newFixture(t)
	f.gateway.auth = authorizedAnswer()
	f.docs = New(f.db, zap.NewNop(), f.gateway, f.publisher, Documents{Files: failingAuthorizedStore{f.files}}, "1")
	inv := f.invoice(billing_models.InvoiceStatusPending)
	if err := f.process(inv.ID); err != nil {
		t.Fatalf("process: %v", err)
	}
	got := f.reload(inv.ID)
	if got.Status != billing_models.InvoiceStatusAuthorized || got.AuthorizedAt == nil {
		t.Fatalf("status = %q authorized_at = %v, want authorized", got.Status, got.AuthorizedAt)
	}
	if len(f.authorized) != 1 {
		t.Errorf("OnAuthorized calls = %v, want one", f.authorized)
	}
	if _, ok := f.authorizedXML(got); ok {
		t.Error("authorized XML stored despite the failing store")
	}
}

func TestProcess_NoAuthorizedXMLUnlessAuthorized(t *testing.T) {
	for name, auth := range map[string]Authorization{
		"pending":        {Status: AuthorizationPending},
		"not authorized": {Status: NotAuthorized, Message: "[SRI-ERROR] FIRMA INVALIDA (ID 39) - x"},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			f.gateway.auth = auth
			inv := f.invoice(billing_models.InvoiceStatusPending)
			_ = f.process(inv.ID)
			if _, ok := f.authorizedXML(f.reload(inv.ID)); ok {
				t.Error("authorized XML stored for a document the SRI did not authorize")
			}
		})
	}
}

// ─── AuthorizedXML (download) ────────────────────────────────────────────────

const testAccessKey = "0109202601179001122300110010010000001231234567810"

// storeSigned stores signedComprobante as the signed XML of accessKey.
func (f *fixture) storeSigned(accessKey string) {
	_ = f.files.Write(f.tenant.ID, signedXMLName(f.kind, accessKey), []byte(signedComprobante))
}

func TestAuthorizedXML_ServesStoredFileWithoutQueryingTheSri(t *testing.T) {
	f := newFixture(t)
	_ = f.files.Write(f.tenant.ID, AuthorizedXMLName(f.kind, testAccessKey), []byte("<autorizacion/>"))
	data, err := f.docs.AuthorizedXML(f.kind, f.tenant.ID, testAccessKey)
	if err != nil || string(data) != "<autorizacion/>" {
		t.Fatalf("data = %q err = %v", data, err)
	}
	if f.gateway.authCalls != 0 {
		t.Errorf("queried the SRI %d times for a stored file", f.gateway.authCalls)
	}
}

func TestAuthorizedXML_BackfillsFromTheSri(t *testing.T) {
	f := newFixture(t)
	f.gateway.auth = authorizedAnswer()
	inv := f.invoice(billing_models.InvoiceStatusAuthorized)
	f.storeSigned(testAccessKey)
	data, err := f.docs.AuthorizedXML(f.kind, f.tenant.ID, testAccessKey)
	if err != nil {
		t.Fatalf("AuthorizedXML: %v", err)
	}
	if got := parseAuthorized(t, data); got.Comprobante != signedComprobante {
		t.Errorf("backfilled comprobante = %q, want the stored signed XML", got.Comprobante)
	}
	if signed, _ := f.files.Read(f.tenant.ID, signedXMLName(f.kind, testAccessKey)); string(signed) != signedComprobante {
		t.Error("signed XML changed by the backfill")
	}
	stored, err := f.files.Read(f.tenant.ID, AuthorizedXMLName(f.kind, testAccessKey))
	if err != nil || !bytes.Equal(stored, data) {
		t.Errorf("backfill not stored: %v", err)
	}
	// Queried in the ambiente of the key (digit 24 = "1"), nothing resent or changed.
	if len(f.gateway.authEnvs) != 1 || f.gateway.authEnvs[0] != "1" || f.gateway.signCalls != 0 || f.gateway.receptionCalls != 0 {
		t.Errorf("auth envs %v, sign %d, reception %d", f.gateway.authEnvs, f.gateway.signCalls, f.gateway.receptionCalls)
	}
	if got := f.reload(inv.ID); got.Status != billing_models.InvoiceStatusAuthorized {
		t.Errorf("status changed to %q", got.Status)
	}

	// Served from the file from then on.
	if _, err := f.docs.AuthorizedXML(f.kind, f.tenant.ID, testAccessKey); err != nil || f.gateway.authCalls != 1 {
		t.Errorf("second call: err %v, SRI queries %d", err, f.gateway.authCalls)
	}
}

// A document issued in producción is queried there even if the lifecycle runs
// in pruebas (and the other way round).
func TestAuthorizedXML_BackfillQueriesTheAmbienteOfTheKey(t *testing.T) {
	f := newFixture(t)
	f.gateway.auth = authorizedAnswer()
	prodKey := []byte(testAccessKey)
	prodKey[23] = '2'
	f.storeSigned(string(prodKey))
	if _, err := f.docs.AuthorizedXML(f.kind, f.tenant.ID, string(prodKey)); err != nil {
		t.Fatalf("AuthorizedXML: %v", err)
	}
	if len(f.gateway.authEnvs) != 1 || f.gateway.authEnvs[0] != "2" {
		t.Errorf("queried ambientes %v, want [2] (lifecycle env is 1)", f.gateway.authEnvs)
	}
}

func TestAuthorizedXML_FailsWhenTheSriDoesNotAnswerAuthorized(t *testing.T) {
	cases := map[string]struct {
		auth    Authorization
		authErr error
		want    error
	}{
		"unreachable":    {authErr: ErrUnreachable, want: ErrUnreachable},
		"pending":        {auth: Authorization{Status: AuthorizationPending}, want: ErrAuthorizedXMLUnavailable},
		"not authorized": {auth: Authorization{Status: NotAuthorized, Message: "[SRI-ERROR] x (ID 1) - y"}, want: ErrAuthorizedXMLUnavailable},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			f.gateway.auth, f.gateway.authErr = tc.auth, tc.authErr
			inv := f.invoice(billing_models.InvoiceStatusAuthorized)
			f.storeSigned(testAccessKey)
			_, err := f.docs.AuthorizedXML(f.kind, f.tenant.ID, testAccessKey)
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
			if f.files.Exists(f.tenant.ID, AuthorizedXMLName(f.kind, testAccessKey)) {
				t.Error("stored an authorized XML")
			}
			if got := f.reload(inv.ID); got.Status != billing_models.InvoiceStatusAuthorized {
				t.Errorf("status changed to %q", got.Status)
			}
			if !f.files.Exists(f.tenant.ID, signedXMLName(f.kind, testAccessKey)) {
				t.Error("signed XML removed")
			}
		})
	}
}

// Without the signed XML there is no comprobante to wrap: nothing is stored
// under the authorized name and the SRI is not even asked.
func TestAuthorizedXML_SignedXMLMissing(t *testing.T) {
	f := newFixture(t)
	f.gateway.auth = authorizedAnswer()
	_, err := f.docs.AuthorizedXML(f.kind, f.tenant.ID, testAccessKey)
	if !errors.Is(err, ErrAuthorizedXMLUnavailable) {
		t.Fatalf("err = %v, want ErrAuthorizedXMLUnavailable", err)
	}
	if f.files.Exists(f.tenant.ID, AuthorizedXMLName(f.kind, testAccessKey)) || f.gateway.authCalls != 0 {
		t.Errorf("stored %v, SRI queries %d", f.files.Exists(f.tenant.ID, AuthorizedXMLName(f.kind, testAccessKey)), f.gateway.authCalls)
	}
}

func TestBuildAuthorizedXML_NumberFallsBackToAccessKeyWhenNotAnAuthorizationNumber(t *testing.T) {
	key := strings.Repeat("1", 49)
	out, err := buildAuthorizedXML(Authorization{Status: Authorized, Number: "1790012345001"}, "<f/>", key, "1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "<numeroAutorizacion>"+key+"</numeroAutorizacion>") {
		t.Fatalf("number should fall back to the access key, got %s", out)
	}
}
