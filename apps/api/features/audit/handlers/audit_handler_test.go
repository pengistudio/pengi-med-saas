package audit_handlers

import (
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"pengi-med-saas/core/audit"
	"pengi-med-saas/core/envelope"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"

	"go.uber.org/zap"
)

func TestGetAuditLogs_TenantIsolation(t *testing.T) {
	db := testutils.SetupTestDB(t,
		&tenant_models.Tenant{},
		&audit.AuditLog{},
	)
	logger := zap.NewNop()

	now := time.Now().UnixNano() % 1000000
	tenant1 := &tenant_models.Tenant{
		Name:         "Audit Tenant 1",
		Slug:         fmt.Sprintf("aud-log-t1-%d", now),
		DisplayToken: fmt.Sprintf("aud-log-tok1-%d", now),
	}
	tenant2 := &tenant_models.Tenant{
		Name:         "Audit Tenant 2",
		Slug:         fmt.Sprintf("aud-log-t2-%d", now),
		DisplayToken: fmt.Sprintf("aud-log-tok2-%d", now),
	}
	if err := db.Create([]*tenant_models.Tenant{tenant1, tenant2}).Error; err != nil {
		t.Fatalf("failed to create test tenants: %v", err)
	}

	log1 := &audit.AuditLog{TenantID: tenant1.ID, UserID: 1, Action: "READ", EntityType: "patients", EntityID: 1, CreatedAt: time.Now()}
	log2 := &audit.AuditLog{TenantID: tenant2.ID, UserID: 2, Action: "READ", EntityType: "patients", EntityID: 2, CreatedAt: time.Now()}
	if err := db.Create([]*audit.AuditLog{log1, log2}).Error; err != nil {
		t.Fatalf("failed to seed audit logs: %v", err)
	}

	handler := NewAuditLogHandler(db, logger)

	c1, _ := testutils.NewGinContext(tenant1.ID, 1)
	c1.Request = httptest.NewRequest("GET", "/audit/logs", nil)
	response1 := handler.GetAuditLogs(c1)
	if response1.Code != 200 {
		t.Fatalf("expected status 200, got %d", response1.Code)
	}
	paged1, ok := response1.Data.(envelope.PagedData)
	if !ok {
		t.Fatalf("expected PagedData response, got %T", response1.Data)
	}
	if paged1.Total != 1 {
		t.Errorf("expected 1 audit log for tenant1, got %d", paged1.Total)
	}

	c2, _ := testutils.NewGinContext(tenant2.ID, 1)
	c2.Request = httptest.NewRequest("GET", "/audit/logs", nil)
	response2 := handler.GetAuditLogs(c2)
	if response2.Code != 200 {
		t.Fatalf("expected status 200, got %d", response2.Code)
	}
	paged2, ok := response2.Data.(envelope.PagedData)
	if !ok {
		t.Fatalf("expected PagedData response, got %T", response2.Data)
	}
	if paged2.Total != 1 {
		t.Errorf("expected 1 audit log for tenant2, got %d", paged2.Total)
	}
}
