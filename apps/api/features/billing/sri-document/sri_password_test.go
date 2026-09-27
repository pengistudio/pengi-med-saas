package sri_document

import (
	"crypto/rand"
	"encoding/base64"
	"testing"

	"pengi-med-saas/core/secretbox"
	billing_models "pengi-med-saas/features/billing/models"

	"go.uber.org/zap"
)

// passwordGateway records the password each Sign call received.
type passwordGateway struct {
	*fakeGateway
	passwords []string
}

func (g *passwordGateway) Sign(p12 []byte, password string, xml string) (string, error) {
	g.passwords = append(g.passwords, password)
	return g.fakeGateway.Sign(p12, password, xml)
}

// withSriPassword stores the tenant's SRI password as the columns given and
// rebuilds the lifecycle over a gateway that records the signing password.
func (f *fixture) withSriPassword(plain, sealed string) *passwordGateway {
	f.t.Helper()
	if err := f.db.Model(&f.tenant).UpdateColumns(map[string]any{
		"sri_password":           plain,
		"sri_password_encrypted": sealed,
	}).Error; err != nil {
		f.t.Fatalf("set SRI password: %v", err)
	}
	gw := &passwordGateway{fakeGateway: f.gateway}
	f.docs = New(f.db, zap.NewNop(), gw, f.publisher, Documents{Files: f.files}, "1")
	return gw
}

func seal(t *testing.T, plain string) string {
	t.Helper()
	box, err := secretbox.FromEnv()
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := box.Seal(plain)
	if err != nil {
		t.Fatal(err)
	}
	return sealed
}

func TestProcess_SealedPassword_IsOpenedBeforeSigning(t *testing.T) {
	f := newFixture(t)
	gw := f.withSriPassword("", seal(t, "p12-secret"))
	inv := f.invoice(billing_models.InvoiceStatusPending)

	if err := f.process(inv.ID); err != nil {
		t.Fatalf("process: %v", err)
	}
	if len(gw.passwords) != 1 || gw.passwords[0] != "p12-secret" {
		t.Fatalf("signed with %q, want the opened password", gw.passwords)
	}
	if got := f.reload(inv.ID); got.Status != billing_models.InvoiceStatusAuthorized {
		t.Fatalf("status = %q, want authorized", got.Status)
	}
}

// A row the migration has not sealed yet (e.g. written by an older instance
// during a deploy) still signs with its plaintext password.
func TestProcess_LegacyPlaintextPassword_StillSigns(t *testing.T) {
	f := newFixture(t)
	gw := f.withSriPassword("legacy", "")
	inv := f.invoice(billing_models.InvoiceStatusPending)

	if err := f.process(inv.ID); err != nil {
		t.Fatalf("process: %v", err)
	}
	if len(gw.passwords) != 1 || gw.passwords[0] != "legacy" {
		t.Fatalf("signed with %q, want legacy", gw.passwords)
	}
}

// A sealed password that does not open (other key, corrupt value) must never
// reach the signer: the document fails, not retried, with a specific code.
func TestProcess_UnopenableSealedPassword_FailsWithoutSigning(t *testing.T) {
	otherKey := make([]byte, 32)
	_, _ = rand.Read(otherKey)
	t.Setenv(secretbox.EnvKey, base64.StdEncoding.EncodeToString(otherKey))
	sealedWithOtherKey := seal(t, "p12-secret")
	t.Setenv(secretbox.EnvKey, "") // back to the dev key

	f := newFixture(t)
	gw := f.withSriPassword("", sealedWithOtherKey)
	inv := f.invoice(billing_models.InvoiceStatusPending)
	f.db.Model(&inv).Update("access_key", existingKey)

	err := f.process(inv.ID)
	if err == nil || IsRetryable(err) {
		t.Fatalf("err = %v, want a non-retryable error", err)
	}
	got := f.reload(inv.ID)
	if got.Status != billing_models.InvoiceStatusFailed || got.ErrorCode == nil || *got.ErrorCode != ErrorCodeSignaturePassword {
		t.Fatalf("status=%q error_code=%v, want failed/%s", got.Status, got.ErrorCode, ErrorCodeSignaturePassword)
	}
	if got.AccessKey == nil || *got.AccessKey != existingKey {
		t.Fatalf("access key = %v, want the original %s", got.AccessKey, existingKey)
	}
	if len(gw.passwords) != 0 || f.gateway.receptionCalls != 0 {
		t.Fatalf("sign=%d reception=%d, want nothing signed or sent", len(gw.passwords), f.gateway.receptionCalls)
	}
}

// In release without SIGNATURE_ENCRYPTION_KEY a sealed password cannot be
// opened: the document fails with the "signature unavailable" code.
func TestProcess_SealedPasswordWithoutKey_FailsWithoutSigning(t *testing.T) {
	sealed := seal(t, "p12-secret")
	t.Setenv("GIN_MODE", "release")
	t.Setenv(secretbox.EnvKey, "")

	f := newFixture(t)
	gw := f.withSriPassword("", sealed)
	inv := f.invoice(billing_models.InvoiceStatusPending)

	err := f.process(inv.ID)
	if err == nil || IsRetryable(err) {
		t.Fatalf("err = %v, want a non-retryable error", err)
	}
	got := f.reload(inv.ID)
	if got.Status != billing_models.InvoiceStatusFailed || got.ErrorCode == nil || *got.ErrorCode != ErrorCodeSignatureUnavailable {
		t.Fatalf("status=%q error_code=%v, want failed/%s", got.Status, got.ErrorCode, ErrorCodeSignatureUnavailable)
	}
	if len(gw.passwords) != 0 {
		t.Fatalf("signed %d times, want none", len(gw.passwords))
	}
}

func TestProcess_NoPasswordAtAll_IsMissingSignature(t *testing.T) {
	f := newFixture(t)
	gw := f.withSriPassword("", "")
	inv := f.invoice(billing_models.InvoiceStatusPending)

	if err := f.process(inv.ID); err == nil {
		t.Fatalf("process succeeded without a password")
	}
	got := f.reload(inv.ID)
	if got.ErrorCode == nil || *got.ErrorCode != ErrorCodeMissingSignature {
		t.Fatalf("error_code = %v, want %s", got.ErrorCode, ErrorCodeMissingSignature)
	}
	if len(gw.passwords) != 0 {
		t.Fatalf("signed without a password")
	}
}
