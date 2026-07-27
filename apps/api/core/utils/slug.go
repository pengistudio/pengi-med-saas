package utils

import (
	"regexp"
	"strings"
)

// GenerateSlug converts a name into a URL/DB-safe slug. Does not guarantee
// uniqueness — callers must check against their own table and retry with a
// numeric suffix (see UserHandler.Register / CompanyHandler.CreateAdditionalCompany).
func GenerateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > 50 {
		slug = slug[:50]
	}
	if slug == "" {
		slug = "empresa"
	}
	return slug
}
