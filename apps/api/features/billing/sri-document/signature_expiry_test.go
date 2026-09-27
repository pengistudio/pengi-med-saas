package sri_document

import (
	"errors"
	"testing"
	"time"

	billing_models "pengi-med-saas/features/billing/models"
)

// The tenant's P12 was valid when uploaded but its certificate has since expired.
func (f *fixture) expireSignature() {
	f.t.Helper()
	if err := f.db.Model(&f.tenant).Update("sri_cert_expiration", time.Now().Add(-time.Hour)).Error; err != nil {
		f.t.Fatalf("expire signature: %v", err)
	}
}

// The tenant uploads a new, valid P12.
func (f *fixture) renewSignature() {
	f.t.Helper()
	if err := f.db.Model(&f.tenant).Update("sri_cert_expiration", time.Now().Add(365*24*time.Hour)).Error; err != nil {
		f.t.Fatalf("renew signature: %v", err)
	}
}

const existingKey = "0109202601179001122300110010010000001231234567810"

// A document queued (or requeued by Sweep) before the certificate expired must
// not be signed or sent with it: it fails, keeps its key, and is not retried
// automatically — the user retries after uploading a new signature.
func TestProcess_ExpiredSignature_FailsWithoutSigningAndKeepsKey(t *testing.T) {
	f := newFixture(t)
	f.expireSignature()
	inv := f.invoice(billing_models.InvoiceStatusPending)
	f.db.Model(&inv).Update("access_key", existingKey)

	err := f.process(inv.ID)
	if err == nil || IsRetryable(err) {
		t.Fatalf("err = %v, want a non-retryable error", err)
	}
	got := f.reload(inv.ID)
	if got.Status != billing_models.InvoiceStatusFailed || got.ErrorCode == nil || *got.ErrorCode != ErrorCodeSignatureExpired {
		t.Fatalf("status=%q error_code=%v, want failed/%s", got.Status, got.ErrorCode, ErrorCodeSignatureExpired)
	}
	if got.AccessKey == nil || *got.AccessKey != existingKey {
		t.Fatalf("access key = %v, want the original %s", got.AccessKey, existingKey)
	}
	if f.gateway.signCalls != 0 || f.gateway.receptionCalls != 0 {
		t.Fatalf("sign=%d reception=%d, want nothing signed or sent with an expired certificate", f.gateway.signCalls, f.gateway.receptionCalls)
	}
}

func TestProcess_ExpiredSignature_NewDocumentGetsNoKey(t *testing.T) {
	f := newFixture(t)
	f.expireSignature()
	inv := f.invoice(billing_models.InvoiceStatusPending)

	if err := f.process(inv.ID); err == nil {
		t.Fatalf("process succeeded with an expired certificate")
	}
	if got := f.reload(inv.ID); got.AccessKey != nil {
		t.Fatalf("access key = %s, want none generated for an attempt that cannot be signed", *got.AccessKey)
	}
}

// A received comprobante is already in the SRI's hands: querying its
// authorization needs no signature, so an expired one must not block it.
func TestProcess_ExpiredSignature_ReceivedDocumentStillGetsAuthorized(t *testing.T) {
	f := newFixture(t)
	f.expireSignature()
	inv := f.invoice(billing_models.InvoiceStatusValidated)
	f.db.Model(&inv).Update("access_key", existingKey)

	if err := f.process(inv.ID); err != nil {
		t.Fatalf("process: %v", err)
	}
	if got := f.reload(inv.ID); got.Status != billing_models.InvoiceStatusAuthorized {
		t.Fatalf("status = %q, want authorized", got.Status)
	}
	if f.gateway.signCalls != 0 || f.gateway.receptionCalls != 0 {
		t.Fatalf("a received document was re-signed or resent")
	}
}

func TestProcess_UnknownExpiration_IsNotTreatedAsExpired(t *testing.T) {
	f := newFixture(t) // fixture tenant has no sri_cert_expiration
	inv := f.invoice(billing_models.InvoiceStatusPending)

	if err := f.process(inv.ID); err != nil {
		t.Fatalf("process: %v", err)
	}
	if got := f.reload(inv.ID); got.Status != billing_models.InvoiceStatusAuthorized {
		t.Fatalf("status = %q, want authorized", got.Status)
	}
}

func TestEnqueue_ExpiredSignature_IsRefusedAndDocumentUntouched(t *testing.T) {
	for _, status := range []string{
		billing_models.InvoiceStatusDraft,
		billing_models.InvoiceStatusPending,
		billing_models.InvoiceStatusFailed,
		billing_models.InvoiceStatusConnectionError,
		billing_models.InvoiceStatusRejected,
	} {
		t.Run(status, func(t *testing.T) {
			f := newFixture(t)
			f.expireSignature()
			inv := f.invoice(status)

			err := f.docs.Enqueue(f.tenantDB(f.tenant.ID), f.kind, uint64(inv.ID))
			if !errors.Is(err, ErrSignatureExpired) {
				t.Fatalf("err = %v, want ErrSignatureExpired", err)
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

func TestEnqueue_ExpiredSignature_ReceivedDocumentIsStillQueued(t *testing.T) {
	f := newFixture(t)
	f.expireSignature()
	inv := f.invoice(billing_models.InvoiceStatusValidated)

	if err := f.docs.Enqueue(f.tenantDB(f.tenant.ID), f.kind, uint64(inv.ID)); err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	if len(f.publisher.messages) != 1 {
		t.Fatalf("published = %v, want one task", f.publisher.messages)
	}
}

// End to end: queued while valid, the certificate expires before the worker
// runs, the user uploads a new one and retries — same key, authorized.
func TestExpiredSignature_RetryAfterRenewalResendsWithTheSameKey(t *testing.T) {
	f := newFixture(t)
	f.renewSignature()
	inv := f.invoice(billing_models.InvoiceStatusDraft)
	f.db.Model(&inv).Update("access_key", existingKey)
	if err := f.docs.Enqueue(f.tenantDB(f.tenant.ID), f.kind, uint64(inv.ID)); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	f.expireSignature()
	if err := f.process(inv.ID); err == nil {
		t.Fatalf("process succeeded with an expired certificate")
	}

	f.renewSignature()
	if err := f.docs.Enqueue(f.tenantDB(f.tenant.ID), f.kind, uint64(inv.ID)); err != nil {
		t.Fatalf("enqueue after renewal: %v", err)
	}
	if err := f.process(inv.ID); err != nil {
		t.Fatalf("process after renewal: %v", err)
	}
	got := f.reload(inv.ID)
	if got.Status != billing_models.InvoiceStatusAuthorized || *got.AccessKey != existingKey {
		t.Fatalf("status=%q key=%v, want authorized with the original key", got.Status, *got.AccessKey)
	}
}
