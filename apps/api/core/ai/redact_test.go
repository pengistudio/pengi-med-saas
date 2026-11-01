package ai

import (
	"strings"
	"testing"
)

func TestRedact_RemovesIdentifiers(t *testing.T) {
	r := NewRedactor("María José", "Pérez Andrade")
	cases := []struct{ in, mustNot, want string }{
		{"Paciente maria perez acude por cefalea", "perez", PlaceholderName},
		{"MARÍA JOSÉ refiere dolor", "MARÍA", PlaceholderName},
		{"Contacto: mjperez@mail.com", "mjperez", PlaceholderEmail},
		{"Llamar al 0991234567", "0991234567", PlaceholderID},
		{"Teléfono +593 99 123 4567", "123 4567", PlaceholderPhone},
		{"Fijo 02-234-5678", "234-5678", PlaceholderPhone},
		{"Cédula 1712345678 y RUC 1712345678001", "1712345678", PlaceholderID},
		{"Ver https://example.com/x", "example.com", PlaceholderURL},
		{"Referida por Dra. Ana Torres", "Ana Torres", PlaceholderProfessional},
		{"Domicilio en Av. Amazonas N34-56, Quito", "Amazonas", PlaceholderAddress},
	}
	for _, tc := range cases {
		got := r.Redact(tc.in)
		if strings.Contains(got, tc.mustNot) || !strings.Contains(got, tc.want) {
			t.Errorf("Redact(%q) = %q", tc.in, got)
		}
	}
}

func TestRedact_KeepsClinicalContent(t *testing.T) {
	r := NewRedactor("Juan Pérez")
	for _, in := range []string{
		"PA 120/80, FC 72, T 38.5, peso 70.5 kg",
		"Metformina 850 mg cada 12 horas por 30 días",
		"Control el 2026-05-01. HbA1c 7.2%",
		"Enfermedad renal crónica estadio 3",
		"Juanete en pie derecho", // contains "Juan" inside a word
	} {
		if got := r.Redact(in); got != in {
			t.Errorf("Redact(%q) changed clinical text to %q", in, got)
		}
	}
}

func TestRedact_AdjacentNamesAndNoNames(t *testing.T) {
	if got := NewRedactor("Juan Pérez").Redact("Juan Pérez llega"); got != "[paciente] [paciente] llega" {
		t.Fatalf("got %q", got)
	}
	if got := NewRedactor().Redact("sin datos 0991234567"); strings.Contains(got, "0991234567") {
		t.Fatalf("got %q", got)
	}
	var nilR *Redactor
	if got := nilR.Redact("correo a@b.co"); strings.Contains(got, "a@b.co") {
		t.Fatalf("nil redactor kept email: %q", got)
	}
}
