// Package doctor_data holds the fixed catalogs of the doctors domain: the
// specialties a doctor can have and the palette their agenda color comes from.
package doctor_data

import (
	"regexp"
	"strings"
)

// SpecialtyOther is the catch-all specialty; its name goes in SpecialtyOther.
const SpecialtyOther = "other"

// DefaultSpecialty is given to doctors created without one (data migration).
const DefaultSpecialty = "general_medicine"

// Specialties is the fixed specialty catalog, in display order. The label of
// each code is the i18n key SpecialtyLabelKey(code).
var Specialties = []string{
	"general_medicine",
	"family_medicine",
	"internal_medicine",
	"pediatrics",
	"gynecology_obstetrics",
	"cardiology",
	"dermatology",
	"endocrinology",
	"gastroenterology",
	"neurology",
	"ophthalmology",
	"otorhinolaryngology",
	"orthopedics_traumatology",
	"psychiatry",
	"psychology",
	"urology",
	"pulmonology",
	"oncology",
	"hematology",
	"rheumatology",
	"nephrology",
	"infectious_diseases",
	"geriatrics",
	"anesthesiology",
	"radiology",
	"emergency_medicine",
	"occupational_medicine",
	"sports_medicine",
	"nutrition",
	"physiotherapy",
	"dentistry",
	"general_surgery",
	"neurosurgery",
	"plastic_surgery",
	"vascular_surgery",
	"pediatric_surgery",
	SpecialtyOther,
}

// SpecialtyLabelKey is the i18n key of a specialty's label.
func SpecialtyLabelKey(code string) string {
	return "doctors.specialty." + code
}

// IsValidSpecialty reports whether code is in the catalog.
func IsValidSpecialty(code string) bool {
	for _, s := range Specialties {
		if s == code {
			return true
		}
	}
	return false
}

// Palette is the set of agenda colors assigned to new doctors, in order.
var Palette = []string{
	"#2563EB", // blue
	"#16A34A", // green
	"#DC2626", // red
	"#9333EA", // purple
	"#EA580C", // orange
	"#0891B2", // cyan
	"#DB2777", // pink
	"#CA8A04", // amber
	"#4F46E5", // indigo
	"#0D9488", // teal
	"#65A30D", // lime
	"#7C3AED", // violet
}

var hexColor = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// IsValidColor reports whether color is a #RRGGBB hex color.
func IsValidColor(color string) bool {
	return hexColor.MatchString(color)
}

// NextColor returns the first palette color not in used (case-insensitive),
// or cycles through the palette when every color is taken.
func NextColor(used []string) string {
	taken := make(map[string]bool, len(used))
	for _, c := range used {
		taken[strings.ToUpper(c)] = true
	}
	for _, c := range Palette {
		if !taken[c] {
			return c
		}
	}
	return Palette[len(used)%len(Palette)]
}
