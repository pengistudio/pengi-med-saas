package reports_handlers

import (
	"sort"

	clinical_models "pengi-med-saas/features/clinical/models"
	reports_dto "pengi-med-saas/features/reports/dto"
	reports_templates "pengi-med-saas/features/reports/templates"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// topDiagnoses is how many CIE-10 codes the report lists.
const topDiagnoses = 20

// recordsQuery is the consultations (medical records) dated in the range.
func recordsQuery(db *gorm.DB, f filter) *gorm.DB {
	q := db.Model(&clinical_models.MedicalRecord{}).
		Where("medical_records.date >= ? AND medical_records.date < ?", f.start, f.end)
	if f.doctorID != nil {
		q = q.Where("medical_records.doctor_id = ?", *f.doctorID)
	}
	return q
}

func (h *ReportsHandler) buildClinical(db *gorm.DB, tenantID uint, f filter) (reports_dto.ClinicalReport, error) {
	report := reports_dto.ClinicalReport{Filters: f.dto(), Top: []reports_dto.DiagnosisCount{}}

	// Diagnoses live in a jsonb array of {code, title}: unnest it in SQL and
	// count the distinct consultations carrying each code.
	join, code, title := "CROSS JOIN LATERAL jsonb_array_elements(medical_records.diagnoses) AS dx", "dx->>'code'", "dx->>'title'"
	if db.Dialector.Name() == "sqlite" {
		join, code, title = "CROSS JOIN json_each(medical_records.diagnoses) AS dx", "json_extract(dx.value, '$.code')", "json_extract(dx.value, '$.title')"
	}
	if err := recordsQuery(db, f).Joins(join).Where(code + " IS NOT NULL AND " + code + " <> ''").
		Select(code + " AS code, MAX(" + title + ") AS title, COUNT(DISTINCT medical_records.id) AS count").
		Group(code).Order("count DESC, code ASC").Limit(topDiagnoses).Scan(&report.Top).Error; err != nil {
		return report, err
	}

	// Consultations and prescriptions per doctor (a record has at most one).
	var records []struct {
		DoctorID      *uint
		DoctorName    string
		Consultations int
		Prescriptions int
	}
	if err := recordsQuery(db, f).
		Joins("LEFT JOIN prescriptions pr ON pr.medical_record_id = medical_records.id AND pr.deleted_at IS NULL").
		Joins("LEFT JOIN doctors d ON d.id = medical_records.doctor_id AND d.tenant_id = medical_records.tenant_id").
		Select("medical_records.doctor_id AS doctor_id, MAX(d.full_name) AS doctor_name, " +
			"COUNT(DISTINCT medical_records.id) AS consultations, COUNT(DISTINCT pr.id) AS prescriptions").
		Group("medical_records.doctor_id").Scan(&records).Error; err != nil {
		return report, err
	}

	// Exam orders issued in the range (voided ones don't count).
	orders := db.Model(&clinical_models.ExamOrder{}).
		Joins("LEFT JOIN doctors d ON d.id = exam_orders.doctor_id AND d.tenant_id = exam_orders.tenant_id").
		Where("exam_orders.created_at >= ? AND exam_orders.created_at < ? AND exam_orders.voided_at IS NULL", f.start, f.end)
	if f.doctorID != nil {
		orders = orders.Where("exam_orders.doctor_id = ?", *f.doctorID)
	}
	var exams []struct {
		DoctorID   *uint
		DoctorName string
		ExamOrders int
	}
	if err := orders.Select("exam_orders.doctor_id AS doctor_id, MAX(d.full_name) AS doctor_name, COUNT(*) AS exam_orders").
		Group("exam_orders.doctor_id").Scan(&exams).Error; err != nil {
		return report, err
	}

	byDoctor := map[doctorKey]*reports_dto.ClinicalByDoctor{}
	doctor := func(id *uint, name string) *reports_dto.ClinicalByDoctor {
		k := doctorKey{}
		if id != nil {
			k = doctorKey{true, *id}
		}
		if byDoctor[k] == nil {
			byDoctor[k] = &reports_dto.ClinicalByDoctor{DoctorID: id, DoctorName: name}
		}
		return byDoctor[k]
	}
	for _, r := range records {
		d := doctor(r.DoctorID, r.DoctorName)
		d.Consultations += r.Consultations
		d.Prescriptions += r.Prescriptions
	}
	for _, r := range exams {
		doctor(r.DoctorID, r.DoctorName).ExamOrders += r.ExamOrders
	}
	report.ByDoctor = make([]reports_dto.ClinicalByDoctor, 0, len(byDoctor))
	for _, d := range byDoctor {
		report.ByDoctor = append(report.ByDoctor, *d)
		report.Totals.Consultations += d.Consultations
		report.Totals.Prescriptions += d.Prescriptions
		report.Totals.ExamOrders += d.ExamOrders
	}
	sort.Slice(report.ByDoctor, func(i, j int) bool {
		a, b := report.ByDoctor[i], report.ByDoctor[j]
		if a.Consultations != b.Consultations {
			return a.Consultations > b.Consultations
		}
		return a.DoctorName < b.DoctorName
	})
	return report, nil
}

// Clinical serves GET /reports/clinical.
func (h *ReportsHandler) Clinical(c *gin.Context) {
	serve(h, c, "reports.clinical.fetch.success", h.buildClinical, clinicalExport)
}

var clinicalExport = exporter[reports_dto.ClinicalReport]{
	name: "clinico",
	doc:  reports_templates.Clinical,
	sections: func(tr translate, r reports_dto.ClinicalReport) []section {
		summary := section{title: tr("reports.section.summary"), headers: []string{tr("reports.col.consultations"), tr("reports.col.prescriptions"), tr("reports.col.exam_orders")},
			rows: [][]string{{itoa(r.Totals.Consultations), itoa(r.Totals.Prescriptions), itoa(r.Totals.ExamOrders)}}}
		top := section{title: tr("reports.section.top_diagnoses"), headers: []string{tr("reports.col.code"), tr("reports.col.diagnosis"), tr("reports.col.count")}}
		for _, d := range r.Top {
			top.rows = append(top.rows, []string{d.Code, d.Title, itoa(d.Count)})
		}
		doctors := section{title: tr("reports.section.by_doctor"), headers: []string{tr("reports.col.doctor"), tr("reports.col.consultations"), tr("reports.col.prescriptions"), tr("reports.col.exam_orders")}}
		for _, d := range r.ByDoctor {
			doctors.rows = append(doctors.rows, []string{doctorLabel(tr, d.DoctorName), itoa(d.Consultations), itoa(d.Prescriptions), itoa(d.ExamOrders)})
		}
		return []section{summary, top, doctors}
	},
	pdfData: func(head reports_templates.Header, r reports_dto.ClinicalReport) any {
		data := reports_templates.ClinicalData{Header: head, Consultations: r.Totals.Consultations, Prescriptions: r.Totals.Prescriptions, ExamOrders: r.Totals.ExamOrders}
		for _, d := range r.Top {
			data.Diagnoses = append(data.Diagnoses, reports_templates.DiagnosisRow{Code: d.Code, Title: d.Title, Count: d.Count})
		}
		for _, d := range r.ByDoctor {
			name := d.DoctorName
			if name == "" {
				name = "Sin médico"
			}
			data.Doctors = append(data.Doctors, reports_templates.ClinicalDoctorRow{Doctor: name, Consultations: d.Consultations, Prescriptions: d.Prescriptions, ExamOrders: d.ExamOrders})
		}
		return data
	},
}
