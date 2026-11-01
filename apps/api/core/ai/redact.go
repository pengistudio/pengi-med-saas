package ai

import (
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

// Placeholders written in place of what Redact removes. They are Spanish
// because the prompts and answers are.
const (
	PlaceholderName         = "[paciente]"
	PlaceholderProfessional = "[profesional]"
	PlaceholderEmail        = "[correo]"
	PlaceholderPhone        = "[teléfono]"
	PlaceholderID           = "[identificación]"
	PlaceholderURL          = "[enlace]"
	PlaceholderAddress      = "[dirección]"
)

var (
	emailRe = regexp.MustCompile(`(?i)[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)
	urlRe   = regexp.MustCompile(`(?i)\b(?:https?://|www\.)\S+`)
	// Phones written with an international prefix (+593 99 123 4567) or an
	// Ecuadorian area/mobile prefix with separators (02-234-5678, 099 123 4567).
	phoneRe = regexp.MustCompile(`\+\d{1,3}[\s\-.]?\(?\d{1,3}\)?(?:[\s\-.]?\d){6,10}\b|\b0\d{1,2}[\s\-.]\d{3}[\s\-.]?\d{3,4}\b`)
	// Any run of 7+ digits: cédula (10), RUC (13), passport-like numbers,
	// phones written together, medical record or insurance numbers. Clinical
	// values (120/80, 38.5, 500 mg, 2026-05-01) never reach 7 digits in a row.
	longNumberRe = regexp.MustCompile(`\b\d{7,}\b`)
	// "Dr. Pérez", "Dra. María López", "doctor Andrade": names of
	// professionals written in free text.
	// Only the title is case insensitive: the name must be capitalized.
	professionalRe = regexp.MustCompile(`\b(?i:dr|dra|doctor|doctora|lcda|lcdo|lic)\.?\s+(?:\p{Lu}[\p{L}'\-]+)(?:\s+\p{Lu}[\p{L}'\-]+){0,2}`)
	// Street addresses: "calle ...", "av. ...", "domicilio en ..." up to the
	// end of the clause.
	addressRe = regexp.MustCompile(`(?i)\b(?:domicilio|dirección|direccion|vive en|reside en|calle|av\.|avenida|barrio|urbanización|urbanizacion|ciudadela|cdla\.)\s(?:av\.|calle|[^.,;\n]){2,60}`)
)

// Redactor removes direct identifiers from free text before it is sent to a
// model: the patient's own names (accent and case insensitive), names of
// professionals after a title, emails, phones, ID numbers (cédula, RUC, any
// 7+ digit run), URLs and street addresses. What stays is clinical content;
// the caller sends age and sex as separate fields, never the birth date.
//
// It is a safety net, not a guarantee: free text can still describe a person
// (a workplace, a relative's name). Callers never send identifier fields
// (name, document, phone, email, address) in the first place.
type Redactor struct {
	nameRe *regexp.Regexp
}

// NewRedactor returns a Redactor that also removes the given names (each
// word of 3+ letters becomes a token, e.g. "María José Pérez Andrade").
func NewRedactor(names ...string) *Redactor {
	seen := map[string]bool{}
	var tokens []string
	for _, name := range names {
		for _, word := range strings.Fields(name) {
			word = strings.Trim(word, ".,;:()\"'")
			if utf8.RuneCountInString(word) < 3 {
				continue
			}
			key := strings.ToLower(foldAccents(word))
			if seen[key] {
				continue
			}
			seen[key] = true
			tokens = append(tokens, accentInsensitive(key))
		}
	}
	r := &Redactor{}
	if len(tokens) > 0 {
		// Longest first so "Andrade" wins over a shorter prefix token.
		sort.Slice(tokens, func(i, j int) bool { return len(tokens[i]) > len(tokens[j]) })
		r.nameRe = regexp.MustCompile(`(?i)(^|[^\p{L}])(` + strings.Join(tokens, "|") + `)($|[^\p{L}])`)
	}
	return r
}

// Redact returns text without the identifiers described on Redactor.
func (r *Redactor) Redact(text string) string {
	if strings.TrimSpace(text) == "" {
		return text
	}
	text = emailRe.ReplaceAllString(text, PlaceholderEmail)
	text = urlRe.ReplaceAllString(text, PlaceholderURL)
	text = phoneRe.ReplaceAllString(text, PlaceholderPhone)
	text = longNumberRe.ReplaceAllString(text, PlaceholderID)
	text = addressRe.ReplaceAllString(text, PlaceholderAddress)
	text = professionalRe.ReplaceAllString(text, PlaceholderProfessional)
	if r != nil && r.nameRe != nil {
		// Matches share their boundary characters, so adjacent names
		// ("Juan Pérez") need a second pass.
		for i := 0; i < 2; i++ {
			text = r.nameRe.ReplaceAllString(text, "${1}"+PlaceholderName+"${3}")
		}
	}
	return text
}

var accentFold = strings.NewReplacer(
	"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n",
	"Á", "a", "É", "e", "Í", "i", "Ó", "o", "Ú", "u", "Ü", "u", "Ñ", "n",
)

func foldAccents(s string) string { return accentFold.Replace(s) }

// accentInsensitive turns a folded lowercase word into a regexp matching it
// with or without Spanish accents.
func accentInsensitive(folded string) string {
	var b strings.Builder
	for _, r := range folded {
		switch r {
		case 'a':
			b.WriteString("[aá]")
		case 'e':
			b.WriteString("[eé]")
		case 'i':
			b.WriteString("[ií]")
		case 'o':
			b.WriteString("[oó]")
		case 'u':
			b.WriteString("[uúü]")
		case 'n':
			b.WriteString("[nñ]")
		default:
			b.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	return b.String()
}
