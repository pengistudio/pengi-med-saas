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
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"

	"pengi-med-saas/core/logger"

	"go.uber.org/zap"
)

var (
	// ErrNoTenant: a tenant table was queried with neither a tenant (For) nor
	// System in the context, in Strict mode.
	ErrNoTenant = errors.New("tenantdb: no tenant bound to this query")
	// ErrTenantMismatch: a row explicitly set to another tenant was created.
	ErrTenantMismatch = errors.New("tenantdb: row belongs to another tenant")
)

type tenantKey struct{}
type systemKey struct{}

// Mode is what happens to a statement on a tenant table bound to neither a
// tenant (For/ForTenant) nor System.
type Mode int32

const (
	// Permissive runs it unfiltered, as before ADR 0002.
	Permissive Mode = iota
	// Warn runs it unfiltered and reports it (table and caller), so remaining
	// unbound call sites can be found in logs before going strict.
	Warn
	// Strict fails it with ErrNoTenant.
	Strict
)

var mode atomic.Int32

func init() { mode.Store(int32(Warn)) }

// SetMode switches the mode and returns a func restoring the previous one.
func SetMode(m Mode) (restore func()) {
	prev := mode.Swap(int32(m))
	return func() { mode.Store(prev) }
}

// ParseMode maps "permissive", "warn" or "strict" to a Mode; anything else is Warn.
func ParseMode(s string) Mode {
	switch s {
	case "permissive":
		return Permissive
	case "strict":
		return Strict
	default:
		return Warn
	}
}

var reporter atomic.Pointer[func(table, caller string)]

func init() {
	report := func(table, caller string) {
		if logger.Log != nil {
			logger.Log.Warn("tenantdb: unbound query on tenant table", zap.String("table", table), zap.String("caller", caller))
		}
	}
	reporter.Store(&report)
}

// SetReporter replaces how Warn mode reports unbound statements (the app logger
// by default) and returns a func restoring the previous reporter.
func SetReporter(report func(table, caller string)) (restore func()) {
	prev := reporter.Swap(&report)
	return func() { reporter.Store(prev) }
}

// unbound applies the mode to a statement bound to neither a tenant nor System.
func unbound(db *gorm.DB) {
	switch Mode(mode.Load()) {
	case Strict:
		_ = db.AddError(ErrNoTenant)
	case Warn:
		(*reporter.Load())(db.Statement.Table, caller())
	}
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
	// Session() makes the handle reusable: handlers run several statements on it.
	return db.WithContext(ctx).
		Set("audit_tenant_id", tenantID).
		Set("audit_user_id", userID).
		Session(&gorm.Session{})
}

// ForTenant binds db to one tenant outside a request, for background work on
// that tenant's data (e.g. syncing its calendar after the request ended).
func ForTenant(db *gorm.DB, tenantID uint) *gorm.DB {
	return db.WithContext(context.WithValue(context.Background(), tenantKey{}, tenantID)).
		Set("audit_tenant_id", tenantID).
		Session(&gorm.Session{})
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
		unbound(db)
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
		unbound(db)
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

// caller is the file:line of the first frame outside GORM and this plugin: the
// code that built the unbound statement.
func caller() string {
	pcs := make([]uintptr, 32)
	frames := runtime.CallersFrames(pcs[:runtime.Callers(2, pcs)])
	for {
		frame, more := frames.Next()
		if !strings.Contains(frame.File, "gorm.io/") && !strings.HasSuffix(frame.File, "core/tenantdb/tenantdb.go") {
			return fmt.Sprintf("%s:%d", frame.File, frame.Line)
		}
		if !more {
			return "unknown"
		}
	}
}
