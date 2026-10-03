package tenantfiles_test

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"testing"

	"pengi-med-saas/core/secretbox"
	"pengi-med-saas/core/tenantfiles"
)

func newBox(t *testing.T) *secretbox.Box {
	t.Helper()
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	box, err := secretbox.New(key)
	if err != nil {
		t.Fatal(err)
	}
	return box
}

func TestEncrypted_RoundTripAndStoredBytesDiffer(t *testing.T) {
	inner := tenantfiles.Memory()
	s := tenantfiles.Encrypted(inner, newBox(t))
	plain := []byte("%PDF-1.7 historia clinica")
	if err := s.Write(1, "attachments/a.pdf", plain); err != nil {
		t.Fatal(err)
	}
	got, err := s.Read(1, "attachments/a.pdf")
	if err != nil || !bytes.Equal(got, plain) {
		t.Fatalf("read = %q, %v", got, err)
	}
	raw, err := inner.Read(1, "attachments/a.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(raw, plain) || bytes.Contains(raw, plain) {
		t.Fatal("inner store holds the plaintext")
	}
}

func TestEncrypted_TamperedContentFails(t *testing.T) {
	inner := tenantfiles.Memory()
	s := tenantfiles.Encrypted(inner, newBox(t))
	if err := s.Write(1, "a.bin", []byte("secret")); err != nil {
		t.Fatal(err)
	}
	raw, _ := inner.Read(1, "a.bin")
	raw[len(raw)-1] ^= 0xff
	if err := inner.Write(1, "a.bin", raw); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read(1, "a.bin"); err == nil {
		t.Fatal("tampered content read without error")
	}
}

func TestEncrypted_WrongKeyFails(t *testing.T) {
	inner := tenantfiles.Memory()
	if err := tenantfiles.Encrypted(inner, newBox(t)).Write(1, "a.bin", []byte("secret")); err != nil {
		t.Fatal(err)
	}
	if _, err := tenantfiles.Encrypted(inner, newBox(t)).Read(1, "a.bin"); err == nil {
		t.Fatal("read with a different key succeeded")
	}
}

func TestEncrypted_TenantsDoNotSeeEachOthersFiles(t *testing.T) {
	s := tenantfiles.Encrypted(tenantfiles.Memory(), newBox(t))
	if err := s.Write(1, "a.bin", []byte("secret")); err != nil {
		t.Fatal(err)
	}
	if s.Exists(2, "a.bin") {
		t.Fatal("tenant 2 sees tenant 1's file")
	}
	if _, err := s.Read(2, "a.bin"); err == nil {
		t.Fatal("tenant 2 read tenant 1's file")
	}
}

func TestEncrypted_RemoveAndExistsDelegate(t *testing.T) {
	inner := tenantfiles.Memory()
	s := tenantfiles.Encrypted(inner, newBox(t))
	if err := s.Write(1, "a.bin", []byte("x")); err != nil {
		t.Fatal(err)
	}
	if !s.Exists(1, "a.bin") || !inner.Exists(1, "a.bin") {
		t.Fatal("Exists does not reflect the inner store")
	}
	if err := s.Remove(1, "a.bin"); err != nil {
		t.Fatal(err)
	}
	if s.Exists(1, "a.bin") || inner.Exists(1, "a.bin") {
		t.Fatal("Remove did not delete from the inner store")
	}
}

func TestEncryptedFromEnv(t *testing.T) {
	t.Setenv(tenantfiles.AttachmentKeyEnv, "")
	if _, err := tenantfiles.EncryptedFromEnv(tenantfiles.Memory()); err == nil {
		t.Fatal("missing key accepted")
	}
	t.Setenv(tenantfiles.AttachmentKeyEnv, "not base64!")
	if _, err := tenantfiles.EncryptedFromEnv(tenantfiles.Memory()); err == nil {
		t.Fatal("invalid base64 accepted")
	}
	t.Setenv(tenantfiles.AttachmentKeyEnv, base64.StdEncoding.EncodeToString([]byte("short")))
	if _, err := tenantfiles.EncryptedFromEnv(tenantfiles.Memory()); err == nil {
		t.Fatal("short key accepted")
	}
	t.Setenv(tenantfiles.AttachmentKeyEnv, base64.StdEncoding.EncodeToString(make([]byte, 32)))
	if _, err := tenantfiles.EncryptedFromEnv(tenantfiles.Memory()); err != nil {
		t.Fatalf("valid key rejected: %v", err)
	}
}

func TestEncrypted_BlobBoundToTenantAndName(t *testing.T) {
	inner := tenantfiles.Memory()
	s := tenantfiles.Encrypted(inner, newBox(t))
	if err := s.Write(1, "a.bin", []byte("secret")); err != nil {
		t.Fatal(err)
	}
	raw, err := inner.Read(1, "a.bin")
	if err != nil {
		t.Fatal(err)
	}
	if err := inner.Write(2, "a.bin", raw); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read(2, "a.bin"); err == nil {
		t.Fatal("blob copied to another tenant decrypted")
	}
	if err := inner.Write(1, "b.bin", raw); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read(1, "b.bin"); err == nil {
		t.Fatal("blob copied to another name decrypted")
	}
	// Equivalent spelling of the same name still matches.
	if got, err := s.Read(1, "./a.bin"); err != nil || string(got) != "secret" {
		t.Fatalf("equivalent name read = %q, %v", got, err)
	}
}
