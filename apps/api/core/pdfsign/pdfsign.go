// Package pdfsign signs PDFs (PAdES) with a user's PKCS#12 certificate and
// builds the visible FirmaEC-style stamp (QR + signer name) drawn on the
// document before it is signed.
package pdfsign

import (
	"bytes"
	"crypto"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/digitorus/pdf"
	"github.com/digitorus/pdfsign/sign"
	"github.com/skip2/go-qrcode"
	"software.sslmate.com/src/go-pkcs12"
)

// ErrInvalidCertificate means the P12 could not be decoded: wrong password or
// a corrupted/unsupported file.
var ErrInvalidCertificate = errors.New("pdfsign: invalid certificate or password")

// ErrWrongPassword means the P12 is readable but the password does not open
// it. It is also an ErrInvalidCertificate, for callers that don't tell them apart.
var ErrWrongPassword = fmt.Errorf("%w: wrong password", ErrInvalidCertificate)

// ErrExpiredCertificate means the certificate's validity period has ended.
var ErrExpiredCertificate = errors.New("pdfsign: certificate expired")

// ErrCertificateNotYetValid means the certificate's validity has not started.
var ErrCertificateNotYetValid = errors.New("pdfsign: certificate not yet valid")

// Certificate is a decoded P12: the signing key plus its certificate chain.
type Certificate struct {
	Cert  *x509.Certificate
	Chain []*x509.Certificate
	key   crypto.Signer
}

// Load decodes a P12. It tries DecodeChain first (intermediate CAs, legacy
// ciphers — what Ecuadorian CAs issue) and falls back to the strict Decode,
// same as the SRI signature upload.
func Load(p12 []byte, password string) (*Certificate, error) {
	key, cert, chain, err := pkcs12.DecodeChain(p12, password)
	if err != nil {
		chainErr := err
		key, cert, err = pkcs12.Decode(p12, password)
		chain = nil
		if err != nil {
			if errors.Is(chainErr, pkcs12.ErrIncorrectPassword) || errors.Is(err, pkcs12.ErrIncorrectPassword) {
				return nil, fmt.Errorf("%w: %v", ErrWrongPassword, err)
			}
			return nil, fmt.Errorf("%w: %v", ErrInvalidCertificate, err)
		}
	}
	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, fmt.Errorf("%w: private key cannot sign", ErrInvalidCertificate)
	}
	return &Certificate{Cert: cert, Chain: chain, key: signer}, nil
}

// SubjectName is the certificate holder's name (CN), the name printed on the stamp.
func (c *Certificate) SubjectName() string {
	if c.Cert.Subject.CommonName != "" {
		return c.Cert.Subject.CommonName
	}
	return c.Cert.Subject.String()
}

// SubjectSerial is the holder's serial number attribute (the cédula on
// Ecuadorian certificates), empty when absent.
func (c *Certificate) SubjectSerial() string { return c.Cert.Subject.SerialNumber }

// Issuer is the issuing CA's name.
func (c *Certificate) Issuer() string {
	if c.Cert.Issuer.CommonName != "" {
		return c.Cert.Issuer.CommonName
	}
	return c.Cert.Issuer.String()
}

// NotAfter is when the certificate expires.
func (c *Certificate) NotAfter() time.Time { return c.Cert.NotAfter }

// ValidAt reports ErrCertificateNotYetValid or ErrExpiredCertificate when t is
// outside the validity period.
func (c *Certificate) ValidAt(t time.Time) error {
	if t.Before(c.Cert.NotBefore) {
		return ErrCertificateNotYetValid
	}
	if t.After(c.Cert.NotAfter) {
		return ErrExpiredCertificate
	}
	return nil
}

// Options describe the signature: shown on the stamp and stored in the PDF
// signature dictionary.
type Options struct {
	Reason   string
	Location string
	SignedAt time.Time
}

// Stamp is the visible signature block the templates render as `.Signature`.
type Stamp struct {
	Name     string
	SignedAt string
	Lines    []string
	QR       template.URL // PNG data URI
}

// validationURL is where FirmaEC-signed documents are validated.
const validationURL = "www.firmadigital.gob.ec"

// NewStamp builds the FirmaEC-style stamp: a QR carrying the signer data plus
// the text lines printed beside it.
func NewStamp(c *Certificate, opts Options) (*Stamp, error) {
	name := c.SubjectName()
	signedAt := opts.SignedAt.Format("2006-01-02T15:04:05-07:00")
	qrText := strings.Join([]string{
		"FIRMADO POR: " + name,
		"RAZON: " + opts.Reason,
		"LOCALIZACION: " + opts.Location,
		"FECHA: " + signedAt,
		"VALIDAR CON: " + validationURL,
	}, "\n")
	png, err := qrcode.Encode(qrText, qrcode.Medium, 256)
	if err != nil {
		return nil, fmt.Errorf("pdfsign: qr: %w", err)
	}
	return &Stamp{
		Name:     name,
		SignedAt: opts.SignedAt.Format("02/01/2006 15:04"),
		Lines:    []string{"Validar únicamente con " + validationURL},
		QR:       template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(png)),
	}, nil
}

// Sign adds an approval signature (PAdES, SHA-256, full chain embedded) to pdfBytes.
func Sign(pdfBytes []byte, c *Certificate, opts Options) ([]byte, error) {
	input := bytes.NewReader(pdfBytes)
	rdr, err := pdf.NewReader(input, int64(len(pdfBytes)))
	if err != nil {
		return nil, fmt.Errorf("pdfsign: read pdf: %w", err)
	}

	chain := append([]*x509.Certificate{c.Cert}, c.Chain...)
	var out bytes.Buffer
	err = sign.Sign(input, &out, rdr, int64(len(pdfBytes)), sign.SignData{
		Signature: sign.SignDataSignature{
			Info: sign.SignDataSignatureInfo{
				Name:     c.SubjectName(),
				Location: opts.Location,
				Reason:   opts.Reason,
				Date:     opts.SignedAt,
			},
			CertType:   sign.ApprovalSignature,
			DocMDPPerm: sign.AllowFillingExistingFormFieldsAndSignaturesPerms,
		},
		Signer:            c.key,
		DigestAlgorithm:   crypto.SHA256,
		Certificate:       c.Cert,
		CertificateChains: [][]*x509.Certificate{chain},
	})
	if err != nil {
		return nil, fmt.Errorf("pdfsign: sign: %w", err)
	}
	return out.Bytes(), nil
}
