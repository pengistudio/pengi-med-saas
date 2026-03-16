package clinical_models

// Cie10Code stores CIE-10 / ICD-10 diagnostic codes.
// The table is populated via the seed migration at startup.
type Cie10Code struct {
	Code  string `gorm:"primaryKey;not null" json:"code"`
	Title string `gorm:"not null"           json:"title"`
}
