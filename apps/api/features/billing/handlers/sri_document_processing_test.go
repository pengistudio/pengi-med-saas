package billing_handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantfiles"
	billing_models "pengi-med-saas/features/billing/models"
	sri_document "pengi-med-saas/features/billing/sri-document"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"
)

type recordingPublisher struct{ published int }

func (p *recordingPublisher) Publish(queue string, body []byte) error {
	p.published++
	return nil
}

// expiredSignatureSetup creates a tenant whose P12 has expired, one invoice per
// status, and an invoice handler over a lifecycle that records publications.
func expiredSignatureSetup(t *testing.T, statuses ...string) (*InvoiceHandler, *recordingPublisher, uint, []uint) {
	t.Helper()
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{}, &billing_models.Invoice{})
	now := time.Now().UnixNano()
	expired := time.Now().Add(-time.Hour)
	tenant := tenant_models.Tenant{
		Name:              "Clinic",
		Slug:              fmt.Sprintf("sri-exp-%d", now),
		DisplayToken:      fmt.Sprintf("tok-sri-exp-%d", now),
		SriP12Path:        "signature.p12",
		SriPassword:       "secret",
		SriCertExpiration: &expired,
	}
	if err := db.Create(&tenant).Error; err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	var ids []uint
	for i, status := range statuses {
		inv := billing_models.Invoice{TenantID: tenant.ID, Sequential: fmt.Sprintf("%09d", i+1), Status: status}
		if err := db.Create(&inv).Error; err != nil {
			t.Fatalf("create invoice: %v", err)
		}
		ids = append(ids, inv.ID)
	}

	publisher := &recordingPublisher{}
	docs := sri_document.New(db, zap.NewNop(), nil, publisher, sri_document.Documents{Files: tenantfiles.Memory()}, "1")
	return NewInvoiceHandler(db, zap.NewNop(), docs), publisher, tenant.ID, ids
}

func assertSignatureExpired(t *testing.T, resp envelope.Response) {
	t.Helper()
	if resp.Code != http.StatusBadRequest || resp.Message != "billing.sri.error.signature_expired" {
		t.Fatalf("code=%d message=%q, want 400 billing.sri.error.signature_expired", resp.Code, resp.Message)
	}
	if appErr, _ := resp.Data.(core_errors.AppError); appErr.ErrorCode != core_errors.ErrBillingSignatureExpired.ErrorCode {
		t.Fatalf("data = %#v, want %s", resp.Data, core_errors.ErrBillingSignatureExpired.ErrorCode)
	}
}

func TestSRIInvoiceProcessing_ExpiredSignatureIsRefused(t *testing.T) {
	h, publisher, tenantID, ids := expiredSignatureSetup(t, billing_models.InvoiceStatusFailed)

	c, _ := testutils.NewGinContext(tenantID, 1)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprint(ids[0])}}

	assertSignatureExpired(t, h.SRIInvoiceProcessing(c))
	if publisher.published != 0 {
		t.Fatalf("an invoice was queued with an expired signature")
	}
}

func TestMultipleSRIInvoiceProcessing_ExpiredSignatureOnlyQueuesReceivedInvoices(t *testing.T) {
	h, publisher, tenantID, ids := expiredSignatureSetup(t, billing_models.InvoiceStatusDraft, billing_models.InvoiceStatusValidated)

	body, _ := json.Marshal(map[string]any{"id_list": ids})
	c, _ := testutils.NewGinContext(tenantID, 1)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")

	assertSignatureExpired(t, h.MultipleSRIInvoiceProcessing(c))
	if publisher.published != 1 {
		t.Fatalf("published = %d, want only the received invoice queued", publisher.published)
	}
}
