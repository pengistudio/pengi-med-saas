package ai_models

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Functions: what an AI use was for. Each has its own monthly quota.
const (
	FunctionSummary = "summary" // patient history summary
	FunctionScribe  = "scribe"  // consultation notes into S/O/A/P (next PR)
	FunctionBot     = "bot"     // WhatsApp bot conversation (later)
)

// Functions lists every function, in the order backoffice shows them.
var Functions = []string{FunctionSummary, FunctionScribe, FunctionBot}

// ValidFunction reports whether fn is a known function.
func ValidFunction(fn string) bool {
	for _, f := range Functions {
		if f == fn {
			return true
		}
	}
	return false
}

// AIUsageLog is one charged AI use: a call that reached the model and
// answered. It holds who, what for, the model and its tokens and cost, never
// the prompt or the answer. Rows count against the plan's monthly quota;
// tokens and cost are shown only in backoffice.
type AIUsageLog struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	TenantID         uint      `gorm:"not null;index:idx_ai_usage_tenant_fn_created" json:"tenant_id"`
	UserID           uint      `gorm:"not null" json:"user_id"`
	Function         string    `gorm:"type:varchar(20);not null;index:idx_ai_usage_tenant_fn_created" json:"function"`
	Model            string    `gorm:"not null" json:"model"`
	ProviderHost     string    `gorm:"type:varchar(255);not null;default:''" json:"provider_host"` // e.g. inference.do-ai.run
	PromptTokens     int       `gorm:"not null;default:0" json:"prompt_tokens"`
	CompletionTokens int       `gorm:"not null;default:0" json:"completion_tokens"`
	CostUSD          float64   `gorm:"not null;default:0" json:"cost_usd"`
	CreatedAt        time.Time `gorm:"not null;index:idx_ai_usage_tenant_fn_created" json:"created_at"`
}

// AISummary is the cached AI summary of a patient's history, one per
// patient. It is valid while SourceFingerprint matches the patient's current
// clinical data (records, exam orders, documents, background); a mismatch
// means something changed and the next request regenerates it. It is an AI
// draft, never part of the medical record: not printed, signed or exported.
type AISummary struct {
	gorm.Model
	TenantID          uint           `gorm:"not null;uniqueIndex:idx_ai_summary_tenant_patient" json:"tenant_id"`
	PatientID         uint           `gorm:"not null;uniqueIndex:idx_ai_summary_tenant_patient" json:"patient_id"`
	SourceFingerprint string         `gorm:"type:varchar(64);not null" json:"-"`
	Content           datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"content"`
	AIModel           string         `gorm:"column:model;not null" json:"model"`
	GeneratedByUserID uint           `json:"generated_by_user_id"`
}
