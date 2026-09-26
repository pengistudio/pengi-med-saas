package pdfrender_test

import (
	"errors"
	"html/template"
	"io/fs"
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

var defaults = fstest.MapFS{
	"receta.html": {Data: []byte(`<p>default {{.Patient}}</p>`)},
}

func TestRender_UsesTheDefaultTemplate(t *testing.T) {
	conv := &fakeConverter{}
	r := pdfrender.New(tenantfiles.Memory(), conv, defaults)

	pdf, err := r.Render(1, "receta.html", map[string]string{"Patient": "Ana"}, utils.A5Landscape)
	if err != nil || string(pdf) != "%PDF" {
		t.Fatalf("pdf=%q err=%v", pdf, err)
	}
	if conv.html != "<p>default Ana</p>" {
		t.Fatalf("html = %q", conv.html)
	}
	if conv.paper != utils.A5Landscape {
		t.Fatalf("paper = %+v, want A5 landscape", conv.paper)
	}
}

func TestRender_PrefersTheTenantsOwnTemplate(t *testing.T) {
	files := tenantfiles.Memory()
	_ = files.Write(1, "receta.html", []byte(`<p>propia {{.Patient}}</p>`))
	conv := &fakeConverter{}
	r := pdfrender.New(files, conv, defaults)

	if _, err := r.Render(1, "receta.html", map[string]string{"Patient": "Ana"}, utils.A4Portrait); err != nil {
		t.Fatalf("render: %v", err)
	}
	if conv.html != "<p>propia Ana</p>" {
		t.Fatalf("html = %q, want the tenant's template", conv.html)
	}

	if _, err := r.Render(2, "receta.html", map[string]string{"Patient": "Luis"}, utils.A4Portrait); err != nil {
		t.Fatalf("render: %v", err)
	}
	if conv.html != "<p>default Luis</p>" {
		t.Fatalf("html = %q, another tenant must get the default", conv.html)
	}
}

func TestRender_UnknownTemplate(t *testing.T) {
	r := pdfrender.New(tenantfiles.Memory(), &fakeConverter{}, defaults)
	if _, err := r.Render(1, "nope.html", nil, utils.A4Portrait); !errors.Is(err, pdfrender.ErrTemplateNotFound) {
		t.Fatalf("err = %v, want ErrTemplateNotFound", err)
	}
}

func TestRender_EscapesDataAsHTML(t *testing.T) {
	conv := &fakeConverter{}
	r := pdfrender.New(tenantfiles.Memory(), conv, defaults)
	_, _ = r.Render(1, "receta.html", map[string]string{"Patient": "<script>x</script>"}, utils.A4Portrait)
	if strings.Contains(conv.html, "<script>") {
		t.Fatalf("data was not HTML-escaped: %q", conv.html)
	}
}

func TestRender_ConverterFailureIsReturned(t *testing.T) {
	r := pdfrender.New(tenantfiles.Memory(), &fakeConverter{err: errors.New("gotenberg down")}, defaults)
	if _, err := r.Render(1, "receta.html", nil, utils.A4Portrait); err == nil {
		t.Fatalf("err = nil, want the converter's error")
	}
}

// Every default template the app ships must be embedded and parse: a template
// missing from the production image is what broke the RIDE for two months.
func TestShippedDefaultTemplatesAreEmbeddedAndParse(t *testing.T) {
	want := map[fs.FS][]string{
		clinical_templates.FS: {"prescription_template.html", "medical_report_template.html", "medical_certificate_template.html"},
		billing_templates.FS:  {"invoice_ride_template.html"},
	}
	for fsys, names := range want {
		for _, name := range names {
			src, err := fs.ReadFile(fsys, name)
			if err != nil {
				t.Fatalf("%s not embedded: %v", name, err)
			}
			if _, err := template.New(name).Parse(string(src)); err != nil {
				t.Fatalf("%s does not parse: %v", name, err)
			}
		}
	}
}
