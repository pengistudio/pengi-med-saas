package secretbox_test

import (
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
