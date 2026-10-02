//go:build smoke

package pdfsign_test

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/digitorus/pdfsign/verify"

	"pengi-med-saas/core/pdfrender"
	"pengi-med-saas/core/pdfsign"
	"pengi-med-saas/core/pdfsign/pdfsigntest"
	"pengi-med-saas/core/tenantfiles"
	clinical_templates "pengi-med-saas/features/clinical/templates"
)

// Signs real Gotenberg output with the visible stamp:
// GOTENBERG_URL=http://localhost:8001 SMOKE_OUT=/tmp go test -tags smoke ./core/pdfsign/
func TestSmoke_SignsRealGotenbergPDFs(t *testing.T) {
	cert, err := pdfsign.Load(pdfsigntest.P12(t, "DRA ANA PEREZ", "clave", time.Now().Add(24*time.Hour)), "clave")
	if err != nil {
		t.Fatal(err)
	}
	opts := pdfsign.Options{Reason: "Smoke", Location: "Ecuador", SignedAt: time.Now()}
	r := pdfrender.New(tenantfiles.Memory(), pdfrender.Gotenberg())
	// The samples already carry a visible stamp; the PDF signature is independent of it.
	for _, doc := range []pdfrender.Document{clinical_templates.Report, clinical_templates.Certificate, clinical_templates.Prescription} {
		name := doc.File
		pdf, err := r.Preview(1, doc, nil)
		if err != nil {
			t.Fatalf("%s: render: %v", name, err)
		}
		signed, err := pdfsign.Sign(pdf, cert, opts)
		if err != nil {
			t.Fatalf("%s: sign: %v", name, err)
		}
		resp, err := verify.Verify(bytes.NewReader(signed), int64(len(signed)))
		if err != nil || len(resp.Signers) != 1 || !resp.Signers[0].ValidSignature {
			t.Fatalf("%s: does not verify: %v %+v", name, err, resp)
		}
		if out := os.Getenv("SMOKE_OUT"); out != "" {
			_ = os.WriteFile(out+"/signed_"+name+".pdf", signed, 0644)
		}
		t.Logf("%s -> %d bytes signed", name, len(signed))
	}
}
