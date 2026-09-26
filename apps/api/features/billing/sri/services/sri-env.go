package services

import (
	"os"
	"strings"
)

// ResolveSriEnv reads SRI_ENV and normalizes it to the SRI ambiente code used in the
// XML and access key: "1" pruebas, "2" producción. Accepts either the numeric code or
// the human-readable "test"/"prod" (case-insensitive); anything else defaults to pruebas.
func ResolveSriEnv() string {
	sriEnv := os.Getenv("SRI_ENV")
	if sriEnv == "2" || strings.EqualFold(sriEnv, "prod") {
		return "2"
	}
	return "1"
}
