package pdfsign

import (
	"crypto/x509"
	"encoding/asn1"
	"strings"
	"unicode/utf8"
)

// Where Ecuadorian CAs put the holder's cédula and RUC. Source: MINTEL's
// FirmaEC library (minka.gob.ec/mintel/ge/firmaec/firmadigital-libreria,
// package ec.gob.firmadigital.libreria.certificate.ec), the reference parser
// for every CA accredited in Ecuador. Most CAs follow the BCE layout
// "<arc>.3.1" = cédula/pasaporte and "<arc>.3.11" = RUC, either as a private
// X.509 extension or as a subjectAltName otherName (Uanataca, Eclipsoft,
// Consejo de la Judicatura); both places are checked for every OID.
var taxIDOIDs = map[string]bool{}

func init() {
	arcs := []string{
		"1.3.6.1.4.1.37947",     // Banco Central del Ecuador
		"1.3.6.1.4.1.37746",     // Security Data
		"1.3.6.1.4.1.18332",     // ANF AC
		"1.3.6.1.4.1.37442",     // ANF AC (second arc)
		"1.3.6.1.4.1.43745.1",   // Consejo de la Judicatura
		"1.3.6.1.4.1.47286.102", // Uanataca
		"1.3.6.1.4.1.57153.102", // Eclipsoft
		"1.3.6.1.4.1.52643",     // Datil
		"1.3.6.1.4.1.59382",     // Lazzate
		"1.3.6.1.4.1.59198",     // Argos Data
		"1.3.6.1.4.1.34380",     // CorpNewBest
		"1.3.6.1.4.1.56105",     // Alpha Technologies
		"1.3.6.1.4.1.61305",     // FirmaSegura
	}
	for _, arc := range arcs {
		taxIDOIDs[arc+".3.1"] = true  // cédula / pasaporte
		taxIDOIDs[arc+".3.11"] = true // RUC
	}
	// Dirección General de Registro Civil (Digercic) uses its own layout.
	taxIDOIDs["1.3.6.1.4.1.55519.1.1.1.5.2.1"] = true // cédula
	taxIDOIDs["1.3.6.1.4.1.55519.1.1.1.5.2.4"] = true // RUC
}

var (
	oidSubjectAltName         = asn1.ObjectIdentifier{2, 5, 29, 17}
	oidOrganizationIdentifier = asn1.ObjectIdentifier{2, 5, 4, 97}
)

// TaxIDs returns every cédula (10 digits) and RUC (13 digits) found in the
// certificate, digits only and deduplicated. It reads the CA-specific
// extensions and subjectAltName entries listed above, the subject
// organizationIdentifier (ETSI "VATEC-<RUC>") and the subject serialNumber
// (ETSI "IDCEC-<cédula>", or a bare cédula/RUC). A bare serialNumber only
// counts when it passes the cédula check digit (or is a RUC ending in 001):
// some CAs (e.g. Security Data) put a non-identity number there. Passports
// and unrecognised values are skipped, so an empty result means "unknown".
func (c *Certificate) TaxIDs() []string {
	var ids []string
	seen := map[string]bool{}
	add := func(id string) {
		if id != "" && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}

	for _, ext := range c.Cert.Extensions {
		if taxIDOIDs[ext.Id.String()] {
			add(normalizeTaxID(asn1Text(ext.Value)))
		}
	}
	for _, value := range sanOtherNames(c.Cert) {
		add(normalizeTaxID(value))
	}
	for _, name := range c.Cert.Subject.Names {
		if name.Type.Equal(oidOrganizationIdentifier) {
			if s, ok := name.Value.(string); ok {
				add(normalizeTaxID(s))
			}
		}
	}
	if serial := strings.TrimSpace(c.Cert.Subject.SerialNumber); serial != "" {
		if strings.HasPrefix(strings.ToUpper(serial), "IDCEC") || strings.HasPrefix(strings.ToUpper(serial), "VATEC") {
			add(normalizeTaxID(serial))
		} else if id := normalizeTaxID(serial); validCedula(id) || (len(id) == 13 && strings.HasSuffix(id, "001")) {
			add(id)
		}
	}
	return ids
}

// MatchesTaxID reports whether any of the certificate ids identifies the
// holder of taxID (the tenant's RUC). A RUC must match exactly; a cédula
// matches the RUC of a persona natural, which is that cédula plus a 3-digit
// establishment suffix (normally "001") — only when the RUC's third digit is
// below 6, since sociedades (9) and public entities (6) have no cédula.
func MatchesTaxID(ids []string, taxID string) bool {
	ruc := digits(taxID)
	for _, id := range ids {
		switch {
		case id == ruc:
			return true
		case len(id) == 10 && len(ruc) == 13 && ruc[:10] == id && ruc[2] < '6':
			return true
		case len(id) == 13 && len(ruc) == 10 && id[:10] == ruc && id[2] < '6':
			return true
		}
	}
	return false
}

// normalizeTaxID strips an ETSI semantics prefix ("IDCEC-", "VATEC-") and
// separators, and returns the id only when it is a 10 or 13 digit number.
func normalizeTaxID(s string) string {
	s = strings.TrimSpace(s)
	upper := strings.ToUpper(s)
	for _, prefix := range []string{"IDCEC", "VATEC"} {
		if strings.HasPrefix(upper, prefix) {
			s = strings.TrimLeft(s[len(prefix):], "-: ")
			break
		}
	}
	s = strings.NewReplacer(" ", "", "-", "", ".", "").Replace(s)
	if (len(s) != 10 && len(s) != 13) || digits(s) != s {
		return ""
	}
	return s
}

func digits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// validCedula checks an Ecuadorian cédula: province 01-24 or 30, third digit
// below 6, and the módulo-10 check digit.
func validCedula(id string) bool {
	if len(id) != 10 || digits(id) != id {
		return false
	}
	province := int(id[0]-'0')*10 + int(id[1]-'0')
	if (province < 1 || province > 24) && province != 30 {
		return false
	}
	if id[2] >= '6' {
		return false
	}
	sum := 0
	for i := 0; i < 9; i++ {
		v := int(id[i] - '0')
		if i%2 == 0 {
			v *= 2
			if v > 9 {
				v -= 9
			}
		}
		sum += v
	}
	return (10-sum%10)%10 == int(id[9]-'0')
}

// asn1Text decodes a DER-encoded ASN.1 string (UTF8String, PrintableString,
// IA5String, ...); CAs store the extension values that way. Anything else
// yields "".
func asn1Text(der []byte) string {
	var raw asn1.RawValue
	if _, err := asn1.Unmarshal(der, &raw); err != nil || raw.Class != asn1.ClassUniversal || raw.IsCompound {
		return ""
	}
	switch raw.Tag {
	case asn1.TagUTF8String, asn1.TagPrintableString, asn1.TagIA5String, asn1.TagT61String, asn1.TagNumericString, asn1.TagOctetString:
		if utf8.Valid(raw.Bytes) {
			return string(raw.Bytes)
		}
	}
	return ""
}

// sanOtherNames returns the values of the subjectAltName otherName entries
// whose type is one of taxIDOIDs. GeneralName otherName is
// [0] IMPLICIT SEQUENCE { type-id OID, value [0] EXPLICIT ANY }.
func sanOtherNames(cert *x509.Certificate) []string {
	var out []string
	for _, ext := range cert.Extensions {
		if !ext.Id.Equal(oidSubjectAltName) {
			continue
		}
		var names asn1.RawValue
		if _, err := asn1.Unmarshal(ext.Value, &names); err != nil {
			continue
		}
		rest := names.Bytes
		for len(rest) > 0 {
			var name asn1.RawValue
			var err error
			if rest, err = asn1.Unmarshal(rest, &name); err != nil {
				break
			}
			if name.Class != asn1.ClassContextSpecific || name.Tag != 0 {
				continue
			}
			var oid asn1.ObjectIdentifier
			valueDER, err := asn1.Unmarshal(name.Bytes, &oid)
			if err != nil || !taxIDOIDs[oid.String()] {
				continue
			}
			var value asn1.RawValue
			if _, err := asn1.Unmarshal(valueDER, &value); err != nil {
				continue
			}
			// value is the [0] EXPLICIT wrapper; its content is the string.
			out = append(out, asn1Text(value.Bytes))
		}
	}
	return out
}
