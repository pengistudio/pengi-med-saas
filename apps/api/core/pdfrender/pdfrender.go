// Package pdfrender turns a printable document (CONTEXT.md: "Documento
// imprimible") plus its data into a PDF for a tenant. It owns the rule "the
// tenant's own template if they uploaded one, otherwise the default embedded in
// the binary", the validation a tenant template must pass before it is stored,
// HTML escaping, and the HTML to PDF conversion (Gotenberg), so no feature
// re-implements them.
//
// Each printable document is a Document value declared next to its default
// template (features/*/templates); callers only pass the Document and its data.
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

// ErrNoCustomTemplate: the tenant has not uploaded a template for the document.
// It wraps fs.ErrNotExist.
var ErrNoCustomTemplate = fmt.Errorf("pdfrender: no custom template: %w", fs.ErrNotExist)

// MaxTemplateSize is the largest template a tenant may upload (512 KB). Images
// go inline as data: URIs, so the limit leaves room for a logo.
const MaxTemplateSize = 512 << 10

// Document is one printable document: its default template, paper format, and
// the sample data a tenant template is validated and previewed with.
type Document struct {
	// ID is the document's slug in URLs (e.g. "medical_report").
	ID string
	// File is the default template's name in Defaults and also the name of the
	// tenant's own template in tenant files. Never rename it: uploaded
	// templates are stored under it.
	File string
	// Defaults holds the default template File (embedded in the binary).
	Defaults fs.FS
	// Paper is the PDF format; fixed per document.
	Paper utils.PDFOptions
	// Feature is the plan module the document belongs to, an EnabledFeatures
	// key ("clinical", "billing").
	Feature string
	// Sample is a realistic value of the document's data type, used to validate
	// and preview templates.
	Sample any
	// Variants are other shapes the data takes in production (e.g. a document
	// not signed yet); a template must also execute with each of them.
	Variants []any
	// Required are values from Sample a template's output must contain, e.g.
	// the signature QR or the RIDE access key.
	Required []RequiredField
}

// RequiredField is a value the rendered Sample must contain. Field names the
// template field (".Signature.QR") so the user knows what is missing.
type RequiredField struct {
	Field string
	Value string
}

// DefaultSource returns the document's default template.
func (d Document) DefaultSource() ([]byte, error) {
	if d.Defaults == nil {
		return nil, fmt.Errorf("%w: %s", ErrTemplateNotFound, d.File)
	}
	src, err := fs.ReadFile(d.Defaults, d.File)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrTemplateNotFound, d.File)
	}
	return src, nil
}

// Converter turns HTML into a PDF of the given paper format. *utils.GotenbergClient
// is the production adapter.
type Converter interface {
	GeneratePDFFromHTMLWithOptions(html string, opts utils.PDFOptions) ([]byte, error)
}

type Renderer struct {
	files     tenantfiles.Store
	converter Converter
	docs      []Document
}

// New builds a Renderer. docs is the catalog of documents tenants can
// customize (Documents, Document); Render accepts any Document.
func New(files tenantfiles.Store, converter Converter, docs ...Document) *Renderer {
	return &Renderer{files: files, converter: converter, docs: docs}
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

// Documents returns the catalog, in the order given to New.
func (r *Renderer) Documents() []Document {
	return append([]Document(nil), r.docs...)
}

// Document finds a catalog document by ID.
func (r *Renderer) Document(id string) (Document, bool) {
	for _, d := range r.docs {
		if d.ID == id {
			return d, true
		}
	}
	return Document{}, false
}

// Render executes the tenant's template for doc (or the default) with data,
// HTML-escaped, and converts it to a PDF in doc's paper format.
func (r *Renderer) Render(tenantID uint, doc Document, data any) ([]byte, error) {
	src, err := r.source(tenantID, doc)
	if err != nil {
		return nil, err
	}
	return r.renderSource(doc, src, data)
}

// Preview renders doc's Sample to a PDF. With src it renders that template,
// validated first and not stored; with nil src, the tenant's effective one.
func (r *Renderer) Preview(tenantID uint, doc Document, src []byte) ([]byte, error) {
	if src == nil {
		var err error
		if src, err = r.source(tenantID, doc); err != nil {
			return nil, err
		}
	} else if err := Validate(doc, src); err != nil {
		return nil, err
	}
	return r.renderSource(doc, src, doc.Sample)
}

// HasCustom reports whether the tenant uploaded its own template for doc.
func (r *Renderer) HasCustom(tenantID uint, doc Document) bool {
	return r.files.Exists(tenantID, doc.File)
}

// Custom returns the tenant's own template for doc, or ErrNoCustomTemplate.
func (r *Renderer) Custom(tenantID uint, doc Document) ([]byte, error) {
	src, err := r.files.Read(tenantID, doc.File)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, ErrNoCustomTemplate
	}
	return src, err
}

// Save validates src (see Validate) and stores it as the tenant's template for
// doc. Documents generated from then on use it.
func (r *Renderer) Save(tenantID uint, doc Document, src []byte) error {
	if err := Validate(doc, src); err != nil {
		return err
	}
	return r.files.Write(tenantID, doc.File, src)
}

// Reset removes the tenant's template for doc, so the default applies again.
func (r *Renderer) Reset(tenantID uint, doc Document) error {
	return r.files.Remove(tenantID, doc.File)
}

func (r *Renderer) renderSource(doc Document, src []byte, data any) ([]byte, error) {
	tmpl, err := template.New(doc.File).Parse(string(src))
	if err != nil {
		return nil, fmt.Errorf("pdfrender: parse %s: %w", doc.File, err)
	}
	var html bytes.Buffer
	if err := tmpl.Execute(&html, data); err != nil {
		return nil, fmt.Errorf("pdfrender: render %s: %w", doc.File, err)
	}
	pdf, err := r.converter.GeneratePDFFromHTMLWithOptions(html.String(), doc.Paper)
	if err != nil {
		return nil, fmt.Errorf("pdfrender: convert %s: %w", doc.File, err)
	}
	return pdf, nil
}

func (r *Renderer) source(tenantID uint, doc Document) ([]byte, error) {
	if src, err := r.files.Read(tenantID, doc.File); err == nil {
		return src, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("pdfrender: read tenant template %s: %w", doc.File, err)
	}
	return doc.DefaultSource()
}
