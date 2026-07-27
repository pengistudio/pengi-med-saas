package services

import (
	"bytes"
	"encoding/base64"
	"fmt"

	qrcode "github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard"
)

// bufferWriteCloser adapts a bytes.Buffer to io.WriteCloser so the QR writer can
// render in-memory instead of to a file on disk.
type bufferWriteCloser struct {
	*bytes.Buffer
}

func (bufferWriteCloser) Close() error { return nil }

// GenerateQRCodeBase64 encodes accessKey (the 49-digit clave de acceso) into a QR
// code PNG and returns it base64-encoded, ready to embed as
// <img src="data:image/png;base64,...">. Required on the RIDE per Anexo 7 of the
// SRI's ficha técnica.
func GenerateQRCodeBase64(accessKey string) (string, error) {
	qrc, err := qrcode.New(accessKey)
	if err != nil {
		return "", fmt.Errorf("failed to create QR code: %w", err)
	}

	buf := &bufferWriteCloser{Buffer: &bytes.Buffer{}}
	writer := standard.NewWithWriter(buf, standard.WithQRWidth(10), standard.WithBuiltinImageEncoder(standard.PNG_FORMAT))

	if err := qrc.Save(writer); err != nil {
		return "", fmt.Errorf("failed to render QR code: %w", err)
	}

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
