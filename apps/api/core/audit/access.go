package audit

import (
	"pengi-med-saas/core/tenantdb"
	"time"

	auth_middleware "pengi-med-saas/features/users/middleware"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RecordAccess logs a READ event for sensitive clinical data (view or export).
// GORM callbacks never fire on Find/First, so this must be called explicitly
// at the end of sensitive read handlers. Failures are always swallowed — a
// broken audit write must never block the clinical operation being audited.
func RecordAccess(db *gorm.DB, c *gin.Context, entityType string, entityID uint, patientID *uint) {
	tenantID := c.GetUint("tenant_id")
	userIDRaw, _, exists := auth_middleware.GetUserFromContext(c)
	if !exists {
		return
	}

	log := AuditLog{
		TenantID:   tenantID,
		UserID:     uint(userIDRaw),
		ActorType:  ActorUser,
		Action:     "READ",
		EntityType: entityType,
		EntityID:   entityID,
		PatientID:  patientID,
		CreatedAt:  time.Now(),
	}

	newDB := tenantdb.For(c, db).Session(&gorm.Session{NewDB: true})
	_ = newDB.Create(&log).Error
}

// RecordPortalAccess logs a READ event made by a patient through the patient
// portal: no staff user is involved, so UserID stays 0 and ActorType says who
// acted; PatientID is that patient. Failures are swallowed like RecordAccess.
func RecordPortalAccess(db *gorm.DB, tenantID, patientID uint, entityType string, entityID uint) {
	pid := patientID
	log := AuditLog{
		TenantID:   tenantID,
		ActorType:  ActorPortalPatient,
		Action:     "READ",
		EntityType: entityType,
		EntityID:   entityID,
		PatientID:  &pid,
		CreatedAt:  time.Now(),
	}
	_ = tenantdb.ForTenant(db, tenantID).Session(&gorm.Session{NewDB: true}).Create(&log).Error
}

// RecordPortalChange logs a change (action CREATE / UPDATE) a patient made
// through the patient portal: portal handlers write on a tenant handle
// without a staff user, so the audit plugin does not record them. Failures
// are swallowed like RecordAccess.
func RecordPortalChange(db *gorm.DB, tenantID, patientID uint, action, entityType string, entityID uint) {
	pid := patientID
	log := AuditLog{
		TenantID:   tenantID,
		ActorType:  ActorPortalPatient,
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		PatientID:  &pid,
		CreatedAt:  time.Now(),
	}
	_ = tenantdb.ForTenant(db, tenantID).Session(&gorm.Session{NewDB: true}).Create(&log).Error
}

// RecordPortalGuestChange logs a change made through the patient portal by a
// person without a patient record (a verified contact requesting a first
// appointment): like RecordPortalChange, with no PatientID.
func RecordPortalGuestChange(db *gorm.DB, tenantID uint, action, entityType string, entityID uint) {
	log := AuditLog{
		TenantID:   tenantID,
		ActorType:  ActorPortalPatient,
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		CreatedAt:  time.Now(),
	}
	_ = tenantdb.ForTenant(db, tenantID).Session(&gorm.Session{NewDB: true}).Create(&log).Error
}
