package pdfsign_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/digitorus/pdfsign/verify"

	"pengi-med-saas/core/pdfsign"
	"pengi-med-saas/core/pdfsign/pdfsigntest"
)

func TestLoad_WrongPasswordIsWrongPassword(t *testing.T) {
	p12 := pdfsigntest.P12(t, "DR TEST", "secret", time.Now().Add(24*time.Hour))
	if _, err := pdfsign.Load(p12, "wrong"); !errors.Is(err, pdfsign.ErrWrongPassword) {
		t.Fatalf("err = %v, want ErrWrongPassword", err)
	}
}

func TestLoad_CorruptedFileIsInvalidCertificate(t *testing.T) {
	_, err := pdfsign.Load([]byte("not a p12 file"), "secret")
	if !errors.Is(err, pdfsign.ErrInvalidCertificate) || errors.Is(err, pdfsign.ErrWrongPassword) {
		t.Fatalf("err = %v, want ErrInvalidCertificate only", err)
	}
}

func TestValidAt_BeforeNotBeforeIsNotYetValid(t *testing.T) {
	p12 := pdfsigntest.P12(t, "DR TEST", "secret", time.Now().Add(24*time.Hour))
	cert, err := pdfsign.Load(p12, "secret")
	if err != nil {
		t.Fatal(err)
	}
	if err := cert.ValidAt(time.Now().Add(-72 * time.Hour)); !errors.Is(err, pdfsign.ErrCertificateNotYetValid) {
		t.Fatalf("ValidAt before NotBefore = %v, want ErrCertificateNotYetValid", err)
	}
}

func TestLoad_ExposesSubject(t *testing.T) {
	p12 := pdfsigntest.P12(t, "DR TEST", "secret", time.Now().Add(24*time.Hour))
	cert, err := pdfsign.Load(p12, "secret")
	if err != nil {
		t.Fatal(err)
	}
	if cert.SubjectName() != "DR TEST" || cert.SubjectSerial() != "0912345678" {
		t.Fatalf("subject = %q / %q", cert.SubjectName(), cert.SubjectSerial())
	}
	if err := cert.ValidAt(time.Now().Add(48 * time.Hour)); !errors.Is(err, pdfsign.ErrExpiredCertificate) {
		t.Fatalf("ValidAt after expiry = %v, want ErrExpiredCertificate", err)
	}
}

func TestSign_ProducesVerifiableSignature(t *testing.T) {
	p12 := pdfsigntest.P12(t, "DR TEST", "secret", time.Now().Add(24*time.Hour))
	cert, err := pdfsign.Load(p12, "secret")
	if err != nil {
		t.Fatal(err)
	}

	signed, err := pdfsign.Sign(pdfsigntest.PDF(), cert, pdfsign.Options{
		Reason: "Informe médico", Location: "Ecuador", SignedAt: time.Now(),
	})
	if err != nil {
		t.Fatalf("sign: %v", err)
	}

	resp, err := verify.Verify(bytes.NewReader(signed), int64(len(signed)))
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if len(resp.Signers) != 1 || !resp.Signers[0].ValidSignature {
		t.Fatalf("signers = %+v, want one valid signature", resp.Signers)
	}
}

func TestNewStamp_CarriesSignerAndQR(t *testing.T) {
	p12 := pdfsigntest.P12(t, "DR TEST", "secret", time.Now().Add(24*time.Hour))
	cert, _ := pdfsign.Load(p12, "secret")
	stamp, err := pdfsign.NewStamp(cert, pdfsign.Options{SignedAt: time.Date(2026, 9, 26, 10, 30, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	if stamp.Name != "DR TEST" || stamp.SignedAt != "26/09/2026 10:30" {
		t.Fatalf("stamp = %+v", stamp)
	}
	if !strings.HasPrefix(string(stamp.QR), "data:image/png;base64,") {
		t.Fatalf("QR is not a PNG data URI")
	}
}
