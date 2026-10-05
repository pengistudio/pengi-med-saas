package pdfrender_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"pengi-med-saas/core/pdfrender"
	"pengi-med-saas/core/tenantfiles"
	billing_templates "pengi-med-saas/features/billing/templates"
	clinical_templates "pengi-med-saas/features/clinical/templates"
)

var catalog = []pdfrender.Document{
	clinical_templates.Prescription,
	clinical_templates.ExamOrder,
	clinical_templates.Report,
	clinical_templates.Certificate,
	billing_templates.InvoiceRide,
}

// Every printable document's default template must be embedded, pass the same
// validation a tenant upload does, and render its Sample. This catches drift
// between a data struct and its template.
func TestCatalog_DefaultTemplatesValidateAndRender(t *testing.T) {
	for _, doc := range catalog {
		t.Run(doc.ID, func(t *testing.T) {
			src, err := doc.DefaultSource()
			if err != nil {
				t.Fatalf("default not embedded: %v", err)
			}
			if err := pdfrender.Validate(doc, src); err != nil {
				t.Fatalf("default does not validate: %v", err)
			}
			conv := &fakeConverter{}
			pdf, err := pdfrender.New(tenantfiles.Memory(), conv).Preview(1, doc, nil)
			if err != nil || !bytes.HasPrefix(pdf, []byte("%PDF")) {
				t.Fatalf("preview: pdf=%q err=%v", pdf, err)
			}
			if conv.paper != doc.Paper {
				t.Fatalf("paper = %+v, want %+v", conv.paper, doc.Paper)
			}
			if strings.Contains(conv.html, "ZgotmplZ") {
				t.Fatalf("html/template rejected a value in the sample output")
			}
			if len(doc.Required) == 0 {
				t.Fatalf("document declares no required fields")
			}
		})
	}
}

func TestCatalog_IDsAndFilesAreUnique(t *testing.T) {
	ids, files := map[string]bool{}, map[string]bool{}
	for _, doc := range catalog {
		if ids[doc.ID] || files[doc.File] {
			t.Fatalf("duplicate document %s / %s", doc.ID, doc.File)
		}
		ids[doc.ID], files[doc.File] = true, true
		if doc.Feature != "clinical" && doc.Feature != "billing" {
			t.Fatalf("%s: feature %q is not an EnabledFeatures key", doc.ID, doc.Feature)
		}
	}
}

func TestValidate(t *testing.T) {
	report := clinical_templates.Report
	signed := `{{if .Signature}}<img src="{{.Signature.QR}}"> {{.Signature.Name}}{{end}}`

	cases := []struct {
		name   string
		src    string
		rule   pdfrender.Rule
		detail string
	}{
		{"parse error", `<p>{{.PatientName</p>`, pdfrender.RuleParse, "medical_report_template.html:1"},
		{"missing field", `<p>{{.Pacient}}</p>` + signed, pdfrender.RuleExecute, ".Pacient"},
		{"field missing only when unsigned", `<p>{{.Signature.Name}}</p><img src="{{.Signature.QR}}">`, pdfrender.RuleExecute, ".Signature.Name"},
		{"no signature", `<p>{{.PatientName}}</p>`, pdfrender.RuleRequired, ".Signature.QR"},
		{"no signer name", `<img src="{{if .Signature}}{{.Signature.QR}}{{end}}">`, pdfrender.RuleRequired, ".Signature.Name"},
		{"external img", `<img src="https://cdn.example.com/logo.png">` + signed, pdfrender.RuleExternalURL, "https://cdn.example.com/logo.png"},
		{"protocol-relative", `<script src='//evil.example/x.js'></script>` + signed, pdfrender.RuleExternalURL, "//evil.example/x.js"},
		{"css url", `<style>body{background:url( "http://x.test/bg.png")}</style>` + signed, pdfrender.RuleExternalURL, "http://x.test/bg.png"},
		{"css import", `<style>@import 'https://fonts.test/a.css';</style>` + signed, pdfrender.RuleExternalURL, "https://fonts.test/a.css"},
		{"link href", `<LINK HREF="https://fonts.test/a.css" rel="stylesheet">` + signed, pdfrender.RuleExternalURL, "https://fonts.test/a.css"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := pdfrender.Validate(report, []byte(tc.src))
			var verr *pdfrender.ValidationError
			if !errors.As(err, &verr) {
				t.Fatalf("err = %v, want a ValidationError", err)
			}
			if verr.Rule != tc.rule || !strings.Contains(verr.Detail, tc.detail) {
				t.Fatalf("rule=%s detail=%q, want %s containing %q", verr.Rule, verr.Detail, tc.rule, tc.detail)
			}
		})
	}

	t.Run("accepts a minimal valid template with data: images", func(t *testing.T) {
		src := `<img src="data:image/png;base64,iVBORw0KGgo="><p>{{.PatientName}}</p>` + signed
		if err := pdfrender.Validate(report, []byte(src)); err != nil {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("RIDE requires the access key", func(t *testing.T) {
		err := pdfrender.Validate(billing_templates.InvoiceRide, []byte(`<p>{{.TradeName}}</p>`))
		var verr *pdfrender.ValidationError
		if !errors.As(err, &verr) || verr.Rule != pdfrender.RuleRequired || verr.Detail != ".AccessKey" {
			t.Fatalf("err = %v, want missing .AccessKey", err)
		}
	})
}
