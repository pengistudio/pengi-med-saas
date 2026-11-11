package reports_handlers

import (
	"time"

	cash_models "pengi-med-saas/features/cash/models"
	cash_services "pengi-med-saas/features/cash/services"
	reports_dto "pengi-med-saas/features/reports/dto"
	reports_templates "pengi-med-saas/features/reports/templates"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// openInvoicesQuery is the invoices issued in the range that still have a
// balance, with what was collected as pp.paid. Like cash_services.
// ApplyPaymentStatus, invoices from before the tenant's first cash session
// with no payment were charged outside the desk and are not receivable; a
// tenant that never opened a session has none.
func openInvoicesQuery(db *gorm.DB, tenantID uint, f filter) (*gorm.DB, bool, error) {
	firstAt, err := cash_services.FirstSessionAt(db)
	if err != nil || firstAt == nil {
		return nil, false, err
	}
	paid := "LEFT JOIN (SELECT invoice_id, SUM(amount) AS paid FROM cash_payments " +
		"WHERE tenant_id = ? AND status = ? AND deleted_at IS NULL GROUP BY invoice_id) pp ON pp.invoice_id = invoices.id"
	q := invoicesQuery(db, f).
		Joins(paid, tenantID, cash_models.PaymentValid).
		Where("invoices.total - COALESCE(pp.paid, 0) > 0.004").
		Where("(pp.paid IS NOT NULL OR invoices.created_at >= ?)", *firstAt)
	return q, true, nil
}

func (h *ReportsHandler) receivableTotal(db *gorm.DB, tenantID uint, f filter) (float64, error) {
	q, ok, err := openInvoicesQuery(db, tenantID, f)
	if err != nil || !ok {
		return 0, err
	}
	var total struct{ Balance float64 }
	if err := q.Select("COALESCE(SUM(invoices.total - COALESCE(pp.paid, 0)), 0) AS balance").Scan(&total).Error; err != nil {
		return 0, err
	}
	return round2(total.Balance), nil
}

const (
	bucket30   = "0_30"
	bucket60   = "31_60"
	bucket61up = "61_plus"
)

func (h *ReportsHandler) buildReceivables(db *gorm.DB, tenantID uint, f filter) (reports_dto.ReceivablesReport, error) {
	report := reports_dto.ReceivablesReport{
		Filters: f.dto(),
		Buckets: []reports_dto.ReceivableBucket{{Key: bucket30}, {Key: bucket60}, {Key: bucket61up}},
		Items:   []reports_dto.ReceivableItem{},
	}
	q, ok, err := openInvoicesQuery(db, tenantID, f)
	if err != nil || !ok {
		return report, err
	}
	var rows []struct {
		ID                uint
		EstablishmentCode string
		EmissionPointCode string
		Sequential        string
		IssueDay          string
		Total             float64
		Paid              float64
		PatientID         *uint
		PatientName       string
	}
	if err := q.Select("invoices.id AS id, invoices.establishment_code AS establishment_code, invoices.emission_point_code AS emission_point_code, " +
		"invoices.sequential AS sequential, " + dayExpr(db, "invoices.issue_date") + " AS issue_day, invoices.total AS total, " +
		"COALESCE(pp.paid, 0) AS paid, invoices.patient_id AS patient_id, " +
		"COALESCE(NULLIF(p.full_name, ''), TRIM(COALESCE(p.first_name, '') || ' ' || COALESCE(p.last_name, ''))) AS patient_name").
		Order("invoices.issue_date ASC, invoices.id ASC").Scan(&rows).Error; err != nil {
		return report, err
	}

	now := time.Now().In(location)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	for _, r := range rows {
		issued, err := time.ParseInLocation("2006-01-02", r.IssueDay, location)
		if err != nil {
			return report, err
		}
		age := int(today.Sub(issued).Hours() / 24)
		if age < 0 {
			age = 0
		}
		balance := round2(r.Total - r.Paid)
		item := reports_dto.ReceivableItem{
			InvoiceID: r.ID, InvoiceNumber: invoiceNumber(r.EstablishmentCode, r.EmissionPointCode, r.Sequential),
			PatientID: r.PatientID, PatientName: r.PatientName, IssueDate: r.IssueDay,
			Total: round2(r.Total), Paid: round2(r.Paid), Balance: balance, AgeDays: age,
			PaymentStatus: cash_services.PaymentStatus(r.Total, r.Paid),
		}
		report.Items = append(report.Items, item)
		report.Totals.Count++
		report.Totals.Total += item.Total
		report.Totals.Paid += item.Paid
		report.Totals.Balance += balance
		b := &report.Buckets[0]
		switch {
		case age > 60:
			b = &report.Buckets[2]
		case age > 30:
			b = &report.Buckets[1]
		}
		b.Count++
		b.Balance += balance
	}
	report.Totals.Total, report.Totals.Paid, report.Totals.Balance = round2(report.Totals.Total), round2(report.Totals.Paid), round2(report.Totals.Balance)
	for i := range report.Buckets {
		report.Buckets[i].Balance = round2(report.Buckets[i].Balance)
	}
	return report, nil
}

func invoiceNumber(establishment, emission, sequential string) string {
	if establishment == "" && emission == "" {
		return sequential
	}
	return establishment + "-" + emission + "-" + sequential
}

// Receivables serves GET /reports/receivables.
func (h *ReportsHandler) Receivables(c *gin.Context) {
	serve(h, c, "reports.receivables.fetch.success", h.buildReceivables, receivablesExport)
}

func displayDate(iso string) string {
	if t, err := time.Parse("2006-01-02", iso); err == nil {
		return t.Format("02/01/2006")
	}
	return iso
}

var receivablesExport = exporter[reports_dto.ReceivablesReport]{
	name: "por_cobrar",
	doc:  reports_templates.Receivables,
	sections: func(tr translate, r reports_dto.ReceivablesReport) []section {
		buckets := section{title: tr("reports.section.buckets"), headers: []string{tr("reports.col.bucket"), tr("reports.col.count"), tr("reports.col.balance")}}
		for _, b := range r.Buckets {
			buckets.rows = append(buckets.rows, []string{tr("reports.bucket." + b.Key), itoa(b.Count), money(b.Balance)})
		}
		items := section{title: tr("reports.section.items"), headers: []string{
			tr("reports.col.invoice"), tr("reports.col.patient"), tr("reports.col.date"), tr("reports.col.total"),
			tr("reports.col.paid"), tr("reports.col.balance"), tr("reports.col.age_days"),
		}}
		for _, i := range r.Items {
			name := i.PatientName
			if name == "" {
				name = tr("reports.final_consumer")
			}
			items.rows = append(items.rows, []string{i.InvoiceNumber, name, i.IssueDate, money(i.Total), money(i.Paid), money(i.Balance), itoa(i.AgeDays)})
		}
		return []section{buckets, items}
	},
	pdfData: func(head reports_templates.Header, r reports_dto.ReceivablesReport) any {
		data := reports_templates.ReceivablesData{
			Header: head, Count: r.Totals.Count, Total: money(r.Totals.Total), Paid: money(r.Totals.Paid), Balance: money(r.Totals.Balance),
		}
		for _, b := range r.Buckets {
			data.Buckets = append(data.Buckets, reports_templates.ReceivableBucketRow{Label: reports_templates.Label(reports_templates.BucketLabels, b.Key), Count: b.Count, Balance: money(b.Balance)})
		}
		for _, i := range r.Items {
			name := i.PatientName
			if name == "" {
				name = "Consumidor final"
			}
			data.Items = append(data.Items, reports_templates.ReceivableRow{
				Invoice: i.InvoiceNumber, Patient: name, Date: displayDate(i.IssueDate),
				Total: money(i.Total), Paid: money(i.Paid), Balance: money(i.Balance), AgeDays: i.AgeDays,
			})
		}
		return data
	},
}
