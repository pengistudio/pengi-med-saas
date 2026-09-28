package catalog

import (
	"sort"
	"strconv"
	"strings"
)

// ResolveLanguage picks the language of a request: the explicit ?lang= query
// value when supported, else the first supported language of the
// Accept-Language header (by q-value, region ignored: "es-EC" → "es"), else
// DefaultLanguage.
func (c *Catalog) ResolveLanguage(query, acceptLanguage string) string {
	if lang := strings.ToLower(strings.TrimSpace(query)); c.Supports(lang) {
		return lang
	}
	for _, lang := range parseAcceptLanguage(acceptLanguage) {
		if c.Supports(lang) {
			return lang
		}
	}
	return DefaultLanguage
}

// parseAcceptLanguage returns the primary subtags of an Accept-Language
// header ordered by q-value (stable for ties), dropping q=0 and "*".
func parseAcceptLanguage(header string) []string {
	type tag struct {
		lang string
		q    float64
	}
	var tags []tag
	for _, part := range strings.Split(header, ",") {
		fields := strings.Split(part, ";")
		lang := strings.ToLower(strings.TrimSpace(fields[0]))
		if i := strings.IndexByte(lang, '-'); i >= 0 {
			lang = lang[:i]
		}
		if lang == "" || lang == "*" {
			continue
		}
		q := 1.0
		for _, param := range fields[1:] {
			name, value, ok := strings.Cut(strings.TrimSpace(param), "=")
			if !ok || strings.TrimSpace(name) != "q" {
				continue
			}
			parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
			if err != nil {
				parsed = 0
			}
			q = parsed
		}
		if q <= 0 {
			continue
		}
		tags = append(tags, tag{lang, q})
	}
	sort.SliceStable(tags, func(i, j int) bool { return tags[i].q > tags[j].q })
	langs := make([]string, len(tags))
	for i, t := range tags {
		langs[i] = t.lang
	}
	return langs
}
