// Package secretbox encrypts small secrets at rest (e.g. P12 passwords) with
// AES-256-GCM. The key comes from SIGNATURE_ENCRYPTION_KEY (32 bytes, base64).
package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
)

// EnvKey is the environment variable holding the base64 encryption key.
const EnvKey = "SIGNATURE_ENCRYPTION_KEY"

// ErrNoKey means SIGNATURE_ENCRYPTION_KEY is unset in release mode.
var ErrNoKey = errors.New("secretbox: " + EnvKey + " is not set")

// devKeySeed derives the key used outside release when the env var is unset,
// so local dev works without setup. Never used with GIN_MODE=release.
const devKeySeed = "pengi-dev-only-signature-key"

// Box seals and opens secrets with one key.
type Box struct{ aead cipher.AEAD }

// New builds a Box from a 32-byte key.
func New(key []byte) (*Box, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("secretbox: key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Box{aead: aead}, nil
}

// FromEnv builds a Box from SIGNATURE_ENCRYPTION_KEY. Outside release mode a
// missing key falls back to a fixed dev key; in release it returns ErrNoKey.
func FromEnv() (*Box, error) {
	raw := os.Getenv(EnvKey)
	if raw == "" {
		if os.Getenv("GIN_MODE") == "release" {
			return nil, ErrNoKey
		}
		sum := sha256.Sum256([]byte(devKeySeed))
		return New(sum[:])
	}
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("secretbox: %s is not valid base64: %w", EnvKey, err)
	}
	return New(key)
}

// FromEnvVar builds a Box from the named variable (32 bytes, base64). Unlike
// FromEnv there is no dev fallback: a missing or invalid key is always an error.
func FromEnvVar(name string) (*Box, error) {
	raw := os.Getenv(name)
	if raw == "" {
		return nil, errors.New("secretbox: " + name + " is not set")
	}
	key, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("secretbox: %s is not valid base64: %w", name, err)
	}
	return New(key)
}

// SealBytes encrypts data and returns nonce || ciphertext.
func (b *Box) SealBytes(data []byte) ([]byte, error) { return b.SealBytesAAD(data, nil) }

// SealBytesAAD is SealBytes with additional authenticated data: the value only
// opens with the same aad. With nil aad the output format is that of SealBytes.
func (b *Box) SealBytesAAD(data, aad []byte) ([]byte, error) {
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return b.aead.Seal(nonce, nonce, data, aad), nil
}

// OpenBytes decrypts a value produced by SealBytes.
func (b *Box) OpenBytes(sealed []byte) ([]byte, error) { return b.OpenBytesAAD(sealed, nil) }

// OpenBytesAAD decrypts a value produced by SealBytesAAD with the same aad.
func (b *Box) OpenBytesAAD(sealed, aad []byte) ([]byte, error) {
	n := b.aead.NonceSize()
	if len(sealed) < n {
		return nil, errors.New("secretbox: ciphertext too short")
	}
	plain, err := b.aead.Open(nil, sealed[:n], sealed[n:], aad)
	if err != nil {
		return nil, fmt.Errorf("secretbox: %w", err)
	}
	return plain, nil
}

// Seal encrypts plaintext and returns base64(nonce || ciphertext).
func (b *Box) Seal(plaintext string) (string, error) {
	sealed, err := b.SealBytes([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Open decrypts a value produced by Seal.
func (b *Box) Open(sealed string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(sealed)
	if err != nil {
		return "", fmt.Errorf("secretbox: %w", err)
	}
	plain, err := b.OpenBytes(data)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
