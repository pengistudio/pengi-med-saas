package services

import (
	"encoding/base64"
	"testing"
)

func TestGenerateBarcodeBase64(t *testing.T) {
	accessKey := "2607202601179000000000110010010000000011855268517"

	b64, err := GenerateBarcodeBase64(accessKey)
	if err != nil {
		t.Fatalf("GenerateBarcodeBase64 failed: %v", err)
	}
	if b64 == "" {
		t.Fatal("expected non-empty base64 string")
	}

	decoded, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatalf("failed to decode base64: %v", err)
	}

	// PNG signature: 89 50 4E 47 0D 0A 1A 0A
	pngSignature := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	if len(decoded) < len(pngSignature) {
		t.Fatalf("decoded barcode image too short: %d bytes", len(decoded))
	}
	for i, b := range pngSignature {
		if decoded[i] != b {
			t.Fatalf("decoded image does not start with PNG signature, got %x", decoded[:8])
		}
	}
}
