package reports_handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/pdfrender"
	"pengi-med-saas/core/tenantdb"
	doctor_models "pengi-med-saas/features/doctors/models"
	reports_templates "pengi-med-saas/features/reports/templates"
	tenant_models "pengi-med-saas/features/tenants/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// translate maps an i18n key to the request's language.
type translate func(key string) string

// translator is the request's translator (set by the i18n middleware); without
// one keys come back as they are.
func translator(c *gin.Context) translate {
	if v, ok := c.Get("translator"); ok {
		if fn, ok := v.(func(string) string); ok {
			return fn
		}
	}
	return func(key string) string { return key }
}

// section is one table of a CSV export: a title row, headers, then rows.
type section struct {
	title   string
	headers []string
	rows    [][]string
}

// exporter says how one report becomes a CSV and a PDF.
type exporter[T any] struct {
	name     string // file name stem
	doc      pdfrender.Document
	sections func(tr translate, r T) []section
	pdfData  func(head reports_templates.Header, r T) any
}

func money(v float64) string { return strconv.FormatFloat(round2(v), 'f', 2, 64) }

func (h *ReportsHandler) writeCSV(c *gin.Context, name string, sections []section) {
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s.csv", name))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Status(http.StatusOK)
	// UTF-8 BOM so Excel opens accents correctly.
	_, _ = c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})
	w := csv.NewWriter(c.Writer)
	for i, s := range sections {
		if i > 0 {
			_ = w.Write([]string{})
		}
		_ = w.Write([]string{s.title})
		_ = w.Write(s.headers)
		_ = w.WriteAll(s.rows)
	}
	w.Flush()
}

func writePDF[T any](h *ReportsHandler, c *gin.Context, out exporter[T], tenantID uint, f filter, r T) {
	db := tenantdb.For(c, h.db)
	var tenant tenant_models.Tenant
	if err := db.First(&tenant, tenantID).Error; err != nil {
		h.internal(c, "failed to fetch tenant for report", err)
		return
	}
	head := reports_templates.Header{
		ClinicName:  tenant.DisplayName(),
		TaxID:       tenant.TaxID,
		From:        f.from.Format("02/01/2006"),
		To:          f.to.Format("02/01/2006"),
		GeneratedAt: time.Now().In(location).Format("02/01/2006 15:04"),
	}
	if f.doctorID != nil {
		var doctor doctor_models.Doctor
		if err := db.First(&doctor, *f.doctorID).Error; err == nil {
			head.Doctor = doctor.FullName
		}
	}
	pdf, err := h.renderer.Render(tenantID, out.doc, out.pdfData(head, r))
	if err != nil {
		h.logger.Error("failed to render report", zap.String("report", out.name), zap.Error(err))
		envelope.Write(c, envelope.ErrorResponse(http.StatusInternalServerError, "reports.error.render_failed", core_errors.ErrReportsRenderFailed))
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s_%s_%s.pdf", out.name, f.from.Format("20060102"), f.to.Format("20060102")))
	c.Data(http.StatusOK, "application/pdf", pdf)
}
