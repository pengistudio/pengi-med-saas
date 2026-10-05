package routes

import (
	"pengi-med-saas/core/pdfrender"
	"pengi-med-saas/core/tenantfiles"
	billing_templates "pengi-med-saas/features/billing/templates"
	clinical_templates "pengi-med-saas/features/clinical/templates"
)

// tenantFiles is where every tenant's files live (the api_storage volume).
var tenantFiles = tenantfiles.Disk(tenantfiles.DefaultRoot)

// printableDocuments is the catalog of printable documents tenants can give
// their own template, in the order Settings lists them.
var printableDocuments = []pdfrender.Document{
	clinical_templates.Prescription,
	clinical_templates.ExamOrder,
	clinical_templates.Report,
	clinical_templates.Certificate,
	billing_templates.InvoiceRide,
}

// documentRenderer renders tenant documents to PDF with the tenant's own
// template if uploaded, otherwise the defaults embedded in the binary.
func documentRenderer() *pdfrender.Renderer {
	return pdfrender.New(tenantFiles, pdfrender.Gotenberg(), printableDocuments...)
}
