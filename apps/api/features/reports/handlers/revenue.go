package reports_handlers

import (
	"sort"

	billing_models "pengi-med-saas/features/billing/models"
	cash_models "pengi-med-saas/features/cash/models"
	reports_dto "pengi-med-saas/features/reports/dto"
	reports_templates "pengi-med-saas/features/reports/templates"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// notBilled are the invoice statuses that don't count as invoiced.
var notBilled = []string{"draft", "rejected"}

// invoicesQuery is the invoices issued in the range, each with its patient
// (alias p) for the doctor filter and breakdown. db is bound to the tenant.
func invoicesQuery(db *gorm.DB, f filter) *gorm.DB {
	q := db.Model(&billing_models.Invoice{}).
		Joins("LEFT JOIN patients p ON p.id = invoices.patient_id AND p.tenant_id = invoices.tenant_id").
		Where("invoices.status NOT IN ?", notBilled).
		Where("invoices.issue_date >= ? AND invoices.issue_date < ?", f.start, f.end)
	if f.doctorID != nil {
		q = q.Where("p.doctor_id = ?", *f.doctorID)
	}
	return q
}

// paymentsQuery is the valid payments and refunds made in the range, each with
// its invoice (alias i) and the invoice's patient (alias p). A refund is a
// payment with a negative amount, so summing amount is payments minus refunds
// (the same convention as cash_services).
func paymentsQuery(db *gorm.DB, f filter) *gorm.DB {
	q := db.Model(&cash_models.Payment{}).
		Joins("JOIN invoices i ON i.id = cash_payments.invoice_id AND i.tenant_id = cash_payments.tenant_id").
		Joins("LEFT JOIN patients p ON p.id = i.patient_id AND p.tenant_id = i.tenant_id").
		Where("cash_payments.status = ?", cash_models.PaymentValid).
		Where("cash_payments.created_at >= ? AND cash_payments.created_at < ?", f.start, f.end)
	if f.doctorID != nil {
		q = q.Where("p.doctor_id = ?", *f.doctorID)
	}
	return q
}

type doctorKey struct {
	set bool
	id  uint
}

func (h *ReportsHandler) buildRevenue(db *gorm.DB, tenantID uint, f filter) (reports_dto.RevenueReport, error) {
	report := reports_dto.RevenueReport{Filters: f.dto(), Granularity: "day"}
	monthly := f.days > monthlyAfterDays
	if monthly {
		report.Granularity = "month"
	}

	// Invoiced: totals, per day, per doctor.
	invDay := dayExpr(db, "invoices.issue_date")
	var invoiced []struct {
		Day   string
		Total float64
		Count int
	}
	if err := invoicesQuery(db, f).Select(invDay + " AS day, SUM(invoices.total) AS total, COUNT(*) AS count").Group("day").Scan(&invoiced).Error; err != nil {
		return report, err
	}
	var invByDoctor []struct {
		DoctorID   *uint
		DoctorName string
		Total      float64
	}
	if err := invoicesQuery(db, f).
		Joins("LEFT JOIN doctors d ON d.id = p.doctor_id AND d.tenant_id = invoices.tenant_id").
		Select("p.doctor_id AS doctor_id, MAX(d.full_name) AS doctor_name, SUM(invoices.total) AS total").
		Group("p.doctor_id").Scan(&invByDoctor).Error; err != nil {
		return report, err
	}

	// Collected: per day, per method, per doctor.
	payDay := dayExpr(db, "cash_payments.created_at")
	var collected []struct {
		Day   string
		Total float64
		Count int
	}
	if err := paymentsQuery(db, f).Select(payDay + " AS day, SUM(cash_payments.amount) AS total, COUNT(*) AS count").Group("day").Scan(&collected).Error; err != nil {
		return report, err
	}
	var byMethod []struct {
		Method string
		Total  float64
		Count  int
	}
	if err := paymentsQuery(db, f).Select("cash_payments.method AS method, SUM(cash_payments.amount) AS total, COUNT(*) AS count").Group("cash_payments.method").Scan(&byMethod).Error; err != nil {
		return report, err
	}
	var payByDoctor []struct {
		DoctorID   *uint
		DoctorName string
		Total      float64
	}
	if err := paymentsQuery(db, f).
		Joins("LEFT JOIN doctors d ON d.id = p.doctor_id AND d.tenant_id = cash_payments.tenant_id").
		Select("p.doctor_id AS doctor_id, MAX(d.full_name) AS doctor_name, SUM(cash_payments.amount) AS total").
		Group("p.doctor_id").Scan(&payByDoctor).Error; err != nil {
		return report, err
	}

	// Receivable: what is still owed on the invoices issued in the range.
	receivable, err := h.receivableTotal(db, tenantID, f)
	if err != nil {
		return report, err
	}

	// Assemble.
	points := map[string]*reports_dto.RevenuePoint{}
	point := func(day string) *reports_dto.RevenuePoint {
		period := day
		if monthly && len(day) >= 7 {
			period = day[:7]
		}
		if points[period] == nil {
			points[period] = &reports_dto.RevenuePoint{Period: period}
		}
		return points[period]
	}
	for _, r := range invoiced {
		point(r.Day).Invoiced += r.Total
		report.Totals.Invoiced += r.Total
		report.Totals.InvoiceCount += r.Count
	}
	for _, r := range collected {
		point(r.Day).Collected += r.Total
		report.Totals.Collected += r.Total
		report.Totals.PaymentCount += r.Count
	}
	report.Totals.Invoiced = round2(report.Totals.Invoiced)
	report.Totals.Collected = round2(report.Totals.Collected)
	report.Totals.Receivable = receivable

	report.Series = make([]reports_dto.RevenuePoint, 0, len(points))
	for _, p := range points {
		p.Invoiced, p.Collected = round2(p.Invoiced), round2(p.Collected)
		report.Series = append(report.Series, *p)
	}
	sort.Slice(report.Series, func(i, j int) bool { return report.Series[i].Period < report.Series[j].Period })

	report.ByMethod = []reports_dto.RevenueMethod{}
	rank := map[string]int{}
	for i, m := range cash_models.Methods {
		rank[m] = i
	}
	for _, r := range byMethod {
		report.ByMethod = append(report.ByMethod, reports_dto.RevenueMethod{Method: r.Method, Total: round2(r.Total), Count: r.Count})
	}
	sort.Slice(report.ByMethod, func(i, j int) bool { return rank[report.ByMethod[i].Method] < rank[report.ByMethod[j].Method] })

	doctors := map[doctorKey]*reports_dto.RevenueByDoctor{}
	doctor := func(id *uint, name string) *reports_dto.RevenueByDoctor {
		k := doctorKey{}
		if id != nil {
			k = doctorKey{true, *id}
		}
		if doctors[k] == nil {
			doctors[k] = &reports_dto.RevenueByDoctor{DoctorID: id, DoctorName: name}
		}
		return doctors[k]
	}
	for _, r := range invByDoctor {
		doctor(r.DoctorID, r.DoctorName).Invoiced += r.Total
	}
	for _, r := range payByDoctor {
		doctor(r.DoctorID, r.DoctorName).Collected += r.Total
	}
	report.ByDoctor = make([]reports_dto.RevenueByDoctor, 0, len(doctors))
	for _, d := range doctors {
		d.Invoiced, d.Collected = round2(d.Invoiced), round2(d.Collected)
		report.ByDoctor = append(report.ByDoctor, *d)
	}
	sort.Slice(report.ByDoctor, func(i, j int) bool {
		if report.ByDoctor[i].Invoiced != report.ByDoctor[j].Invoiced {
			return report.ByDoctor[i].Invoiced > report.ByDoctor[j].Invoiced
		}
		return report.ByDoctor[i].DoctorName < report.ByDoctor[j].DoctorName
	})
	return report, nil
}

// Revenue serves GET /reports/revenue.
func (h *ReportsHandler) Revenue(c *gin.Context) {
	serve(h, c, "reports.revenue.fetch.success", h.buildRevenue, revenueExport)
}

var revenueExport = exporter[reports_dto.RevenueReport]{
	name: "ingresos",
	doc:  reports_templates.Revenue,
	sections: func(tr translate, r reports_dto.RevenueReport) []section {
		summary := section{title: tr("reports.section.summary"), headers: []string{tr("reports.col.invoiced"), tr("reports.col.collected"), tr("reports.col.receivable")},
			rows: [][]string{{money(r.Totals.Invoiced), money(r.Totals.Collected), money(r.Totals.Receivable)}}}
		series := section{title: tr("reports.section.series"), headers: []string{tr("reports.col.period"), tr("reports.col.invoiced"), tr("reports.col.collected")}}
		for _, p := range r.Series {
			series.rows = append(series.rows, []string{p.Period, money(p.Invoiced), money(p.Collected)})
		}
		methods := section{title: tr("reports.section.by_method"), headers: []string{tr("reports.col.method"), tr("reports.col.count"), tr("reports.col.total")}}
		for _, m := range r.ByMethod {
			methods.rows = append(methods.rows, []string{tr("reports.method." + m.Method), itoa(m.Count), money(m.Total)})
		}
		doctors := section{title: tr("reports.section.by_doctor"), headers: []string{tr("reports.col.doctor"), tr("reports.col.invoiced"), tr("reports.col.collected")}}
		for _, d := range r.ByDoctor {
			doctors.rows = append(doctors.rows, []string{doctorLabel(tr, d.DoctorName), money(d.Invoiced), money(d.Collected)})
		}
		return []section{summary, series, methods, doctors}
	},
	pdfData: func(head reports_templates.Header, r reports_dto.RevenueReport) any {
		data := reports_templates.RevenueData{
			Header: head, Invoiced: money(r.Totals.Invoiced), Collected: money(r.Totals.Collected), Receivable: money(r.Totals.Receivable),
			InvoiceCount: r.Totals.InvoiceCount, PaymentCount: r.Totals.PaymentCount, Monthly: r.Granularity == "month",
		}
		for _, p := range r.Series {
			data.Series = append(data.Series, reports_templates.RevenueSeriesRow{Period: p.Period, Invoiced: money(p.Invoiced), Collected: money(p.Collected)})
		}
		for _, m := range r.ByMethod {
			data.Methods = append(data.Methods, reports_templates.RevenueMethodRow{Method: reports_templates.Label(reports_templates.MethodLabels, m.Method), Count: m.Count, Total: money(m.Total)})
		}
		for _, d := range r.ByDoctor {
			name := d.DoctorName
			if name == "" {
				name = "Sin médico"
			}
			data.Doctors = append(data.Doctors, reports_templates.RevenueDoctorRow{Doctor: name, Invoiced: money(d.Invoiced), Collected: money(d.Collected)})
		}
		return data
	},
}
