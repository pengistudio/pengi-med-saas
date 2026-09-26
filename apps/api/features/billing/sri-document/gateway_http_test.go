package sri_document

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// signerStub mimics the sri-xml-signer service: each route answers with the
// given status code and JSON body.
func signerStub(t *testing.T, routes map[string]stubResponse) *HTTPGateway {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp, ok := routes[r.URL.Path]
		if !ok {
			t.Fatalf("unexpected request to %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.code)
		_ = json.NewEncoder(w).Encode(resp.body)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("SRI_SIGNER_SERVICE_URL", srv.URL)
	return NewHTTPGateway()
}

type stubResponse struct {
	code int
	body map[string]any
}

func unreachableGateway(t *testing.T) *HTTPGateway {
	t.Helper()
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close() // nothing listens on this address any more
	t.Setenv("SRI_SIGNER_SERVICE_URL", srv.URL)
	return NewHTTPGateway()
}

func TestHTTPGateway_Sign(t *testing.T) {
	g := signerStub(t, map[string]stubResponse{
		"/sign": {200, map[string]any{"data": map[string]any{"xml": "<signed/>"}}},
	})
	signed, err := g.Sign([]byte("p12"), "pw", "<factura/>")
	if err != nil || signed != "<signed/>" {
		t.Fatalf("signed=%q err=%v", signed, err)
	}
}

func TestHTTPGateway_SignRejectedCertificateIsNotUnreachable(t *testing.T) {
	g := signerStub(t, map[string]stubResponse{
		"/sign": {400, map[string]any{"message": "Contraseña del certificado incorrecta"}},
	})
	_, err := g.Sign([]byte("p12"), "bad", "<factura/>")
	if err == nil || errors.Is(err, ErrUnreachable) {
		t.Fatalf("err = %v, want a non-unreachable error", err)
	}
}

func TestHTTPGateway_TransportFailuresAreUnreachable(t *testing.T) {
	g := unreachableGateway(t)
	if _, err := g.Sign(nil, "", ""); !errors.Is(err, ErrUnreachable) {
		t.Fatalf("sign err = %v, want ErrUnreachable", err)
	}
	if err := g.SubmitForReception("<signed/>", "1"); !errors.Is(err, ErrUnreachable) {
		t.Fatalf("reception err = %v, want ErrUnreachable", err)
	}
	if _, err := g.QueryAuthorization("key", "1"); !errors.Is(err, ErrUnreachable) {
		t.Fatalf("authorization err = %v, want ErrUnreachable", err)
	}
}

func TestHTTPGateway_SubmitForReception(t *testing.T) {
	cases := map[string]struct {
		resp          stubResponse
		wantRejection bool
		wantUnreach   bool
	}{
		"recibida": {resp: stubResponse{200, map[string]any{"data": map[string]any{"status": "RECIBIDA"}}}},
		"clave ya registrada (ID 43)": {resp: stubResponse{422, map[string]any{
			"message": "[SRI-ERROR] CLAVE ACCESO REGISTRADA (ID 43) - La clave de acceso ya se encuentra registrada"}}},
		"secuencial ya registrado (ID 45)": {resp: stubResponse{422, map[string]any{
			"message": "[SRI-ERROR] ERROR SECUENCIAL REGISTRADO (ID 45) - Sin información adicional"}}},
		"devuelta": {resp: stubResponse{422, map[string]any{
			"message": "[SRI-ERROR] ARCHIVO NO CUMPLE ESTRUCTURA XML (ID 35) - detalle"}}, wantRejection: true},
		"soap caído": {resp: stubResponse{422, map[string]any{
			"message": "Error SOAP al validar comprobante: connect ETIMEDOUT"}}, wantUnreach: true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			g := signerStub(t, map[string]stubResponse{"/validate/test": tc.resp})
			err := g.SubmitForReception("<signed/>", "1")

			var rejection *Rejection
			switch {
			case tc.wantRejection:
				if !errors.As(err, &rejection) {
					t.Fatalf("err = %v, want *Rejection", err)
				}
			case tc.wantUnreach:
				if !errors.Is(err, ErrUnreachable) {
					t.Fatalf("err = %v, want ErrUnreachable", err)
				}
			default:
				if err != nil {
					t.Fatalf("err = %v, want nil (received)", err)
				}
			}
		})
	}
}

func TestHTTPGateway_SubmitForReceptionUsesProductionRoute(t *testing.T) {
	g := signerStub(t, map[string]stubResponse{
		"/validate/prod": {200, map[string]any{"data": map[string]any{"status": "RECIBIDA"}}},
	})
	if err := g.SubmitForReception("<signed/>", "2"); err != nil {
		t.Fatalf("err = %v", err)
	}
}

func TestHTTPGateway_QueryAuthorization(t *testing.T) {
	cases := map[string]struct {
		resp        stubResponse
		want        AuthorizationStatus
		wantUnreach bool
	}{
		"autorizado": {resp: stubResponse{200, map[string]any{"data": map[string]any{
			"estadoAutorizacion": "AUTORIZADO", "fechaAutorizacion": "2026-09-01T10:00:00.000-05:00"}}}, want: Authorized},
		"en procesamiento": {resp: stubResponse{422, map[string]any{
			"message": "Estado del comprobante: EN PROCESAMIENTO"}}, want: AuthorizationPending},
		"aún sin autorización": {resp: stubResponse{422, map[string]any{
			"message": "No se encontró <autorizacion> en la respuesta SOAP."}}, want: AuthorizationPending},
		"sin información de autorización": {resp: stubResponse{422, map[string]any{
			"message": "No se recibió información de autorización del SRI"}}, want: AuthorizationPending},
		"no autorizado con mensaje": {resp: stubResponse{422, map[string]any{
			"message": "[SRI-ERROR] FIRMA INVALIDA (ID 39) - La firma es invalida"}}, want: NotAuthorized},
		"no autorizado sin mensaje": {resp: stubResponse{422, map[string]any{
			"message": "Estado del comprobante: NO AUTORIZADO"}}, want: NotAuthorized},
		"soap caído": {resp: stubResponse{422, map[string]any{
			"message": "socket hang up"}}, wantUnreach: true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			g := signerStub(t, map[string]stubResponse{"/authorization/test": tc.resp})
			auth, err := g.QueryAuthorization("key", "1")
			if tc.wantUnreach {
				if !errors.Is(err, ErrUnreachable) {
					t.Fatalf("err = %v, want ErrUnreachable", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			if auth.Status != tc.want {
				t.Fatalf("status = %v, want %v", auth.Status, tc.want)
			}
			if tc.want == NotAuthorized && auth.Message == "" {
				t.Fatalf("missing SRI reason")
			}
		})
	}
}

func TestHTTPGateway_QueryAuthorizationParsesFechaAutorizacion(t *testing.T) {
	g := signerStub(t, map[string]stubResponse{
		"/authorization/test": {200, map[string]any{"data": map[string]any{
			"estadoAutorizacion": "AUTORIZADO", "fechaAutorizacion": "2026-09-01T10:00:00.000-05:00"}}},
	})
	auth, err := g.QueryAuthorization("key", "1")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if want := time.Date(2026, 9, 1, 15, 0, 0, 0, time.UTC); !auth.AuthorizedAt.Equal(want) {
		t.Fatalf("authorized_at = %v, want %v", auth.AuthorizedAt, want)
	}
}
