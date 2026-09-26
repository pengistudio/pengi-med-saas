// Package pdfrender turns a named HTML template plus data into a PDF for a
// tenant. It owns the rule "the tenant's own template if they uploaded one,
// otherwise the default embedded in the binary", HTML escaping, and the HTML to
// PDF conversion (Gotenberg), so no feature re-implements them.
package pdfrender

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"os"

	"pengi-med-saas/core/tenantfiles"
	"pengi-med-saas/core/utils"
)

// ErrTemplateNotFound: neither the tenant nor the defaults have the template.
var ErrTemplateNotFound = errors.New("pdfrender: template not found")

// Converter turns HTML into a PDF of the given paper format. *utils.GotenbergClient
// is the production adapter.
type Converter interface {
	GeneratePDFFromHTMLWithOptions(html string, opts utils.PDFOptions) ([]byte, error)
}

type Renderer struct {
	files     tenantfiles.Store
	converter Converter
	defaults  []fs.FS
}

// New builds a Renderer. A tenant's own template is looked up in files under
// the template's name; defaults are searched in order.
func New(files tenantfiles.Store, converter Converter, defaults ...fs.FS) *Renderer {
	return &Renderer{files: files, converter: converter, defaults: defaults}
}

// Gotenberg is the production converter, at GOTENBERG_URL (default the compose
// service http://gotenberg:3000).
func Gotenberg() *utils.GotenbergClient {
	url := os.Getenv("GOTENBERG_URL")
	if url == "" {
		url = "http://gotenberg:3000"
	}
	return utils.NewGotenbergClient(url)
}

// Render executes the tenant's template (or the default) with data, HTML-escaped,
// and converts it to a PDF in the given paper format.
func (r *Renderer) Render(tenantID uint, templateName string, data any, paper utils.PDFOptions) ([]byte, error) {
	src, err := r.source(tenantID, templateName)
	if err != nil {
		return nil, err
	}
	tmpl, err := template.New(templateName).Parse(string(src))
	if err != nil {
		return nil, fmt.Errorf("pdfrender: parse %s: %w", templateName, err)
	}
	var html bytes.Buffer
	if err := tmpl.Execute(&html, data); err != nil {
		return nil, fmt.Errorf("pdfrender: render %s: %w", templateName, err)
	}
	pdf, err := r.converter.GeneratePDFFromHTMLWithOptions(html.String(), paper)
	if err != nil {
		return nil, fmt.Errorf("pdfrender: convert %s: %w", templateName, err)
	}
	return pdf, nil
}

func (r *Renderer) source(tenantID uint, templateName string) ([]byte, error) {
	if src, err := r.files.Read(tenantID, templateName); err == nil {
		return src, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("pdfrender: read tenant template %s: %w", templateName, err)
	}
	for _, fsys := range r.defaults {
		if src, err := fs.ReadFile(fsys, templateName); err == nil {
			return src, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrTemplateNotFound, templateName)
}
