package ai_services

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"pengi-med-saas/core/ai"
	clinical_models "pengi-med-saas/features/clinical/models"

	"gorm.io/gorm"
)

// Bounds of what one summary sends, so a long history can't blow the
// context window or the cost of a use.
const (
	maxSummaryRecords      = 30   // most recent consultations
	maxSummaryExamOrders   = 20   // most recent exam orders
	maxSummaryCertificates = 10   // most recent medical certificates
	maxFieldRunes          = 1200 // per free-text field
)

// summaryPromptVersion is part of the fingerprint: changing the prompt or
// the input format invalidates every cached summary.
const summaryPromptVersion = "summary-v1"

// SummarySource is a consultation the summary may cite, shown as a link in
// the panel. It never goes to the model (only its label does).
type SummarySource struct {
	Label           string    `json:"label"` // C1, C2… as the model cites it
	MedicalRecordID uint      `json:"medical_record_id"`
	Date            time.Time `json:"date"`
	Motive          string    `json:"motive"`
}

// SummaryInput is what a summary is built from: the de-identified text sent
// to the model and the consultations it can cite.
type SummaryInput struct {
	Prompt   string
	Sources  []SummarySource
	Included int // consultations sent
	Omitted  int // older consultations left out (maxSummaryRecords)
	Empty    bool
}

// PatientHistory is the structured clinical data of one patient a summary
// reads. Attachments (files, exam results) are never part of it.
type PatientHistory struct {
	Patient      clinical_models.Patient
	Records      []clinical_models.MedicalRecord // newest first
	TotalRecords int
	ExamOrders   []clinical_models.ExamOrder // newest first
	Certificates []clinical_models.MedicalCertificate
}

// LoadHistory reads a patient's structured history through db, which must
// be bound to the patient's tenant.
func LoadHistory(db *gorm.DB, patientID uint) (PatientHistory, error) {
	var h PatientHistory
	if err := db.First(&h.Patient, patientID).Error; err != nil {
		return h, err
	}
	var total int64
	if err := db.Model(&clinical_models.MedicalRecord{}).Where("patient_id = ?", patientID).Count(&total).Error; err != nil {
		return h, err
	}
	h.TotalRecords = int(total)
	if err := db.Where("patient_id = ?", patientID).
		Preload("SOAPRecord").Preload("Prescription").Preload("Prescription.Items").Preload("VitalSigns").
		Order("date DESC, id DESC").Limit(maxSummaryRecords).Find(&h.Records).Error; err != nil {
		return h, err
	}
	if err := db.Where("patient_id = ? AND status <> ?", patientID, clinical_models.ExamOrderStatusVoided).
		Preload("Items").Order("created_at DESC, id DESC").Limit(maxSummaryExamOrders).Find(&h.ExamOrders).Error; err != nil {
		return h, err
	}
	if err := db.Where("patient_id = ?", patientID).
		Order("created_at DESC, id DESC").Limit(maxSummaryCertificates).Find(&h.Certificates).Error; err != nil {
		return h, err
	}
	return h, nil
}

// sourceRow is the id and last change of one source row.
type sourceRow struct {
	ID        uint
	UpdatedAt time.Time
}

// Fingerprint identifies the current state of a patient's clinical data: the
// patient's background (updated_at) and the id + last change of every
// consultation, exam order and medical document, and of their child rows
// (SOAP, prescription and its items, vital signs, exam order items), which
// the record handlers update without touching the parent. Creating, editing
// or deleting any of them changes it, which invalidates the cached summary.
// db must be bound to the patient's tenant (First fails for another tenant's
// patient, so the joins below never reach another tenant's rows).
func Fingerprint(db *gorm.DB, patientID uint) (string, error) {
	var patient clinical_models.Patient
	if err := db.Select("id", "updated_at").First(&patient, patientID).Error; err != nil {
		return "", err
	}
	h := sha256.New()
	fmt.Fprintf(h, "%s|p%d:%d", summaryPromptVersion, patient.ID, patient.UpdatedAt.UnixNano())
	// Live consultations of the patient, for the child tables without patient_id.
	const liveRecord = "JOIN medical_records mr ON %s AND mr.deleted_at IS NULL AND mr.patient_id = ?"
	for _, src := range []struct {
		tag   string
		model any
		table string
		join  string // empty: the table has patient_id
	}{
		{"r", &clinical_models.MedicalRecord{}, "medical_records", ""},
		{"e", &clinical_models.ExamOrder{}, "exam_orders", ""},
		{"c", &clinical_models.MedicalCertificate{}, "medical_certificates", ""},
		{"m", &clinical_models.MedicalReport{}, "medical_reports", ""},
		{"s", &clinical_models.SOAPRecord{}, "soap_records", fmt.Sprintf(liveRecord, "mr.soap_record_id = soap_records.id")},
		{"x", &clinical_models.Prescription{}, "prescriptions", fmt.Sprintf(liveRecord, "mr.prescription_id = prescriptions.id")},
		{"i", &clinical_models.PrescriptionItem{}, "prescription_items",
			"JOIN prescriptions px ON px.id = prescription_items.prescription_id AND px.deleted_at IS NULL " +
				fmt.Sprintf(liveRecord, "mr.prescription_id = px.id")},
		{"v", &clinical_models.VitalSigns{}, "vital_signs", fmt.Sprintf(liveRecord, "mr.id = vital_signs.medical_record_id")},
		{"o", &clinical_models.ExamOrderItem{}, "exam_order_items",
			"JOIN exam_orders eo ON eo.id = exam_order_items.exam_order_id AND eo.deleted_at IS NULL AND eo.patient_id = ?"},
	} {
		var rows []sourceRow
		q := db.Model(src.model).Select(src.table + ".id AS id, " + src.table + ".updated_at AS updated_at")
		if src.join == "" {
			q = q.Where(src.table+".patient_id = ?", patientID)
		} else {
			q = q.Joins(src.join, patientID)
		}
		if err := q.Order(src.table + ".id").Find(&rows).Error; err != nil {
			return "", err
		}
		for _, r := range rows {
			fmt.Fprintf(h, "|%s%d:%d", src.tag, r.ID, r.UpdatedAt.UnixNano())
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// BuildSummaryInput turns a history into the de-identified text sent to the
// model: age and sex instead of birth date and name, no document, phone,
// email, address, doctor or clinic, free text through ai.Redactor.
func BuildSummaryInput(h PatientHistory, now time.Time) SummaryInput {
	p := h.Patient
	names := []string{p.FirstName, p.LastName}
	if p.FullName != nil {
		names = append(names, *p.FullName)
	}
	r := ai.NewRedactor(names...)
	clean := func(s string) string { return truncate(strings.TrimSpace(r.Redact(s))) }

	var b strings.Builder
	b.WriteString("PACIENTE\n")
	fmt.Fprintf(&b, "Edad: %s\n", ageText(p.BirthDate, now))
	fmt.Fprintf(&b, "Sexo: %s\n", sexText(p.Gender))
	writeField(&b, "Antecedentes personales patológicos", clean(p.APP))
	writeField(&b, "Antecedentes familiares", clean(p.APF))
	writeField(&b, "Antecedentes quirúrgicos", clean(p.APQX))
	writeField(&b, "Alergias", clean(allergiesText(p.Allergies)))
	writeField(&b, "Diagnóstico registrado en la ficha", clean(p.Diagnosis))
	if p.Critical {
		b.WriteString("Marcado como paciente crítico: sí\n")
	}

	in := SummaryInput{}
	// Oldest first, so the model reads the evolution in order; labels follow
	// that order (C1 is the oldest consultation sent).
	records := h.Records
	in.Included = len(records)
	in.Omitted = max(h.TotalRecords-len(records), 0)
	if len(records) > 0 {
		b.WriteString("\nCONSULTAS (de la más antigua a la más reciente")
		if in.Omitted > 0 {
			fmt.Fprintf(&b, "; %d consultas anteriores no incluidas", in.Omitted)
		}
		b.WriteString(")\n")
	}
	for i := len(records) - 1; i >= 0; i-- {
		rec := records[i]
		label := "C" + strconv.Itoa(len(in.Sources)+1)
		in.Sources = append(in.Sources, SummarySource{Label: label, MedicalRecordID: rec.ID, Date: rec.Date, Motive: rec.Motive})
		fmt.Fprintf(&b, "\n[%s] Fecha: %s", label, rec.Date.Format("2006-01-02"))
		switch rec.VisitType {
		case "first":
			b.WriteString(" (primera consulta)")
		case "followup":
			b.WriteString(" (control)")
		}
		b.WriteString("\n")
		writeField(&b, "Motivo", clean(rec.Motive))
		writeField(&b, "Observación", clean(rec.Observation))
		writeField(&b, "Subjetivo", clean(rec.SOAPRecord.Subjective))
		writeField(&b, "Objetivo", clean(rec.SOAPRecord.Objective))
		writeField(&b, "Análisis", clean(rec.SOAPRecord.Assessment))
		writeField(&b, "Plan", clean(rec.SOAPRecord.Plan))
		writeField(&b, "Diagnósticos", clean(diagnosesText(rec.Diagnoses)))
		writeField(&b, "Signos vitales", vitalsText(rec.VitalSigns))
		if rx := rec.Prescription; rx != nil {
			var items []string
			for _, it := range rx.Items {
				items = append(items, strings.Join(nonEmpty(it.Medication, it.Dose, it.Frequency, it.Duration), " "))
			}
			writeField(&b, "Receta", clean(strings.Join(items, "; ")))
			writeField(&b, "Indicaciones", clean(rx.Indications))
		}
	}

	if len(h.ExamOrders) > 0 {
		b.WriteString("\nÓRDENES DE EXÁMENES (los resultados no se incluyen)\n")
	}
	for i := len(h.ExamOrders) - 1; i >= 0; i-- {
		o := h.ExamOrders[i]
		var names []string
		for _, it := range o.Items {
			names = append(names, it.Name)
		}
		fmt.Fprintf(&b, "- %s: %s (estado: %s)", o.CreatedAt.Format("2006-01-02"), clean(strings.Join(names, ", ")), examStatusText(o.Status))
		if o.Priority == clinical_models.ExamPriorityUrgent {
			b.WriteString(" [urgente]")
		}
		if n := clean(o.Notes); n != "" {
			fmt.Fprintf(&b, ". Notas: %s", n)
		}
		b.WriteString("\n")
	}

	if len(h.Certificates) > 0 {
		b.WriteString("\nCERTIFICADOS MÉDICOS\n")
	}
	for i := len(h.Certificates) - 1; i >= 0; i-- {
		cert := h.Certificates[i]
		fmt.Fprintf(&b, "- %s: %s", cert.CreatedAt.Format("2006-01-02"), clean(cert.Diagnosis))
		if cert.RestDays != nil && *cert.RestDays > 0 {
			fmt.Fprintf(&b, " (reposo %d días)", *cert.RestDays)
		}
		b.WriteString("\n")
	}

	in.Prompt = b.String()
	in.Empty = len(records) == 0 && len(h.ExamOrders) == 0 && len(h.Certificates) == 0 &&
		strings.TrimSpace(p.APP+p.APF+p.APQX+allergiesText(p.Allergies)+p.Diagnosis) == ""
	return in
}

func writeField(b *strings.Builder, label, value string) {
	if value == "" {
		return
	}
	fmt.Fprintf(b, "%s: %s\n", label, value)
}

func truncate(s string) string {
	if utf8.RuneCountInString(s) <= maxFieldRunes {
		return s
	}
	return string([]rune(s)[:maxFieldRunes]) + "…"
}

func nonEmpty(values ...string) []string {
	var out []string
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// ageText is the age in years (months under two), never the birth date.
func ageText(birth, now time.Time) string {
	if birth.IsZero() || birth.After(now) {
		return "no registrada"
	}
	years := now.Year() - birth.Year()
	if now.YearDay() < birth.YearDay() {
		years--
	}
	if years < 2 {
		months := (now.Year()-birth.Year())*12 + int(now.Month()) - int(birth.Month())
		if now.Day() < birth.Day() {
			months--
		}
		return fmt.Sprintf("%d meses", max(months, 0))
	}
	return fmt.Sprintf("%d años", years)
}

func sexText(gender string) string {
	switch strings.ToLower(strings.TrimSpace(gender)) {
	case "m", "male", "masculino", "hombre":
		return "masculino"
	case "f", "female", "femenino", "mujer":
		return "femenino"
	case "":
		return "no registrado"
	default:
		return "otro"
	}
}

// allergiesText reads Patient.Allergies, a JSON array of strings (or plain
// text in older rows).
func allergiesText(raw string) string {
	raw = strings.TrimSpace(raw)
	var list []string
	if strings.HasPrefix(raw, "[") && json.Unmarshal([]byte(raw), &list) == nil {
		return strings.Join(nonEmpty(list...), ", ")
	}
	return raw
}

func diagnosesText(raw []byte) string {
	var items []clinical_models.DiagnosisItem
	if len(raw) == 0 || json.Unmarshal(raw, &items) != nil {
		return ""
	}
	var out []string
	for _, d := range items {
		out = append(out, strings.Join(nonEmpty(d.Code, d.Title), " "))
	}
	return strings.Join(nonEmpty(out...), "; ")
}

func vitalsText(v *clinical_models.VitalSigns) string {
	if v == nil {
		return ""
	}
	var parts []string
	if v.BloodPressure != "" {
		parts = append(parts, "PA "+v.BloodPressure)
	}
	if v.HeartRate != nil {
		parts = append(parts, fmt.Sprintf("FC %d", *v.HeartRate))
	}
	if v.Temperature != nil {
		parts = append(parts, fmt.Sprintf("T %.1f °C", *v.Temperature))
	}
	if v.Weight != nil {
		parts = append(parts, fmt.Sprintf("peso %.1f kg", *v.Weight))
	}
	if v.Height != nil {
		parts = append(parts, fmt.Sprintf("talla %.0f cm", *v.Height))
	}
	return strings.Join(parts, ", ")
}

func examStatusText(status string) string {
	switch status {
	case clinical_models.ExamOrderStatusIssued:
		return "emitida, sin resultados"
	case clinical_models.ExamOrderStatusPartialResults:
		return "resultados parciales"
	case clinical_models.ExamOrderStatusCompleteResults:
		return "resultados completos"
	default:
		return status
	}
}
