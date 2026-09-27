package pdfsign_test

import (
	"reflect"
	"sort"
	"testing"
	"time"

	"pengi-med-saas/core/pdfsign"
	"pengi-med-saas/core/pdfsign/pdfsigntest"
)

func loadCert(t *testing.T, c pdfsigntest.Cert) *pdfsign.Certificate {
	t.Helper()
	cert, err := pdfsign.Load(pdfsigntest.P12With(t, c, "ok", time.Now().Add(24*time.Hour)), "ok")
	if err != nil {
		t.Fatal(err)
	}
	return cert
}

func TestTaxIDs(t *testing.T) {
	cases := []struct {
		name string
		cert pdfsigntest.Cert
		want []string
	}{
		{"BCE extensions (cédula + RUC)", pdfsigntest.Cert{Extensions: map[string]string{
			"1.3.6.1.4.1.37947.3.1": "1712345675", "1.3.6.1.4.1.37947.3.11": "1790012345001",
		}}, []string{"1712345675", "1790012345001"}},
		{"Security Data extensions", pdfsigntest.Cert{Extensions: map[string]string{
			"1.3.6.1.4.1.37746.3.11": "0992345678001",
		}}, []string{"0992345678001"}},
		{"ANF AC 18332 arc", pdfsigntest.Cert{Extensions: map[string]string{"1.3.6.1.4.1.18332.3.11": "1790012345001"}}, []string{"1790012345001"}},
		{"Digercic", pdfsigntest.Cert{Extensions: map[string]string{"1.3.6.1.4.1.55519.1.1.1.5.2.4": "1790012345001"}}, []string{"1790012345001"}},
		{"Uanataca subjectAltName otherName", pdfsigntest.Cert{SANOtherNames: map[string]string{
			"1.3.6.1.4.1.47286.102.3.1": "0923456784", "1.3.6.1.4.1.47286.102.3.11": "0990012345001",
		}}, []string{"0923456784", "0990012345001"}},
		{"Consejo de la Judicatura subjectAltName", pdfsigntest.Cert{SANOtherNames: map[string]string{"1.3.6.1.4.1.43745.1.3.11": "1790012345001"}}, []string{"1790012345001"}},
		{"ETSI subject attributes", pdfsigntest.Cert{SubjectSerial: "IDCEC-1712345675", OrganizationIdentifier: "VATEC-1790012345001"}, []string{"1712345675", "1790012345001"}},
		{"plain valid cédula in subject serialNumber", pdfsigntest.Cert{SubjectSerial: "0601234560"}, []string{"0601234560"}},
		{"subject serialNumber that is not an id (Security Data style)", pdfsigntest.Cert{SubjectSerial: "191018151429"}, nil},
		{"subject serialNumber with bad check digit", pdfsigntest.Cert{SubjectSerial: "0912345678"}, nil},
		{"passport", pdfsigntest.Cert{SubjectSerial: "PASEC-A1234567", Extensions: map[string]string{"1.3.6.1.4.1.37947.3.1": "A1234567"}}, nil},
		{"duplicates collapse", pdfsigntest.Cert{SubjectSerial: "1712345675", Extensions: map[string]string{"1.3.6.1.4.1.37947.3.1": "1712345675"}}, []string{"1712345675"}},
		{"unknown format", pdfsigntest.Cert{CommonName: "X"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := loadCert(t, tc.cert).TaxIDs()
			sort.Strings(got)
			if len(got) == 0 && len(tc.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("TaxIDs() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestMatchesTaxID(t *testing.T) {
	cases := []struct {
		name  string
		ids   []string
		ruc   string
		match bool
	}{
		{"company RUC", []string{"1712345675", "1790012345001"}, "1790012345001", true},
		{"company RUC with formatting", []string{"1790012345001"}, " 1790012345-001 ", true},
		{"persona natural: cédula + 001", []string{"1712345675"}, "1712345675001", true},
		{"persona natural: other establishment suffix", []string{"1712345675"}, "1712345675002", true},
		{"tenant tax id is the cédula", []string{"1712345675001"}, "1712345675", true},
		{"doctor's personal cert on a company", []string{"1712345675"}, "1790012345001", false},
		{"another company", []string{"0990012345001"}, "1790012345001", false},
		{"cédula never matches a sociedad RUC prefix", []string{"1790012345"}, "1790012345001", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := pdfsign.MatchesTaxID(tc.ids, tc.ruc); got != tc.match {
				t.Fatalf("MatchesTaxID(%v, %q) = %v, want %v", tc.ids, tc.ruc, got, tc.match)
			}
		})
	}
}
