// Package document_template_handlers lets a tenant give each printable
// document (CONTEXT.md: "Documento imprimible") its own template: list them,
// download the default or the custom one, upload (validated), restore the
// default, and preview a template with sample data. The rules live in
// core/pdfrender; this is the HTTP surface.
package document_template_handlers

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/pdfrender"
	"pengi-med-saas/core/tenantdb"
	"pengi-med-saas/core/utils"
	company_models "pengi-med-saas/features/companies/models"
	company_services "pengi-med-saas/features/companies/services"
	tenant_models "pengi-med-saas/features/tenants/models"
)

type DocumentTemplateHandler struct {
	db       *gorm.DB
	logger   *zap.Logger
	renderer *pdfrender.Renderer
	// enabledFeatures resolves the plan modules of the request's tenant.
	enabledFeatures func(c *gin.Context) (tenant_models.EnabledFeatures, error)
}

func NewDocumentTemplateHandler(db *gorm.DB, logger *zap.Logger, renderer *pdfrender.Renderer) *DocumentTemplateHandler {
	h := &DocumentTemplateHandler{db: db, logger: logger, renderer: renderer}
	h.enabledFeatures = h.tenantEnabledFeatures
	return h
}

// DocumentTemplate is one printable document as Settings lists it.
type DocumentTemplate struct {
	ID        string `json:"id"`
	HasCustom bool   `json:"has_custom"`
	Paper     string `json:"paper"` // "a4_portrait" | "a5_landscape"
}

// ListDocumentTemplates returns the printable documents of the tenant's plan
// modules and whether each has a custom template.
func (h *DocumentTemplateHandler) ListDocumentTemplates(c *gin.Context) envelope.Response {
	features, err := h.enabledFeatures(c)
	if err != nil {
		h.logger.Error("Failed to resolve enabled features", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	tenantID := tenantdb.TenantID(c)
	out := []DocumentTemplate{}
	for _, doc := range h.renderer.Documents() {
		if !featureEnabled(features, doc.Feature) {
			continue
		}
		out = append(out, DocumentTemplate{ID: doc.ID, HasCustom: h.renderer.HasCustom(tenantID, doc), Paper: paperName(doc.Paper)})
	}
	return envelope.SuccessResponse(out, "document_templates.list.success")
}

// DownloadDefaultTemplate streams the document's default template, the
// starting point for a custom one.
func (h *DocumentTemplateHandler) DownloadDefaultTemplate(c *gin.Context) {
	doc, failure := h.document(c)
	if failure != nil {
		envelope.Write(c, *failure)
		return
	}
	src, err := doc.DefaultSource()
	if err != nil {
		h.logger.Error("Default template missing from the binary", zap.String("document", doc.ID), zap.Error(err))
		envelope.Write(c, envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal))
		return
	}
	sendHTML(c, doc.File, src)
}

// DownloadCustomTemplate streams the tenant's own template; 404 if none.
func (h *DocumentTemplateHandler) DownloadCustomTemplate(c *gin.Context) {
	doc, failure := h.document(c)
	if failure != nil {
		envelope.Write(c, *failure)
		return
	}
	src, err := h.renderer.Custom(tenantdb.TenantID(c), doc)
	if errors.Is(err, pdfrender.ErrNoCustomTemplate) {
		envelope.Write(c, envelope.ErrorResponse(http.StatusNotFound, "document_templates.error.no_custom", core_errors.ErrDocTemplateNoCustom))
		return
	}
	if err != nil {
		h.logger.Error("Failed to read custom template", zap.String("document", doc.ID), zap.Error(err))
		envelope.Write(c, envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal))
		return
	}
	sendHTML(c, "custom_"+doc.File, src)
}

// UploadDocumentTemplate validates the multipart field "template" and stores it
// as the tenant's template for the document. Documents generated from then on
// use it; already generated (or signed) ones don't change.
func (h *DocumentTemplateHandler) UploadDocumentTemplate(c *gin.Context) envelope.Response {
	doc, failure := h.document(c)
	if failure != nil {
		return *failure
	}
	src, failure := h.readTemplate(c, true)
	if failure != nil {
		return *failure
	}
	if err := h.renderer.Save(tenantdb.TenantID(c), doc, src); err != nil {
		return h.saveError(doc, err)
	}
	h.logger.Info("Document template uploaded", zap.Uint("tenant_id", tenantdb.TenantID(c)), zap.String("document", doc.ID))
	return envelope.SuccessResponse(DocumentTemplate{ID: doc.ID, HasCustom: true, Paper: paperName(doc.Paper)}, "document_templates.upload.success")
}

// DeleteDocumentTemplate removes the tenant's template, restoring the default.
func (h *DocumentTemplateHandler) DeleteDocumentTemplate(c *gin.Context) envelope.Response {
	doc, failure := h.document(c)
	if failure != nil {
		return *failure
	}
	if err := h.renderer.Reset(tenantdb.TenantID(c), doc); err != nil {
		h.logger.Error("Failed to remove custom template", zap.String("document", doc.ID), zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	return envelope.SuccessResponse(DocumentTemplate{ID: doc.ID, HasCustom: false, Paper: paperName(doc.Paper)}, "document_templates.delete.success")
}

// PreviewDocumentTemplate renders the document's sample data to a PDF: with a
// multipart "template" it uses that one (validated, not stored), otherwise the
// tenant's effective template.
func (h *DocumentTemplateHandler) PreviewDocumentTemplate(c *gin.Context) {
	doc, failure := h.document(c)
	if failure != nil {
		envelope.Write(c, *failure)
		return
	}
	src, failure := h.readTemplate(c, false)
	if failure != nil {
		envelope.Write(c, *failure)
		return
	}
	pdf, err := h.renderer.Preview(tenantdb.TenantID(c), doc, src)
	if err != nil {
		var verr *pdfrender.ValidationError
		if errors.As(err, &verr) {
			envelope.Write(c, validationResponse(verr))
			return
		}
		h.logger.Error("Failed to render template preview", zap.String("document", doc.ID), zap.Error(err))
		envelope.Write(c, envelope.ErrorResponse(http.StatusInternalServerError, "document_templates.error.preview", core_errors.ErrDocTemplatePreviewFailed))
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=preview_%s.pdf", doc.ID))
	c.Data(http.StatusOK, "application/pdf", pdf)
}

// document resolves :doc to a catalog document of an enabled plan module;
// anything else is 404.
func (h *DocumentTemplateHandler) document(c *gin.Context) (pdfrender.Document, *envelope.Response) {
	notFound := envelope.ErrorResponse(http.StatusNotFound, "document_templates.error.not_found", core_errors.ErrDocTemplateNotFound)
	doc, ok := h.renderer.Document(c.Param("doc"))
	if !ok {
		return doc, &notFound
	}
	features, err := h.enabledFeatures(c)
	if err != nil {
		h.logger.Error("Failed to resolve enabled features", zap.Error(err))
		resp := envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
		return doc, &resp
	}
	if !featureEnabled(features, doc.Feature) {
		return doc, &notFound
	}
	return doc, nil
}

// readTemplate reads the multipart file "template", at most
// pdfrender.MaxTemplateSize. When required is false, a request without it
// returns nil src.
func (h *DocumentTemplateHandler) readTemplate(c *gin.Context, required bool) ([]byte, *envelope.Response) {
	tooLarge := envelope.ErrorResponse(http.StatusRequestEntityTooLarge, "document_templates.error.too_large",
		core_errors.ErrDocTemplateTooLarge.WithDetail(fmt.Sprintf("%d KB", pdfrender.MaxTemplateSize>>10)))
	fileRequired := envelope.ErrorResponse(http.StatusBadRequest, "document_templates.error.file_required", core_errors.ErrDocTemplateFileRequired)

	// Room for the multipart envelope around the file.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, pdfrender.MaxTemplateSize+64<<10)
	file, header, err := c.Request.FormFile("template")
	if err != nil {
		var maxErr *http.MaxBytesError
		switch {
		case errors.As(err, &maxErr):
			return nil, &tooLarge
		case !required && (errors.Is(err, http.ErrMissingFile) || errors.Is(err, http.ErrNotMultipart)):
			return nil, nil
		default:
			return nil, &fileRequired
		}
	}
	defer file.Close()
	if header.Size > pdfrender.MaxTemplateSize {
		return nil, &tooLarge
	}
	src, err := io.ReadAll(io.LimitReader(file, pdfrender.MaxTemplateSize+1))
	if err != nil {
		h.logger.Error("Failed to read template upload", zap.Error(err))
		return nil, &fileRequired
	}
	if len(src) > pdfrender.MaxTemplateSize {
		return nil, &tooLarge
	}
	if len(src) == 0 {
		return nil, &fileRequired
	}
	return src, nil
}

func (h *DocumentTemplateHandler) saveError(doc pdfrender.Document, err error) envelope.Response {
	var verr *pdfrender.ValidationError
	if errors.As(err, &verr) {
		return validationResponse(verr)
	}
	h.logger.Error("Failed to save custom template", zap.String("document", doc.ID), zap.Error(err))
	return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
}

// validationResponse tells the user which rule the template broke, with the
// failing field or reference as the error's detail.
func validationResponse(verr *pdfrender.ValidationError) envelope.Response {
	key, code := "document_templates.error.parse", core_errors.ErrDocTemplateParse
	switch verr.Rule {
	case pdfrender.RuleExternalURL:
		key, code = "document_templates.error.external_url", core_errors.ErrDocTemplateExternalURL
	case pdfrender.RuleExecute:
		key, code = "document_templates.error.execute", core_errors.ErrDocTemplateExecute
	case pdfrender.RuleRequired:
		key, code = "document_templates.error.required", core_errors.ErrDocTemplateRequiredField
	}
	return envelope.ErrorResponse(http.StatusBadRequest, key, code.WithDetail(verr.Detail))
}

func (h *DocumentTemplateHandler) tenantEnabledFeatures(c *gin.Context) (tenant_models.EnabledFeatures, error) {
	var company company_models.Company
	if err := tenantdb.For(c, h.db).First(&company).Error; err != nil {
		return tenant_models.EnabledFeatures{}, err
	}
	return company_services.EnabledFeaturesForCompany(h.db, company.ID)
}

// featureEnabled maps a Document.Feature to its EnabledFeatures flag.
func featureEnabled(f tenant_models.EnabledFeatures, feature string) bool {
	switch feature {
	case "clinical":
		return f.Clinical
	case "billing":
		return f.Billing
	case "team":
		return f.Team
	case "kanban":
		return f.Kanban
	}
	return false
}

func paperName(p utils.PDFOptions) string {
	switch p {
	case utils.A5Landscape:
		return "a5_landscape"
	case utils.A4Portrait:
		return "a4_portrait"
	}
	return "custom"
}

func sendHTML(c *gin.Context, filename string, src []byte) {
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Data(http.StatusOK, "text/html; charset=utf-8", src)
}
