package portal_services

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"pengi-med-saas/core/config"
	"pengi-med-saas/core/tenantdb"
	portal_models "pengi-med-saas/features/portal/models"

	"gorm.io/gorm"
)

// Sign-in code rules.
const (
	CodeLength = 6
	CodeTTL    = 10 * time.Minute
	// MaxAttempts is how many times a code can be tried, the right try included.
	MaxAttempts = 5
	// MaxCodesPerWindow codes per patient within CodeWindow; more requests
	// are answered the same but send nothing.
	MaxCodesPerWindow = 3
	CodeWindow        = 15 * time.Minute
)

// Cost ceiling of sign-in codes (each one is a paid email or WhatsApp
// message). Counted from portal_login_codes rows created in the window, any
// channel, so they survive restarts. "Day" is the last 24 hours. Over any of
// them nothing is sent and the login answers as usual.
const (
	MaxCodesPerPatientPerDay = 6
	MaxCodesPerTenantPerHour = 30
	MaxCodesPerTenantPerDay  = 150
	// The global daily ceiling, across every clinic, is
	// config.PortalMaxCodesPerDay (PORTAL_MAX_CODES_PER_DAY, default 500).
	BudgetDay = 24 * time.Hour
)

// Budgets a code counts toward (LimitError.Budget, logs).
const (
	BudgetPatientWindow = "patient_window" // MaxCodesPerWindow / CodeWindow
	BudgetPatientDay    = "patient_day"    // MaxCodesPerPatientPerDay
	BudgetTenantHour    = "tenant_hour"    // MaxCodesPerTenantPerHour
	BudgetTenantDay     = "tenant_day"     // MaxCodesPerTenantPerDay
	BudgetGlobalDay     = "global_day"     // PORTAL_MAX_CODES_PER_DAY
)

// LimitError is a code not issued because Budget is spent. It matches
// ErrTooManyCodes with errors.Is.
type LimitError struct{ Budget string }

func (e *LimitError) Error() string { return "portal code budget reached: " + e.Budget }
func (e *LimitError) Unwrap() error { return ErrTooManyCodes }

// PatientBudget reports whether one patient's budget is spent (normal use,
// not abuse of the clinic).
func (e *LimitError) PatientBudget() bool {
	return e.Budget == BudgetPatientWindow || e.Budget == BudgetPatientDay
}

// TenantBudget reports whether the clinic's own budget (not one patient's or
// the global one) is spent: staff are told about it.
func (e *LimitError) TenantBudget() bool {
	return e.Budget == BudgetTenantHour || e.Budget == BudgetTenantDay
}

// issueMu serializes the budget check and the insert, so parallel requests
// can't overshoot a budget (one API process in production).
var issueMu sync.Mutex

var (
	// ErrInvalidCode covers every verification failure (unknown, wrong,
	// expired, used, too many attempts): callers answer all of them the same.
	ErrInvalidCode = errors.New("invalid portal code")
	// ErrTooManyCodes: a budget is spent (see LimitError).
	ErrTooManyCodes = errors.New("too many portal codes requested")
)

// GenerateCode returns a uniformly random 6-digit code.
func GenerateCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// HashCode is the stored form of a code: an HMAC-SHA256 keyed with AUTH_KEY
// (the pepper) over tenant, patient and code, so a leaked table can't be
// brute-forced offline and a hash is useless for another patient.
func HashCode(tenantID, patientID uint, code string) string {
	mac := hmac.New(sha256.New, []byte("portal-code:"+config.Get().AuthKey))
	fmt.Fprintf(mac, "%d:%d:%s", tenantID, patientID, code)
	return hex.EncodeToString(mac.Sum(nil))
}

// IssueCode creates a new code for the patient over channel and expires the
// patient's previous unused codes. A *LimitError (errors.Is ErrTooManyCodes)
// when a budget is spent: the clinic's or the global one first, then the
// patient's.
func IssueCode(db *gorm.DB, tenantID, patientID uint, channel string, now time.Time) (string, portal_models.PortalLoginCode, error) {
	issueMu.Lock()
	defer issueMu.Unlock()
	scoped := tenantdb.ForTenant(db, tenantID)
	if budget, err := spentBudget(db, tenantID, patientID, now); err != nil {
		return "", portal_models.PortalLoginCode{}, err
	} else if budget != "" {
		return "", portal_models.PortalLoginCode{}, &LimitError{Budget: budget}
	}
	code, err := GenerateCode()
	if err != nil {
		return "", portal_models.PortalLoginCode{}, err
	}
	row := portal_models.PortalLoginCode{
		// UTC like every bound here: SQLite (CI) compares timestamps as text.
		Model:     gorm.Model{CreatedAt: now.UTC()},
		TenantID:  tenantID,
		PatientID: patientID,
		Channel:   channel,
		CodeHash:  HashCode(tenantID, patientID, code),
		ExpiresAt: now.Add(CodeTTL).UTC(),
	}
	err = scoped.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&portal_models.PortalLoginCode{}).
			Where("patient_id = ? AND used_at IS NULL AND expires_at > ?", patientID, now.UTC()).
			Update("expires_at", now.UTC()).Error; err != nil {
			return err
		}
		return tx.Create(&row).Error
	})
	if err != nil {
		return "", portal_models.PortalLoginCode{}, err
	}
	return code, row, nil
}

// spentBudget is the first budget the next code would exceed, "" if none.
func spentBudget(db *gorm.DB, tenantID, patientID uint, now time.Time) (string, error) {
	// UTC like every bound here: SQLite (CI) compares timestamps as text.
	dayAgo := now.Add(-BudgetDay).UTC()
	var global int64
	if err := tenantdb.System(db).Model(&portal_models.PortalLoginCode{}).
		Where("created_at > ?", dayAgo).Count(&global).Error; err != nil {
		return "", err
	}
	if global >= int64(config.Get().PortalMaxCodesPerDay) {
		return BudgetGlobalDay, nil
	}
	var counts struct {
		TenantDay     int64
		TenantHour    int64
		PatientDay    int64
		PatientWindow int64
	}
	if err := tenantdb.ForTenant(db, tenantID).Model(&portal_models.PortalLoginCode{}).
		Select(`COUNT(*) AS tenant_day,
			COALESCE(SUM(CASE WHEN created_at > ? THEN 1 ELSE 0 END), 0) AS tenant_hour,
			COALESCE(SUM(CASE WHEN patient_id = ? THEN 1 ELSE 0 END), 0) AS patient_day,
			COALESCE(SUM(CASE WHEN patient_id = ? AND created_at > ? THEN 1 ELSE 0 END), 0) AS patient_window`,
			now.Add(-time.Hour).UTC(), patientID, patientID, now.Add(-CodeWindow).UTC()).
		Where("created_at > ?", dayAgo).
		Scan(&counts).Error; err != nil {
		return "", err
	}
	switch {
	case counts.TenantDay >= MaxCodesPerTenantPerDay:
		return BudgetTenantDay, nil
	case counts.TenantHour >= MaxCodesPerTenantPerHour:
		return BudgetTenantHour, nil
	case counts.PatientDay >= MaxCodesPerPatientPerDay:
		return BudgetPatientDay, nil
	case counts.PatientWindow >= MaxCodesPerWindow:
		return BudgetPatientWindow, nil
	}
	return "", nil
}

// VerifyCode checks code against the patient's latest code and consumes it.
// Every try counts toward MaxAttempts, and the code works only once.
func VerifyCode(db *gorm.DB, tenantID, patientID uint, code string, now time.Time) error {
	code = strings.TrimSpace(code)
	if len(code) != CodeLength {
		return ErrInvalidCode
	}
	scoped := tenantdb.ForTenant(db, tenantID)
	var row portal_models.PortalLoginCode
	if err := scoped.Where("patient_id = ?", patientID).Order("id DESC").Limit(1).Find(&row).Error; err != nil {
		return err
	}
	if row.ID == 0 || row.UsedAt != nil || !now.Before(row.ExpiresAt) || row.Attempts >= MaxAttempts {
		return ErrInvalidCode
	}
	// Count the try first, atomically, so parallel guesses can't exceed the limit.
	res := scoped.Model(&portal_models.PortalLoginCode{}).
		Where("id = ? AND used_at IS NULL AND attempts < ?", row.ID, MaxAttempts).
		Update("attempts", gorm.Expr("attempts + 1"))
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return ErrInvalidCode
	}
	if !hmac.Equal([]byte(HashCode(tenantID, patientID, code)), []byte(row.CodeHash)) {
		return ErrInvalidCode
	}
	res = scoped.Model(&portal_models.PortalLoginCode{}).
		Where("id = ? AND used_at IS NULL", row.ID).
		Update("used_at", now.UTC())
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected != 1 {
		return ErrInvalidCode // consumed by a parallel request
	}
	return nil
}
