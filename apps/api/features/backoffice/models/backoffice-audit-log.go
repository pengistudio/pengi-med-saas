package backoffice_models

import "time"

// BackofficeAuditLog is one entry of the backoffice audit trail: a request an
// admin made that changes something (every non-read request under
// /backoffice/* by a signed-in admin, plus sign-ins and sign-outs), whatever
// its outcome. It records who, what route and the result, never the request
// body: bodies carry passwords, codes and API keys.
//
// Rows are only ever inserted; UserName is a snapshot, so the entry still
// names the admin after the account is renamed or deleted.
type BackofficeAuditLog struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	CreatedAt        time.Time `gorm:"index" json:"created_at"`
	BackofficeUserID uint      `gorm:"index" json:"backoffice_user_id"`
	UserName         string    `gorm:"size:255" json:"user_name"`
	Method           string    `gorm:"size:10" json:"method"`
	// Route is the registered pattern (/backoffice/plans/:id) and Path the
	// URL requested (/backoffice/plans/3): the first says what kind of change
	// it was, the second on which record.
	Route  string `gorm:"size:255" json:"route"`
	Path   string `gorm:"size:255" json:"path"`
	Status int    `json:"status"`
	IP     string `gorm:"size:64" json:"ip"`
	// UserAgent is cut to 255 characters.
	UserAgent string `gorm:"size:255" json:"user_agent"`
}
