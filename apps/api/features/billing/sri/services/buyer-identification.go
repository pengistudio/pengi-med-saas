package services

import "strings"

// ResolveBuyerIdentificationType returns the SRI buyer identification type code:
//
//	04 RUC (13 digits, ends in 001)
//	05 Cédula (10 digits)
//	06 Pasaporte / otro (anything else non-empty)
//	07 Consumidor Final (no document)
func ResolveBuyerIdentificationType(document string) string {
	if document == "" {
		return "07"
	}

	digitsOnly := true
	for _, r := range document {
		if r < '0' || r > '9' {
			digitsOnly = false
			break
		}
	}

	switch {
	case digitsOnly && len(document) == 13 && strings.HasSuffix(document, "001"):
		return "04"
	case digitsOnly && len(document) == 10:
		return "05"
	default:
		return "06"
	}
}
