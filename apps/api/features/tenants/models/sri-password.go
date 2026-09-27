package tenant_models

import (
	"errors"

	"pengi-med-saas/core/secretbox"
)

// The SRI P12 password lives sealed (core/secretbox, SIGNATURE_ENCRYPTION_KEY)
// in SriPasswordEncrypted. SriPassword is the legacy plaintext column: current
// code only ever clears it, so a non-empty value was written by an older
// instance after the last seal and is the current password until the
// DB20260927_5 migration (or a new upload) seals it.

// ErrNoSriPassword means the tenant has no SRI signature password stored.
var ErrNoSriPassword = errors.New("tenant has no SRI signature password")

// HasSriPassword reports whether a P12 password is stored, sealed or legacy.
func (t *Tenant) HasSriPassword() bool {
	return t.SriPasswordEncrypted != "" || t.SriPassword != ""
}

// SealSriPassword stores plain sealed with box and clears the legacy column.
func (t *Tenant) SealSriPassword(box *secretbox.Box, plain string) error {
	sealed, err := box.Seal(plain)
	if err != nil {
		return err
	}
	t.SriPasswordEncrypted = sealed
	t.SriPassword = ""
	return nil
}

// OpenSriPassword returns the P12 password to sign with. A legacy plaintext
// value wins (see above); otherwise the sealed one is opened with the key from
// the environment, returning secretbox.ErrNoKey when it is not set.
func (t *Tenant) OpenSriPassword() (string, error) {
	if t.SriPassword != "" {
		return t.SriPassword, nil
	}
	if t.SriPasswordEncrypted == "" {
		return "", ErrNoSriPassword
	}
	box, err := secretbox.FromEnv()
	if err != nil {
		return "", err
	}
	return box.Open(t.SriPasswordEncrypted)
}
