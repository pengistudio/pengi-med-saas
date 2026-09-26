// Package tenantdb isolates tenants at the data layer (docs/adr/0002). A GORM
// plugin adds tenant_id = ? to every query, update and delete on a model with a
// TenantID field, and stamps TenantID on create, for the tenant bound to the
// statement's context by For. Handlers never filter by tenant themselves.
//
//	db := tenantdb.For(c, h.db)  // bound to the request's tenant
//	db.First(&patient, id)       // another tenant's patient: ErrRecordNotFound
//
// Work that spans tenants (workers, schedulers, seeds) opts out with System.
// Tables without tenant_id inherit their parent's tenant and are only reached
// through a scoped parent.
package tenantdb

import (
	"context"
	"errors"
	"reflect"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

var (
	// ErrNoTenant: a tenant table was queried with neither a tenant (For) nor
	// System in the context, while strict mode is on.
	ErrNoTenant = errors.New("tenantdb: no tenant bound to this query")
	// ErrTenantMismatch: a row explicitly set to another tenant was created.
	ErrTenantMismatch = errors.New("tenantdb: row belongs to another tenant")
)

type tenantKey struct{}
type systemKey struct{}

// strict makes unbound queries on tenant tables fail. Off during the staged
// migration of ADR 0002; switched on once every call site goes through For.
var strict atomic.Bool

// SetStrict switches strict mode and returns a func restoring the previous one.
func SetStrict(on bool) (restore func()) {
	prev := strict.Swap(on)
	return func() { strict.Store(prev) }
}

// TenantID is the tenant TenantMiddleware resolved for this request, or 0.
func TenantID(c *gin.Context) uint {
	return c.GetUint("tenant_id")
}

// For binds db to the request's tenant: every statement built from it is
// filtered to, and stamped with, that tenant. It also carries the audit
// metadata the audit plugin records.
func For(c *gin.Context, db *gorm.DB) *gorm.DB {
	ctx := context.Background()
	if c.Request != nil {
		ctx = c.Request.Context()
	}
	tenantID := TenantID(c)
	if tenantID != 0 {
		ctx = context.WithValue(ctx, tenantKey{}, tenantID)
	}
	userID, _ := c.Get("user_id")
	return db.WithContext(ctx).
		Set("audit_tenant_id", tenantID).
		Set("audit_user_id", userID)
}

// ForTenant binds db to one tenant outside a request, for background work on
// that tenant's data (e.g. syncing its calendar after the request ended).
func ForTenant(db *gorm.DB, tenantID uint) *gorm.DB {
	return db.WithContext(context.WithValue(context.Background(), tenantKey{}, tenantID)).
		Set("audit_tenant_id", tenantID)
}

// System opts db out of tenant isolation, for work that spans every tenant.
func System(db *gorm.DB) *gorm.DB {
	return db.WithContext(context.WithValue(db.Statement.Context, systemKey{}, true))
}

// Register installs the isolation callbacks on db.
func Register(db *gorm.DB) error {
	cb := db.Callback()
	return errors.Join(
		cb.Query().Before("gorm:query").Register("tenantdb:scope_query", scope),
		cb.Row().Before("gorm:row").Register("tenantdb:scope_row", scope),
		cb.Update().Before("gorm:update").Register("tenantdb:scope_update", scope),
		cb.Delete().Before("gorm:delete").Register("tenantdb:scope_delete", scope),
		cb.Create().Before("gorm:create").Register("tenantdb:stamp_create", stamp),
	)
}

// binding reports how a statement relates to tenants: which tenant it is bound
// to, whether it is a system statement, and the model's tenant field (nil for
// shared tables, which are left alone).
func binding(db *gorm.DB) (tenantID uint, system bool, field *schema.Field) {
	if db.Statement.Schema == nil {
		return 0, false, nil
	}
	f := db.Statement.Schema.LookUpField("TenantID")
	if f == nil {
		return 0, false, nil
	}
	ctx := db.Statement.Context
	if ctx != nil {
		if on, _ := ctx.Value(systemKey{}).(bool); on {
			return 0, true, f
		}
		tenantID, _ = ctx.Value(tenantKey{}).(uint)
	}
	return tenantID, false, f
}

func scope(db *gorm.DB) {
	if db.Error != nil {
		return
	}
	tenantID, system, f := binding(db)
	if f == nil || system {
		return
	}
	if tenantID == 0 {
		if strict.Load() {
			_ = db.AddError(ErrNoTenant)
		}
		return
	}
	db.Statement.AddClause(clause.Where{Exprs: []clause.Expression{
		clause.Eq{Column: clause.Column{Table: clause.CurrentTable, Name: f.DBName}, Value: tenantID},
	}})
}

func stamp(db *gorm.DB) {
	if db.Error != nil {
		return
	}
	tenantID, system, f := binding(db)
	if f == nil || system {
		return
	}
	if tenantID == 0 {
		if strict.Load() {
			_ = db.AddError(ErrNoTenant)
		}
		return
	}
	stampOne := func(row reflect.Value) {
		value, zero := f.ValueOf(db.Statement.Context, row)
		if zero {
			_ = f.Set(db.Statement.Context, row, tenantID)
			return
		}
		if value != any(tenantID) {
			_ = db.AddError(ErrTenantMismatch)
		}
	}
	switch rv := db.Statement.ReflectValue; rv.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < rv.Len(); i++ {
			stampOne(reflect.Indirect(rv.Index(i)))
		}
	case reflect.Struct:
		stampOne(rv)
	}
}
