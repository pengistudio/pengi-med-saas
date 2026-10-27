package ecid

import "testing"

func TestValidCedula(t *testing.T) {
	valid := []string{"0102030400", "1710034065", "0926687856", "3050000003", "1700000001"}
	for _, id := range valid {
		if !ValidCedula(id) {
			t.Errorf("%s should be valid", id)
		}
	}
	invalid := []string{
		"",
		"010203040",   // 9 digits
		"01020304056", // 11 digits
		"0102030405",  // wrong check digit
		"2510034065",  // province 25
		"0010034065",  // province 00
		"1760034065",  // third digit 6
		"17100340a5",  // not a digit
		" 1710034065", // untrimmed
	}
	for _, id := range invalid {
		if ValidCedula(id) {
			t.Errorf("%q should be invalid", id)
		}
	}
}

func TestNormalizePassport(t *testing.T) {
	for in, want := range map[string]string{"ab12345": "AB12345", "  X1234 ": "X1234", "A1B2C3D4E5F6G7H8I9J0": "A1B2C3D4E5F6G7H8I9J0"} {
		if got, ok := NormalizePassport(in); !ok || got != want {
			t.Errorf("%q: got %q %v, want %q", in, got, ok, want)
		}
	}
	for _, in := range []string{"", "A123", "A1B2C3D4E5F6G7H8I9J0K", "AB-12345", "ÁB12345", "AB 12345"} {
		if _, ok := NormalizePassport(in); ok {
			t.Errorf("%q should be invalid", in)
		}
	}
}
