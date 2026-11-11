package cash_handlers

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	cash_models "pengi-med-saas/features/cash/models"
	cash_templates "pengi-med-saas/features/cash/templates"
	tenant_models "pengi-med-saas/features/tenants/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// reportLocation is the clock the closing report prints (times are stored in
// UTC); Ecuador has a single time zone.
var reportLocation = func() *time.Location {
	if loc, err := time.LoadLocation("America/Guayaquil"); err == nil {
		return loc
	}
	return time.FixedZone("ECT", -5*60*60)
}()

// DownloadSessionReport serves the closing report PDF of a closed session,
// rendered with the tenant's template or the default (core/pdfrender).
func (h *CashHandler) DownloadSessionReport(c *gin.Context) {
	id, ok := paramID(c)
	if !ok {
		envelope.Write(c, invalid("cash.session.error.invalid_id"))
		return
	}
	db := tenantdb.For(c, h.db)
	var session cash_models.CashSession
	if err := db.First(&session, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			envelope.Write(c, envelope.ErrorResponse(http.StatusNotFound, "cash.session.error.not_found", core_errors.ErrCashSessionNotFound))
			return
		}
		envelope.Write(c, h.internal("failed to fetch cash session", err))
		return
	}
	if session.Status != cash_models.SessionClosed {
		envelope.Write(c, envelope.ErrorResponse(http.StatusConflict, "cash.session.error.not_closed", core_errors.ErrCashSessionNotClosed))
		return
	}
	detail, err := h.detail(db, session)
	if err != nil {
		envelope.Write(c, h.internal("failed to build cash session detail", err))
		return
	}
	var tenant tenant_models.Tenant
	if err := db.First(&tenant, tenantdb.TenantID(c)).Error; err != nil {
		envelope.Write(c, h.internal("failed to fetch tenant for cash report", err))
		return
	}

	pdf, err := h.renderer.Render(tenantdb.TenantID(c), cash_templates.SessionReport, reportData(tenant, detail, time.Now()))
	if err != nil {
		h.logger.Error("failed to render cash session report", zap.Uint("session_id", session.ID), zap.Error(err))
		envelope.Write(c, envelope.ErrorResponse(http.StatusInternalServerError, "cash.session.error.report_failed", core_errors.ErrCashInternal))
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=cierre_caja_%06d.pdf", session.ID))
	c.Data(http.StatusOK, "application/pdf", pdf)
}

func money(v float64) string { return fmt.Sprintf("%.2f", cash_models.Round2(v)) }

func moneyPtr(v *float64) string {
	if v == nil {
		return ""
	}
	return money(*v)
}

func localTime(t time.Time, layout string) string { return t.In(reportLocation).Format(layout) }

// reportData formats a closed session for cash_session_report_template.html.
func reportData(tenant tenant_models.Tenant, d SessionDetail, now time.Time) cash_templates.SessionReportData {
	data := cash_templates.SessionReportData{
		ClinicName:   tenant.DisplayName(),
		TaxID:        tenant.TaxID,
		SessionID:    fmt.Sprintf("%06d", d.ID),
		OpenedAt:     localTime(d.OpenedAt, "02/01/2006 15:04"),
		OpenedBy:     d.OpenedByName,
		ClosedBy:     d.ClosedByName,
		CloseNote:    d.CloseNote,
		OpeningFloat: money(d.OpeningFloat),
		CashIn:       money(d.Totals.CashIn),
		CashRefunded: money(d.Totals.CashRefunded),
		MovementsIn:  money(d.Totals.MovementsIn),
		MovementsOut: money(d.Totals.MovementsOut),
		ExpectedCash: moneyPtr(d.ExpectedCash),
		CountedCash:  moneyPtr(d.CountedCash),
		Difference:   moneyPtr(d.Difference),
		GeneratedAt:  localTime(now, "02/01/2006 15:04"),
	}
	if d.ClosedAt != nil {
		data.ClosedAt = localTime(*d.ClosedAt, "02/01/2006 15:04")
	}
	for _, m := range d.Totals.ByMethod {
		data.Methods = append(data.Methods, cash_templates.SessionReportMethod{
			Method: cash_templates.Label(cash_templates.MethodLabels, m.Method), Count: m.Count, Total: money(m.Total),
		})
	}
	for _, p := range d.Payments {
		data.Payments = append(data.Payments, cash_templates.SessionReportPayment{
			Time:      localTime(p.CreatedAt, "15:04"),
			Invoice:   p.InvoiceNumber,
			Method:    cash_templates.Label(cash_templates.MethodLabels, p.Method),
			Kind:      cash_templates.Label(cash_templates.KindLabels, p.Kind),
			Reference: p.Reference,
			Amount:    money(p.Amount),
			Voided:    p.Status == cash_models.PaymentVoided,
		})
	}
	for _, m := range d.Movements {
		data.Movements = append(data.Movements, cash_templates.SessionReportMovement{
			Time:     localTime(m.CreatedAt, "15:04"),
			Type:     cash_templates.Label(cash_templates.MovementTypeLabels, m.Type),
			Category: cash_templates.Label(cash_templates.CategoryLabels, m.Category),
			Reason:   m.Reason,
			Amount:   money(m.Amount),
		})
	}
	return data
}
