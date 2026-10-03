package secretbox_test

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"testing"

	"pengi-med-saas/core/secretbox"
)

func TestSealOpen_RoundTrip(t *testing.T) {
	box, err := secretbox.FromEnv()
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := box.Seal("p12-password")
	if err != nil {
		t.Fatal(err)
	}
	if sealed == "p12-password" {
		t.Fatal("sealed value is the plaintext")
	}
	got, err := box.Open(sealed)
	if err != nil || got != "p12-password" {
		t.Fatalf("Open = %q, %v", got, err)
	}
}

func TestOpen_WrongKeyFails(t *testing.T) {
	a, _ := secretbox.FromEnv()
	sealed, _ := a.Seal("secret")

	key := make([]byte, 32)
	_, _ = rand.Read(key)
	t.Setenv(secretbox.EnvKey, base64.StdEncoding.EncodeToString(key))
	b, err := secretbox.FromEnv()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.Open(sealed); err == nil {
		t.Fatal("Open with another key succeeded")
	}
}

func TestFromEnv_ReleaseWithoutKey(t *testing.T) {
	t.Setenv(secretbox.EnvKey, "")
	t.Setenv("GIN_MODE", "release")
	if _, err := secretbox.FromEnv(); !errors.Is(err, secretbox.ErrNoKey) {
		t.Fatalf("err = %v, want ErrNoKey", err)
	}
}

func TestBytes_NoAADMatchesPlainAEAD(t *testing.T) {
	key := make([]byte, 32)
	box, err := secretbox.New(key)
	if err != nil {
		t.Fatal(err)
	}
	// A value sealed by the raw AES-GCM construction the old code used.
	block, _ := aes.NewCipher(key)
	g, _ := cipher.NewGCM(block)
	nonce := make([]byte, g.NonceSize())
	legacy := g.Seal(nonce, nonce, []byte("old"), nil)
	if got, err := box.OpenBytes(legacy); err != nil || string(got) != "old" {
		t.Fatalf("legacy open = %q, %v", got, err)
	}
	if got, err := box.OpenBytesAAD(legacy, nil); err != nil || string(got) != "old" {
		t.Fatalf("legacy open (nil aad) = %q, %v", got, err)
	}
	sealed, _ := box.SealBytes([]byte("new"))
	if got, err := g.Open(nil, sealed[:g.NonceSize()], sealed[g.NonceSize():], nil); err != nil || string(got) != "new" {
		t.Fatalf("raw open of SealBytes = %q, %v", got, err)
	}
}

func TestBytesAAD_MismatchFails(t *testing.T) {
	box, err := secretbox.New(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := box.SealBytesAAD([]byte("x"), []byte("ctx-a"))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := box.OpenBytesAAD(sealed, []byte("ctx-a")); err != nil || string(got) != "x" {
		t.Fatalf("open = %q, %v", got, err)
	}
	if _, err := box.OpenBytesAAD(sealed, []byte("ctx-b")); err == nil {
		t.Fatal("wrong aad accepted")
	}
	if _, err := box.OpenBytes(sealed); err == nil {
		t.Fatal("missing aad accepted")
	}
}
