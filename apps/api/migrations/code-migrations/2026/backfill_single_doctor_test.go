package y2026

import (
	"fmt"
	"testing"
	"time"

	"pengi-med-saas/core/database"
	"pengi-med-saas/core/tenantdb"
	clinical_models "pengi-med-saas/features/clinical/models"
	doctor_models "pengi-med-saas/features/doctors/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"

	"gorm.io/gorm"
)

// singleDoctorRows are one row of every backfilled table for a tenant.
type singleDoctorRows struct {
	patient      uint
	appointment  uint
	record       uint
	prescription uint
	certificate  uint
	report       uint
	order        uint
}

func createSingleDoctorRows(t *testing.T, db *gorm.DB, tenantID uint) singleDoctorRows {
	t.Helper()
	must := func(v any) {
		if err := db.Create(v).Error; err != nil {
			t.Fatalf("create %T: %v", v, err)
		}
	}
	p := clinical_models.Patient{TenantID: tenantID, FirstName: "P", LastName: "X", Institution: "H", Document: fmt.Sprintf("DOC-%d", time.Now().UnixNano())}
	must(&p)
	a := clinical_models.Appointment{TenantID: tenantID, PatientID: p.ID, Title: "Control", Date: time.Now(), StartTime: "09:00", EndTime: "09:30"}
	must(&a)
	r := clinical_models.MedicalRecord{TenantID: tenantID, PatientID: p.ID, Motive: "M", Prescription: &clinical_models.Prescription{Content: "Rx"}}
	must(&r)
	cert := clinical_models.MedicalCertificate{TenantID: tenantID, PatientID: p.ID, Diagnosis: "D"}
	must(&cert)
	rep := clinical_models.MedicalReport{TenantID: tenantID, PatientID: p.ID}
	must(&rep)
	o := clinical_models.ExamOrder{TenantID: tenantID, Number: uint(time.Now().UnixNano() % 1e6), PatientID: p.ID, OrderedByID: 1}
	must(&o)
	return singleDoctorRows{p.ID, a.ID, r.ID, *r.PrescriptionID, cert.ID, rep.ID, o.ID}
}

func doctorOf(t *testing.T, db *gorm.DB, model any, id uint) *uint {
	t.Helper()
	var out struct{ DoctorID *uint }
	if err := db.Model(model).Select("doctor_id").Where("id = ?", id).Scan(&out).Error; err != nil {
		t.Fatal(err)
	}
	return out.DoctorID
}

func TestBackfillSingleDoctor(t *testing.T) {
	db := tenantdb.System(testutils.SetupTestDB(t, &tenant_models.Tenant{}, &doctor_models.Doctor{},
		&clinical_models.Patient{}, &clinical_models.Appointment{}, &clinical_models.SOAPRecord{}, &clinical_models.MedicalRecord{},
		&clinical_models.Prescription{}, &clinical_models.MedicalCertificate{}, &clinical_models.MedicalReport{}, &clinical_models.ExamOrder{}))
	now := time.Now().UnixNano()
	tenant := func(i int) tenant_models.Tenant {
		tn := tenant_models.Tenant{Name: "C", Slug: fmt.Sprintf("bf-%d-%d", i, now), DisplayToken: fmt.Sprintf("tok-bf-%d-%d", i, now)}
		if err := db.Create(&tn).Error; err != nil {
			t.Fatal(err)
		}
		return tn
	}
	doctor := func(tenantID uint, name string) doctor_models.Doctor {
		d := doctor_models.Doctor{TenantID: tenantID, FullName: name, Specialty: "general_medicine", Active: true}
		if err := db.Create(&d).Error; err != nil {
			t.Fatal(err)
		}
		return d
	}

	single, multi := tenant(1), tenant(2)
	only := doctor(single.ID, "Única")
	doctor(multi.ID, "A")
	doctor(multi.ID, "B")
	singleRows := createSingleDoctorRows(t, db, single.ID)
	multiRows := createSingleDoctorRows(t, db, multi.ID)

	// A row that already has a doctor keeps it.
	other := doctor(single.ID+1000, "Otro tenant") // a doctor id that is not the tenant's only one
	keep := clinical_models.Appointment{TenantID: single.ID, PatientID: singleRows.patient, Title: "Ya asignada", Date: time.Now(), StartTime: "10:00", EndTime: "10:30", DoctorID: &other.ID}
	if err := db.Create(&keep).Error; err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 2; i++ { // idempotent
		if err := database.GlobalDBMap[backfillSingleDoctorID].Execute(db); err != nil {
			t.Fatalf("run %d: %v", i+1, err)
		}
	}

	checks := []struct {
		name  string
		model any
		id    func(singleDoctorRows) uint
	}{
		{"patient", &clinical_models.Patient{}, func(r singleDoctorRows) uint { return r.patient }},
		{"appointment", &clinical_models.Appointment{}, func(r singleDoctorRows) uint { return r.appointment }},
		{"record", &clinical_models.MedicalRecord{}, func(r singleDoctorRows) uint { return r.record }},
		{"prescription", &clinical_models.Prescription{}, func(r singleDoctorRows) uint { return r.prescription }},
		{"certificate", &clinical_models.MedicalCertificate{}, func(r singleDoctorRows) uint { return r.certificate }},
		{"report", &clinical_models.MedicalReport{}, func(r singleDoctorRows) uint { return r.report }},
		{"exam order", &clinical_models.ExamOrder{}, func(r singleDoctorRows) uint { return r.order }},
	}
	for _, ck := range checks {
		if got := doctorOf(t, db, ck.model, ck.id(singleRows)); got == nil || *got != only.ID {
			t.Errorf("single-doctor tenant %s: doctor_id = %v, want %d", ck.name, got, only.ID)
		}
		if got := doctorOf(t, db, ck.model, ck.id(multiRows)); got != nil {
			t.Errorf("multi-doctor tenant %s: doctor_id = %d, want null", ck.name, *got)
		}
	}
	if got := doctorOf(t, db, &clinical_models.Appointment{}, keep.ID); got == nil || *got != other.ID {
		t.Errorf("existing doctor_id overwritten: %v", got)
	}
}
