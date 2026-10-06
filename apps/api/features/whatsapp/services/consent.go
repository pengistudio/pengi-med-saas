package whatsapp_services

import (
	"strings"
	"sync"

	clinical_models "pengi-med-saas/features/clinical/models"
	"pengi-med-saas/i18n/catalog"
	i18n_messages "pengi-med-saas/i18n/messages"

	"gorm.io/gorm"
)

// Keys of the free texts Pengi sends to patients on its own (KindSystem).
const (
	OptInRequestKey   = "whatsapp.patient.opt_in.request"
	OptInConfirmedKey = "whatsapp.patient.opt_in.confirmed"
)

// PatientLanguage is the language of the texts sent to patients.
const PatientLanguage = "es"

// optInYesKeywords are the answers that grant consent after Pengi asked.
var optInYesKeywords = map[string]bool{"SI": true, "SÍ": true, "ACEPTO": true, "OK": true}

// IsOptInYes reports whether body is a consent answer ("Sí", "si.", "OK!").
func IsOptInYes(body string) bool {
	s := strings.ToUpper(strings.Trim(strings.TrimSpace(body), ".!¡ "))
	return optInYesKeywords[strings.Join(strings.Fields(s), " ")]
}

var (
	patientCatalogOnce sync.Once
	patientCatalog     *catalog.Catalog
)

// PatientText is the text of key in PatientLanguage, from the message catalog
// embedded in the binary (the key itself if it can't be loaded).
func PatientText(key string) string {
	patientCatalogOnce.Do(func() {
		if c, err := catalog.Load(i18n_messages.FS); err == nil {
			patientCatalog = c
		}
	})
	if patientCatalog == nil {
		return key
	}
	return patientCatalog.Translate(PatientLanguage, key)
}

// PatientsByPhone returns the tenant's patients (db bound to the tenant) whose
// phone normalizes to phone. Phones are free text, so the match is done in Go.
func PatientsByPhone(db *gorm.DB, phone string) ([]clinical_models.Patient, error) {
	var patients []clinical_models.Patient
	if err := db.Select("id", "phone", "first_name", "last_name", "whatsapp_opt_in", "whatsapp_opt_in_source").
		Where("phone <> ''").Find(&patients).Error; err != nil {
		return nil, err
	}
	out := patients[:0]
	for _, p := range patients {
		if n, err := NormalizePhone(p.Phone); err == nil && n == phone {
			out = append(out, p)
		}
	}
	return out, nil
}
