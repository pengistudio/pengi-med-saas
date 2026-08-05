package services

import (
	"errors"
	"testing"
)

func TestWrapSriError_GenuineSriRejection_NotClassifiedAsConnectionError(t *testing.T) {
	err := wrapSriError("failed to authorize XML with SRI", errors.New("[SRI-ERROR] FIRMA INVALIDA (ID 39) - La firma es invalida"))

	if errors.Is(err, ErrSriConnection) {
		t.Errorf("expected a genuine SRI rejection ([SRI-ERROR] prefix) to NOT be classified as ErrSriConnection, got: %v", err)
	}
}

func TestWrapSriError_TimeoutOrNetworkError_ClassifiedAsConnectionError(t *testing.T) {
	cases := []string{
		`Post "http://sri-xml-signer:9000/validate/test": context deadline exceeded (Client.Timeout exceeded while awaiting headers)`,
		"Error SOAP al validar comprobante: connect ECONNREFUSED 190.152.216.11:443",
		"No se encontró <autorizacion> en la respuesta SOAP.",
	}
	for _, raw := range cases {
		err := wrapSriError("failed to validate XML with SRI", errors.New(raw))
		if !errors.Is(err, ErrSriConnection) {
			t.Errorf("expected %q to be classified as ErrSriConnection, got: %v", raw, err)
		}
	}
}
