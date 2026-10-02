package pdfrender_test

import (
	"errors"
	"strings"
	"testing"
	"testing/fstest"

	"pengi-med-saas/core/pdfrender"
	"pengi-med-saas/core/tenantfiles"
	"pengi-med-saas/core/utils"
	billing_templates "pengi-med-saas/features/billing/templates"
	clinical_templates "pengi-med-saas/features/clinical/templates"
)

// fakeConverter records the HTML and paper format it was asked to convert.
type fakeConverter struct {
	html  string
	paper utils.PDFOptions
	err   error
}

func (f *fakeConverter) GeneratePDFFromHTMLWithOptions(html string, opts utils.PDFOptions) ([]byte, error) {
	f.html, f.paper = html, opts
	return []byte("%PDF"), f.err
}

type recipeData struct{ Patient string }

var recipe = pdfrender.Document{
	ID:       "receta",
	File:     "receta.html",
	Defaults: fstest.MapFS{"receta.html": {Data: []byte(`<p>default {{.Patient}}</p>`)}},
	Paper:    utils.A5Landscape,
	Feature:  "clinical",
	Sample:   recipeData{Patient: "Ana Sample"},
	Required: []pdfrender.RequiredField{{Field: ".Patient", Value: "Ana Sample"}},
}

func TestRender_UsesTheDefaultTemplate(t *testing.T) {
	conv := &fakeConverter{}
	r := pdfrender.New(tenantfiles.Memory(), conv)

	pdf, err := r.Render(1, recipe, recipeData{Patient: "Ana"})
	if err != nil || string(pdf) != "%PDF" {
		t.Fatalf("pdf=%q err=%v", pdf, err)
	}
	if conv.html != "<p>default Ana</p>" {
		t.Fatalf("html = %q", conv.html)
	}
	if conv.paper != utils.A5Landscape {
		t.Fatalf("paper = %+v, want the document's A5 landscape", conv.paper)
	}
}

func TestRender_PrefersTheTenantsOwnTemplate(t *testing.T) {
	files := tenantfiles.Memory()
	_ = files.Write(1, "receta.html", []byte(`<p>propia {{.Patient}}</p>`))
	conv := &fakeConverter{}
	r := pdfrender.New(files, conv)

	if _, err := r.Render(1, recipe, recipeData{Patient: "Ana"}); err != nil {
		t.Fatalf("render: %v", err)
	}
	if conv.html != "<p>propia Ana</p>" {
		t.Fatalf("html = %q, want the tenant's template", conv.html)
	}

	if _, err := r.Render(2, recipe, recipeData{Patient: "Luis"}); err != nil {
		t.Fatalf("render: %v", err)
	}
	if conv.html != "<p>default Luis</p>" {
		t.Fatalf("html = %q, another tenant must get the default", conv.html)
	}
}

func TestRender_MissingDefault(t *testing.T) {
	r := pdfrender.New(tenantfiles.Memory(), &fakeConverter{})
	doc := recipe
	doc.File = "nope.html"
	if _, err := r.Render(1, doc, nil); !errors.Is(err, pdfrender.ErrTemplateNotFound) {
		t.Fatalf("err = %v, want ErrTemplateNotFound", err)
	}
}

func TestRender_EscapesDataAsHTML(t *testing.T) {
	conv := &fakeConverter{}
	r := pdfrender.New(tenantfiles.Memory(), conv)
	_, _ = r.Render(1, recipe, recipeData{Patient: "<script>x</script>"})
	if strings.Contains(conv.html, "<script>") {
		t.Fatalf("data was not HTML-escaped: %q", conv.html)
	}
}

func TestRender_ConverterFailureIsReturned(t *testing.T) {
	r := pdfrender.New(tenantfiles.Memory(), &fakeConverter{err: errors.New("gotenberg down")})
	if _, err := r.Render(1, recipe, recipeData{}); err == nil {
		t.Fatalf("err = nil, want the converter's error")
	}
}

func TestCatalog_LookupAndOrder(t *testing.T) {
	r := pdfrender.New(tenantfiles.Memory(), &fakeConverter{}, clinical_templates.Prescription, billing_templates.InvoiceRide)
	if docs := r.Documents(); len(docs) != 2 || docs[0].ID != "prescription" || docs[1].ID != "invoice_ride" {
		t.Fatalf("documents = %+v", docs)
	}
	if d, ok := r.Document("invoice_ride"); !ok || d.File != "invoice_ride_template.html" {
		t.Fatalf("lookup invoice_ride = %+v, %v", d, ok)
	}
	if _, ok := r.Document("nope"); ok {
		t.Fatalf("unknown document found")
	}
}

func TestSaveCustomReset(t *testing.T) {
	files := tenantfiles.Memory()
	conv := &fakeConverter{}
	r := pdfrender.New(files, conv, recipe)

	if r.HasCustom(1, recipe) {
		t.Fatalf("has custom before upload")
	}
	if _, err := r.Custom(1, recipe); !errors.Is(err, pdfrender.ErrNoCustomTemplate) {
		t.Fatalf("custom err = %v, want ErrNoCustomTemplate", err)
	}

	var verr *pdfrender.ValidationError
	if err := r.Save(1, recipe, []byte(`<p>{{.Nope}}</p>`)); !errors.As(err, &verr) {
		t.Fatalf("save invalid err = %v, want a ValidationError", err)
	}
	if r.HasCustom(1, recipe) {
		t.Fatalf("an invalid template was stored")
	}

	if err := r.Save(1, recipe, []byte(`<h1>{{.Patient}}</h1>`)); err != nil {
		t.Fatalf("save: %v", err)
	}
	if src, err := r.Custom(1, recipe); err != nil || string(src) != `<h1>{{.Patient}}</h1>` {
		t.Fatalf("custom = %q, %v", src, err)
	}
	if _, err := r.Preview(1, recipe, nil); err != nil || conv.html != "<h1>Ana Sample</h1>" {
		t.Fatalf("preview effective html=%q err=%v", conv.html, err)
	}
	if _, err := r.Preview(1, recipe, []byte(`<h2>{{.Patient}}</h2>`)); err != nil || conv.html != "<h2>Ana Sample</h2>" {
		t.Fatalf("preview upload html=%q err=%v", conv.html, err)
	}
	if string(mustCustom(t, r)) != `<h1>{{.Patient}}</h1>` {
		t.Fatalf("preview must not store the template")
	}

	if err := r.Reset(1, recipe); err != nil || r.HasCustom(1, recipe) {
		t.Fatalf("reset err=%v has=%v", err, r.HasCustom(1, recipe))
	}
}

func mustCustom(t *testing.T, r *pdfrender.Renderer) []byte {
	t.Helper()
	src, err := r.Custom(1, recipe)
	if err != nil {
		t.Fatalf("custom: %v", err)
	}
	return src
}
