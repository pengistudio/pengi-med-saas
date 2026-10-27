// Package ecid validates Ecuadorian identity documents: the cédula (with its
// check digit) and passport numbers.
package ecid

import "strings"

// ValidCedula checks an Ecuadorian cédula: 10 digits, province 01-24 or 30,
// third digit below 6, and the módulo-10 check digit.
func ValidCedula(id string) bool {
	if len(id) != 10 {
		return false
	}
	for i := 0; i < len(id); i++ {
		if id[i] < '0' || id[i] > '9' {
			return false
		}
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

// NormalizePassport trims and uppercases a passport number; ok is false
// unless it is 5 to 20 ASCII letters or digits.
func NormalizePassport(s string) (string, bool) {
	s = strings.ToUpper(strings.TrimSpace(s))
	if len(s) < 5 || len(s) > 20 {
		return "", false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'A' || c > 'Z') {
			return "", false
		}
	}
	return s, true
}
