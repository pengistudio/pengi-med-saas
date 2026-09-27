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
	"pengi-med-saas/core/utils"
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
	stamp, err := pdfsign.NewStamp(cert, opts)
	if err != nil {
		t.Fatal(err)
	}

	r := pdfrender.New(tenantfiles.Memory(), pdfrender.Gotenberg(), clinical_templates.FS)
	for name, paper := range map[string]utils.PDFOptions{
		"medical_report_template.html":      utils.A4Portrait,
		"medical_certificate_template.html": utils.A4Portrait,
		"prescription_template.html":        utils.A5Landscape,
	} {
		pdf, err := r.Render(1, name, map[string]any{"Signature": stamp, "DoctorName": "Ana Perez", "TradeName": "Clinica"}, paper)
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
