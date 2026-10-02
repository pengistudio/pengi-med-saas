//go:build smoke

package pdfrender_test

import (
	"bytes"
	"os"
	"testing"

	"pengi-med-saas/core/pdfrender"
	"pengi-med-saas/core/tenantfiles"
	billing_templates "pengi-med-saas/features/billing/templates"
	clinical_templates "pengi-med-saas/features/clinical/templates"
)

// Run against a real Gotenberg: GOTENBERG_URL=http://localhost:8001 go test -tags smoke ./core/pdfrender/
func TestSmoke_RealGotenbergRendersEveryDefaultTemplate(t *testing.T) {
	r := pdfrender.New(tenantfiles.Memory(), pdfrender.Gotenberg())
	for _, doc := range []pdfrender.Document{clinical_templates.Prescription, clinical_templates.Report, clinical_templates.Certificate, billing_templates.InvoiceRide} {
		name := doc.File
		pdf, err := r.Preview(1, doc, nil)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !bytes.HasPrefix(pdf, []byte("%PDF")) {
			t.Fatalf("%s: not a PDF", name)
		}
		t.Logf("%s -> %d bytes", name, len(pdf))
		_ = os.WriteFile(os.Getenv("SMOKE_OUT")+"/"+name+".pdf", pdf, 0644)
	}
}
