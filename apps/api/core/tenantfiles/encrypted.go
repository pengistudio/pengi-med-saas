package tenantfiles

import (
	"fmt"

	"pengi-med-saas/core/secretbox"
)

// AttachmentKeyEnv is the environment variable holding the base64 key that
// encrypts patient attachments (separate from SIGNATURE_ENCRYPTION_KEY).
const AttachmentKeyEnv = "ATTACHMENT_ENCRYPTION_KEY"

// EncryptedStore wraps a Store so everything it holds is encrypted at rest.
type EncryptedStore struct {
	inner Store
	box   *secretbox.Box
}

// Encrypted encrypts on Write and decrypts on Read; Remove and Exists go
// straight to inner.
func Encrypted(inner Store, box *secretbox.Box) *EncryptedStore {
	return &EncryptedStore{inner: inner, box: box}
}

// EncryptedFromEnv is Encrypted keyed by ATTACHMENT_ENCRYPTION_KEY. It fails
// when the key is missing or invalid, in every mode.
func EncryptedFromEnv(inner Store) (*EncryptedStore, error) {
	box, err := secretbox.FromEnvVar(AttachmentKeyEnv)
	if err != nil {
		return nil, err
	}
	return Encrypted(inner, box), nil
}

// aad binds a blob to its tenant and canonical name, so a ciphertext copied to
// another tenant or file name no longer decrypts.
func aad(tenantID uint, name string) ([]byte, error) {
	c, err := clean(name)
	if err != nil {
		return nil, err
	}
	return []byte(fmt.Sprintf("tenantfiles/v1/%d/%s", tenantID, c)), nil
}

func (e *EncryptedStore) Write(tenantID uint, name string, data []byte) error {
	ad, err := aad(tenantID, name)
	if err != nil {
		return err
	}
	sealed, err := e.box.SealBytesAAD(data, ad)
	if err != nil {
		return err
	}
	return e.inner.Write(tenantID, name, sealed)
}

// Read fails if the stored bytes were not written with this key for this
// tenant and name, or were altered.
func (e *EncryptedStore) Read(tenantID uint, name string) ([]byte, error) {
	ad, err := aad(tenantID, name)
	if err != nil {
		return nil, err
	}
	sealed, err := e.inner.Read(tenantID, name)
	if err != nil {
		return nil, err
	}
	data, err := e.box.OpenBytesAAD(sealed, ad)
	if err != nil {
		return nil, fmt.Errorf("tenantfiles: decrypt %q: %w", name, err)
	}
	return data, nil
}

func (e *EncryptedStore) Remove(tenantID uint, name string) error {
	return e.inner.Remove(tenantID, name)
}

func (e *EncryptedStore) Exists(tenantID uint, name string) bool {
	return e.inner.Exists(tenantID, name)
}
