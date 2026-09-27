// Package pdfsigntest builds throwaway P12 certificates and PDFs for tests.
package pdfsigntest

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"testing"
	"time"

	"software.sslmate.com/src/go-pkcs12"
)

// P12 returns a self-signed P12 for commonName valid until notAfter.
func P12(t testing.TB, commonName, password string, notAfter time.Time) []byte {
	t.Helper()
	return P12With(t, Cert{CommonName: commonName, SubjectSerial: "0912345678"}, password, notAfter)
}

// Cert describes the holder data of a test certificate, laid out the way
// Ecuadorian CAs carry it.
type Cert struct {
	CommonName string
	// SubjectSerial is the subject serialNumber attribute (2.5.4.5); empty omits it.
	SubjectSerial string
	// OrganizationIdentifier is the subject organizationIdentifier (2.5.4.97),
	// e.g. "VATEC-1790012345001"; empty omits it.
	OrganizationIdentifier string
	// Extensions are private X.509 extensions, OID → UTF8String value
	// (e.g. BCE "1.3.6.1.4.1.37947.3.11" → RUC).
	Extensions map[string]string
	// SANOtherNames are subjectAltName otherName entries, OID → UTF8String
	// value (how Uanataca, Eclipsoft and the Consejo de la Judicatura carry them).
	SANOtherNames map[string]string
}

var oidOrganizationIdentifier = asn1.ObjectIdentifier{2, 5, 4, 97}
var oidSubjectAltName = asn1.ObjectIdentifier{2, 5, 29, 17}

// P12With returns a self-signed P12 carrying the holder data in c.
func P12With(t testing.TB, c Cert, password string, notAfter time.Time) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	subject := pkix.Name{CommonName: c.CommonName, SerialNumber: c.SubjectSerial}
	if c.OrganizationIdentifier != "" {
		subject.ExtraNames = append(subject.ExtraNames, pkix.AttributeTypeAndValue{Type: oidOrganizationIdentifier, Value: c.OrganizationIdentifier})
	}
	var exts []pkix.Extension
	for oid, value := range c.Extensions {
		exts = append(exts, pkix.Extension{Id: parseOID(t, oid), Value: utf8String(t, value)})
	}
	if len(c.SANOtherNames) > 0 {
		exts = append(exts, pkix.Extension{Id: oidSubjectAltName, Value: otherNames(t, c.SANOtherNames)})
	}
	tmpl := &x509.Certificate{
		SerialNumber:    big.NewInt(time.Now().UnixNano()),
		Subject:         subject,
		Issuer:          pkix.Name{CommonName: "Test CA"},
		NotBefore:       time.Now().Add(-48 * time.Hour),
		NotAfter:        notAfter,
		KeyUsage:        x509.KeyUsageDigitalSignature | x509.KeyUsageContentCommitment,
		ExtraExtensions: exts,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	p12, err := pkcs12.Modern.Encode(key, cert, nil, password)
	if err != nil {
		t.Fatal(err)
	}
	return p12
}

func parseOID(t testing.TB, s string) asn1.ObjectIdentifier {
	t.Helper()
	var oid asn1.ObjectIdentifier
	for _, part := range strings.Split(s, ".") {
		n, err := strconv.Atoi(part)
		if err != nil {
			t.Fatalf("bad OID %q: %v", s, err)
		}
		oid = append(oid, n)
	}
	return oid
}

func utf8String(t testing.TB, s string) []byte {
	t.Helper()
	der, err := asn1.MarshalWithParams(s, "utf8")
	if err != nil {
		t.Fatal(err)
	}
	return der
}

// otherNames encodes GeneralNames holding one otherName per entry:
// [0] IMPLICIT SEQUENCE { type-id OID, value [0] EXPLICIT UTF8String }.
func otherNames(t testing.TB, names map[string]string) []byte {
	t.Helper()
	var seq []byte
	for oid, value := range names {
		oidDER, err := asn1.Marshal(parseOID(t, oid))
		if err != nil {
			t.Fatal(err)
		}
		valueDER, err := asn1.Marshal(asn1.RawValue{Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true, Bytes: utf8String(t, value)})
		if err != nil {
			t.Fatal(err)
		}
		entry, err := asn1.Marshal(asn1.RawValue{Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true, Bytes: append(oidDER, valueDER...)})
		if err != nil {
			t.Fatal(err)
		}
		seq = append(seq, entry...)
	}
	der, err := asn1.Marshal(asn1.RawValue{Class: asn1.ClassUniversal, Tag: asn1.TagSequence, IsCompound: true, Bytes: seq})
	if err != nil {
		t.Fatal(err)
	}
	return der
}

// PDF returns a minimal one-page PDF with a correct xref table.
func PDF() []byte {
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] >>",
	}
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.7\n")
	offsets := make([]int, len(objects))
	for i, obj := range objects {
		offsets[i] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for _, off := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return buf.Bytes()
}
