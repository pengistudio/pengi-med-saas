package audit

import (
	"encoding/json"
	"reflect"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func RegisterCallbacks(db *gorm.DB) {
	// After Create
	db.Callback().Create().After("gorm:create").Register("audit:after_create", afterCreateCallback)

	// Before Update (to capture old values)
	db.Callback().Update().Before("gorm:update").Register("audit:before_update", beforeUpdateCallback)

	// After Update
	db.Callback().Update().After("gorm:update").Register("audit:after_update", afterUpdateCallback)

	// After Delete
	db.Callback().Delete().After("gorm:delete").Register("audit:after_delete", afterDeleteCallback)
}

func getAuditContextData(db *gorm.DB) (userID uint, tenantID uint, hasContext bool) {
	uID, uOk := db.Get("audit_user_id")
	if !uOk {
		return 0, 0, false
	}
	tID, tOk := db.Get("audit_tenant_id")
	if !tOk {
		return 0, 0, false
	}

	// db.Get returns interface{}, we must cast
	var finalUserID, finalTenantID uint

	switch v := uID.(type) {
	case uint:
		finalUserID = v
	case int:
		finalUserID = uint(v)
	case int64:
		finalUserID = uint(v)
	case float64:
		finalUserID = uint(v)
	}

	switch v := tID.(type) {
	case uint:
		finalTenantID = v
	case int:
		finalTenantID = uint(v)
	case int64:
		finalTenantID = uint(v)
	case float64:
		finalTenantID = uint(v)
	}

	return finalUserID, finalTenantID, true
}

func recordAudit(db *gorm.DB, action string, oldValues, newValues interface{}) {
	if db.Statement.Error != nil {
		return
	}

	dest := db.Statement.Dest
	model := db.Statement.Model

	// Check if it's an Auditable model (check both Dest and Model)
	isAuditable := false
	if auditable, ok := dest.(Auditable); ok && auditable.IsAuditable() {
		isAuditable = true
	} else if model != nil {
		if auditable, ok := model.(Auditable); ok && auditable.IsAuditable() {
			isAuditable = true
		} else {
			val := reflect.ValueOf(model)
			if val.Kind() == reflect.Ptr && !val.IsNil() {
				if aud, ok := val.Interface().(Auditable); ok && aud.IsAuditable() {
					isAuditable = true
				}
			}
		}
	}

	if !isAuditable {
		// try pointer on Dest
		val := reflect.ValueOf(dest)
		if val.Kind() == reflect.Ptr && !val.IsNil() {
			if auditable, ok := val.Interface().(Auditable); ok && auditable.IsAuditable() {
				isAuditable = true
			}
		}
	}

	if !isAuditable {
		return
	}

	userID, tenantID, hasCtx := getAuditContextData(db)
	if !hasCtx {
		return // Not running in an HTTP context (e.g. background job/migration)
	}

	entityID := getEntityID(db)
	if entityID == 0 {
		if model != nil {
			mVal := reflect.ValueOf(model)
			if mVal.Kind() == reflect.Ptr {
				mVal = mVal.Elem()
			}
			if mVal.Kind() == reflect.Struct {
				if idField := mVal.FieldByName("ID"); idField.IsValid() {
					entityID = uint(idField.Uint())
				}
			}
		}

		if entityID == 0 {
			return // Couldn't find ID
		}
	}

	// We use the table name as entity type if not specified by another interface
	entityType := db.Statement.Table

	var oldJson, newJson datatypes.JSON

	if oldValues != nil {
		if b, err := json.Marshal(oldValues); err == nil {
			oldJson = datatypes.JSON(b)
		}
	}
	if newValues != nil {
		if b, err := json.Marshal(newValues); err == nil {
			newJson = datatypes.JSON(b)
		}
	}

	log := AuditLog{
		TenantID:   tenantID,
		UserID:     userID,
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		OldValues:  oldJson,
		NewValues:  newJson,
		CreatedAt:  time.Now(),
	}

	// Iniciar una nueva sesión de db sin callbacks para evitar bucles infinitos
	// si AuditLog llegase a ser auditable en el futuro
	newDB := db.Session(&gorm.Session{NewDB: true})
	_ = newDB.Create(&log).Error
}

func afterCreateCallback(db *gorm.DB) {
	recordAudit(db, "CREATE", nil, db.Statement.Dest)
}

func beforeUpdateCallback(db *gorm.DB) {
	if db.Statement.Error != nil {
		return
	}

	entityID := getEntityID(db)
	if entityID == 0 {
		return
	}

	// Read the old data from DB before it's overwritten
	tableName := db.Statement.Table
	if tableName == "" && db.Statement.Schema != nil {
		tableName = db.Statement.Schema.Table
	}

	if tableName == "" {
		return
	}

	// We use a fresh DB session without callbacks to avoid infinite loops,
	// and fetch the current row as a generic map
	var oldData map[string]interface{}
	err := db.Session(&gorm.Session{NewDB: true}).
		Table(tableName).
		Where("id = ?", entityID).
		Scan(&oldData).Error

	if err == nil && oldData != nil {
		// Store the old data in the GORM statement for the after_update callback to pick up
		db.Set("audit_old_values", oldData)
	}
}

func afterUpdateCallback(db *gorm.DB) {
	if db.Statement.Error != nil {
		return
	}

	var oldValues interface{}
	if val, ok := db.Get("audit_old_values"); ok {
		oldValues = val
	}

	valJSON := db.Statement.Dest
	if reflect.TypeOf(db.Statement.Dest).Kind() == reflect.Map {
		valJSON = map[string]interface{}{
			"updated_fields": db.Statement.Dest,
			"entity_id":      getEntityID(db),
		}
	}

	recordAudit(db, "UPDATE", oldValues, valJSON)
}

func afterDeleteCallback(db *gorm.DB) {
	recordAudit(db, "DELETE", db.Statement.Dest, nil)
}
