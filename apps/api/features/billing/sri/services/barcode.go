package services

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/png"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
)

// GenerateBarcodeBase64 encodes accessKey (the 49-digit clave de acceso) into a Code128
// barcode PNG and returns it base64-encoded, ready to embed as
// <img src="data:image/png;base64,...">. Required on the RIDE.
func GenerateBarcodeBase64(accessKey string) (string, error) {
	// Create the barcode
	bc, err := code128.Encode(accessKey)
	if err != nil {
		return "", fmt.Errorf("failed to create barcode: %w", err)
	}

	// Get the original width (1 pixel per module) and scale by exactly 4x
	// This guarantees that every individual bar is exactly 4 pixels thick,
	// preventing uneven scaling or squished bars.
	baseWidth := bc.Bounds().Max.X
	scaled, err := barcode.Scale(bc, baseWidth*4, 300)
	if err != nil {
		return "", fmt.Errorf("failed to scale barcode: %w", err)
	}

	// Encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, scaled); err != nil {
		return "", fmt.Errorf("failed to encode barcode to png: %w", err)
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
