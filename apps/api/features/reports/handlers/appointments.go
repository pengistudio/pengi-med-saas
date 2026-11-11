package reports_handlers

import (
	"fmt"
	"sort"

	clinical_models "pengi-med-saas/features/clinical/models"
	reports_dto "pengi-med-saas/features/reports/dto"
	reports_templates "pengi-med-saas/features/reports/templates"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// appointmentsQuery is the appointments whose date falls in the range. The
// date column holds a plain calendar date, compared as DATE(date) like the
// agenda does.
func appointmentsQuery(db *gorm.DB, f filter) *gorm.DB {
	q := db.Model(&clinical_models.Appointment{}).
		Where("DATE(appointments.date) >= DATE(?) AND DATE(appointments.date) <= DATE(?)", utcDay(f.from), utcDay(f.to))
	if f.doctorID != nil {
		q = q.Where("appointments.doctor_id = ?", *f.doctorID)
	}
	return q
}

const (
	statusCompleted = clinical_models.AppointmentStatusCompleted
	statusCancelled = clinical_models.AppointmentStatusCancelled
	statusNoShow    = clinical_models.AppointmentStatusNoShow
)

// statusCounts are the SUM(CASE ...) columns every breakdown shares.
const statusCounts = "COUNT(*) AS total, " +
	"SUM(CASE WHEN appointments.status = 'completed' THEN 1 ELSE 0 END) AS completed, " +
	"SUM(CASE WHEN appointments.status = 'cancelled' THEN 1 ELSE 0 END) AS cancelled, " +
	"SUM(CASE WHEN appointments.status = 'no_show' THEN 1 ELSE 0 END) AS no_show"

func (h *ReportsHandler) buildAppointments(db *gorm.DB, tenantID uint, f filter) (reports_dto.AppointmentsReport, error) {
	report := reports_dto.AppointmentsReport{Filters: f.dto()}

	var byStatus []reports_dto.AppointmentStatus
	if err := appointmentsQuery(db, f).Select("appointments.status AS status, COUNT(*) AS count").Group("appointments.status").Scan(&byStatus).Error; err != nil {
		return report, err
	}
	rank := map[string]int{"scheduled": 0, "confirmed": 1, statusCompleted: 2, statusCancelled: 3, statusNoShow: 4}
	sort.Slice(byStatus, func(i, j int) bool { return rank[byStatus[i].Status] < rank[byStatus[j].Status] })
	report.ByStatus = append([]reports_dto.AppointmentStatus{}, byStatus...)
	for _, s := range byStatus {
		report.Totals.Total += s.Count
		switch s.Status {
		case statusCompleted:
			report.Totals.Completed = s.Count
		case statusCancelled:
			report.Totals.Cancelled = s.Count
		case statusNoShow:
			report.Totals.NoShow = s.Count
		}
	}
	if attended := report.Totals.Completed + report.Totals.NoShow; attended > 0 {
		report.Totals.NoShowRate = float64(int(float64(report.Totals.NoShow)/float64(attended)*10000+0.5)) / 10000
	}

	report.ByDoctor = []reports_dto.AppointmentByDoctor{}
	if err := appointmentsQuery(db, f).
		Joins("LEFT JOIN doctors d ON d.id = appointments.doctor_id AND d.tenant_id = appointments.tenant_id").
		Select("appointments.doctor_id AS doctor_id, MAX(d.full_name) AS doctor_name, " + statusCounts).
		Group("appointments.doctor_id").Order("total DESC").Scan(&report.ByDoctor).Error; err != nil {
		return report, err
	}

	report.ByType = []reports_dto.AppointmentByType{}
	if err := appointmentsQuery(db, f).
		Joins("LEFT JOIN appointment_types t ON t.id = appointments.appointment_type_id AND t.tenant_id = appointments.tenant_id").
		Select("appointments.appointment_type_id AS type_id, MAX(t.name) AS type_name, COUNT(*) AS count").
		Group("appointments.appointment_type_id").Order("count DESC").Scan(&report.ByType).Error; err != nil {
		return report, err
	}

	report.Series = []reports_dto.AppointmentDayPoint{}
	if err := appointmentsQuery(db, f).
		Select("CAST(DATE(appointments.date) AS TEXT) AS day, COUNT(*) AS total, SUM(CASE WHEN appointments.status = 'no_show' THEN 1 ELSE 0 END) AS no_show").
		Group("day").Order("day ASC").Scan(&report.Series).Error; err != nil {
		return report, err
	}

	// New vs recurring: distinct patients attended or expected in the range,
	// new when their first appointment ever (cancelled ones don't count) falls
	// inside it.
	first := "JOIN (SELECT patient_id, CAST(MIN(DATE(date)) AS TEXT) AS first_day FROM appointments " +
		"WHERE tenant_id = ? AND deleted_at IS NULL AND status <> 'cancelled' GROUP BY patient_id) fa ON fa.patient_id = appointments.patient_id"
	var patients struct{ New, Recurring int }
	if err := appointmentsQuery(db, f).Joins(first, tenantID).Where("appointments.status <> 'cancelled'").
		Select("COUNT(DISTINCT CASE WHEN fa.first_day >= ? THEN appointments.patient_id END) AS new, "+
			"COUNT(DISTINCT CASE WHEN fa.first_day < ? THEN appointments.patient_id END) AS recurring",
			f.from.Format("2006-01-02"), f.from.Format("2006-01-02")).Scan(&patients).Error; err != nil {
		return report, err
	}
	report.Patients = reports_dto.AppointmentPatients{New: patients.New, Recurring: patients.Recurring}
	return report, nil
}

// Appointments serves GET /reports/appointments.
func (h *ReportsHandler) Appointments(c *gin.Context) {
	serve(h, c, "reports.appointments.fetch.success", h.buildAppointments, appointmentsExport)
}

var appointmentsExport = exporter[reports_dto.AppointmentsReport]{
	name: "citas",
	doc:  reports_templates.Appointments,
	sections: func(tr translate, r reports_dto.AppointmentsReport) []section {
		summary := section{title: tr("reports.section.summary"),
			headers: []string{tr("reports.col.total"), tr("reports.col.completed"), tr("reports.col.cancelled"), tr("reports.col.no_show"), tr("reports.col.no_show_rate"), tr("reports.col.new_patients"), tr("reports.col.recurring_patients")},
			rows: [][]string{{itoa(r.Totals.Total), itoa(r.Totals.Completed), itoa(r.Totals.Cancelled), itoa(r.Totals.NoShow),
				fmt.Sprintf("%.2f%%", r.Totals.NoShowRate*100), itoa(r.Patients.New), itoa(r.Patients.Recurring)}}}
		status := section{title: tr("reports.section.by_status"), headers: []string{tr("reports.col.status"), tr("reports.col.count")}}
		for _, s := range r.ByStatus {
			status.rows = append(status.rows, []string{tr("reports.appt_status." + s.Status), itoa(s.Count)})
		}
		doctors := section{title: tr("reports.section.by_doctor"), headers: []string{tr("reports.col.doctor"), tr("reports.col.total"), tr("reports.col.completed"), tr("reports.col.cancelled"), tr("reports.col.no_show")}}
		for _, d := range r.ByDoctor {
			doctors.rows = append(doctors.rows, []string{doctorLabel(tr, d.DoctorName), itoa(d.Total), itoa(d.Completed), itoa(d.Cancelled), itoa(d.NoShow)})
		}
		types := section{title: tr("reports.section.by_type"), headers: []string{tr("reports.col.type"), tr("reports.col.count")}}
		for _, t := range r.ByType {
			name := t.TypeName
			if name == "" {
				name = tr("reports.no_type")
			}
			types.rows = append(types.rows, []string{name, itoa(t.Count)})
		}
		series := section{title: tr("reports.section.series"), headers: []string{tr("reports.col.date"), tr("reports.col.total"), tr("reports.col.no_show")}}
		for _, p := range r.Series {
			series.rows = append(series.rows, []string{p.Day, itoa(p.Total), itoa(p.NoShow)})
		}
		return []section{summary, status, doctors, types, series}
	},
	pdfData: func(head reports_templates.Header, r reports_dto.AppointmentsReport) any {
		data := reports_templates.AppointmentsData{
			Header: head, Total: r.Totals.Total, Completed: r.Totals.Completed, Cancelled: r.Totals.Cancelled, NoShow: r.Totals.NoShow,
			NoShowRate:  fmt.Sprintf("%.1f%%", r.Totals.NoShowRate*100),
			NewPatients: r.Patients.New, RecurringPatients: r.Patients.Recurring,
		}
		for _, s := range r.ByStatus {
			data.Statuses = append(data.Statuses, reports_templates.AppointmentStatusRow{Status: reports_templates.Label(reports_templates.StatusLabels, s.Status), Count: s.Count})
		}
		for _, d := range r.ByDoctor {
			name := d.DoctorName
			if name == "" {
				name = "Sin médico"
			}
			data.Doctors = append(data.Doctors, reports_templates.AppointmentDoctorRow{Doctor: name, Total: d.Total, Completed: d.Completed, Cancelled: d.Cancelled, NoShow: d.NoShow})
		}
		for _, t := range r.ByType {
			name := t.TypeName
			if name == "" {
				name = "Sin tipo"
			}
			data.Types = append(data.Types, reports_templates.AppointmentTypeRow{Type: name, Count: t.Count})
		}
		for _, p := range r.Series {
			data.Series = append(data.Series, reports_templates.AppointmentDayRow{Day: p.Day, Total: p.Total, NoShow: p.NoShow})
		}
		return data
	},
}
