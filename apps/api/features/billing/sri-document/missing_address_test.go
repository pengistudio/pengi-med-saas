package sri_document

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"pengi-med-saas/core/tenantdb"
	billing_models "pengi-med-saas/features/billing/models"
	tenant_models "pengi-med-saas/features/tenants/models"
	"pengi-med-saas/testutils"
)

const addressPlaceholder = "Dirección no provista"

func TestEnqueue_BlankAddress_IsRefusedAndDocumentUntouched(t *testing.T) {
	for _, address := range []string{"", "   ", " \t\n"} {
		for _, status := range []string{
			billing_models.InvoiceStatusDraft,
			billing_models.InvoiceStatusFailed,
			billing_models.InvoiceStatusConnectionError,
			billing_models.InvoiceStatusRejected,
		} {
			t.Run(fmt.Sprintf("%q/%s", address, status), func(t *testing.T) {
				f := newFixture(t)
				f.db.Model(&f.tenant).Update("address", address)
				inv := f.invoice(status)

				err := f.docs.Enqueue(f.tenantDB(f.tenant.ID), f.kind, uint64(inv.ID))
				if !errors.Is(err, ErrMissingAddress) {
					t.Fatalf("err = %v, want ErrMissingAddress", err)
				}
				if len(f.publisher.messages) != 0 {
					t.Fatalf("published = %v, want nothing queued", f.publisher.messages)
				}
				if got := f.reload(inv.ID).Status; got != status {
					t.Fatalf("status = %q, want unchanged %q", got, status)
				}
			})
		}
	}
}

func TestEnqueue_WithAddress_IsQueued(t *testing.T) {
	f := newFixture(t)
	inv := f.invoice(billing_models.InvoiceStatusDraft)

	if err := f.docs.Enqueue(f.tenantDB(f.tenant.ID), f.kind, uint64(inv.ID)); err != nil {
		t.Fatalf("Enqueue: %v", err)
	}
	if len(f.publisher.messages) != 1 {
		t.Fatalf("published = %v, want one message", f.publisher.messages)
	}
}

// The address was removed after the document was queued: the attempt fails with
// a clear code instead of sending a placeholder.
func TestProcess_BlankAddress_FailsThroughBuildErrorPath(t *testing.T) {
	f := newFixture(t)
	f.kind.BuildXML = Invoice.BuildXML
	f.db.Model(&f.tenant).Update("address", "  ")
	inv := f.invoice(billing_models.InvoiceStatusPending)

	if err := f.process(inv.ID); err == nil {
		t.Fatalf("process succeeded without an address")
	}
	got := f.reload(inv.ID)
	if got.Status != billing_models.InvoiceStatusFailed || got.ErrorCode == nil || *got.ErrorCode != ErrorCodeMissingAddress {
		t.Fatalf("status=%q error_code=%v, want failed/%s", got.Status, got.ErrorCode, ErrorCodeMissingAddress)
	}
	if f.gateway.signCalls != 0 || f.gateway.receptionCalls != 0 {
		t.Fatalf("a document without address was signed or sent")
	}
}

func TestBuildXML_BlankAddress_FailsForEveryKind(t *testing.T) {
	db := testutils.SetupTestDB(t, &tenant_models.Tenant{})
	now := time.Now().UnixNano()
	for _, address := range []string{"", "  "} {
		tenant := tenant_models.Tenant{Name: "T", Slug: fmt.Sprintf("na-%d", now), DisplayToken: fmt.Sprintf("tok-na-%d", now), Address: address}
		for _, kind := range []Kind{Invoice, CreditNote, DebitNote} {
			xml, err := kind.BuildXML(tenantdb.System(db), 1, tenant, "key", "1")
			if !errors.Is(err, ErrMissingAddress) {
				t.Fatalf("%s: err = %v, want ErrMissingAddress", kind.Name, err)
			}
			if strings.Contains(xml, addressPlaceholder) {
				t.Fatalf("%s: XML carries the placeholder", kind.Name)
			}
		}
	}
}

func TestEstablishmentAddress_NeverInventsText(t *testing.T) {
	if got := establishmentAddress(tenant_models.Tenant{Address: "  "}); got != "" {
		t.Fatalf("got %q, want empty", got)
	}
	if got := establishmentAddress(tenant_models.Tenant{Address: " Av. 1 "}); got != "Av. 1" {
		t.Fatalf("got %q, want trimmed address", got)
	}
}
