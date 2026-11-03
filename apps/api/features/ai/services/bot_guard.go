package ai_services

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"

	whatsapp_models "pengi-med-saas/features/whatsapp/models"
)

// Checks of the WhatsApp assistant that never trust the model: the triggers
// that hand a message to reception before any model call, and the guards
// every model reply must pass before it is sent. All work on text folded to
// lower case without accents.

// MaxBotMessageRunes: a longer patient message goes to reception unread by
// the model.
const MaxBotMessageRunes = 1000

// MaxBotReplyRunes caps the model's reply (the prompt asks for 3 sentences).
const MaxBotReplyRunes = 600

// botPatterns compiles whole-word alternatives over folded text.
func botPatterns(alternatives ...string) *regexp.Regexp {
	return regexp.MustCompile(`(^|[^\p{L}\d])(?:` + strings.Join(alternatives, "|") + `)($|[^\p{L}\d])`)
}

// Emergencies: the patient is told to call 911 / go to an emergency room,
// then the chat goes to reception. Matched as substrings of the text reduced
// to letters (see triggerForms), so spacing, accents, leetspeak ("d0lor") and
// stretched letters ("auxiliooo") don't hide them. Errs toward false
// positives: a false alarm is a handoff, a miss is silence.
var emergencyStems = []string{
	"emergenc", "emergency",
	"no respira", "no puedo respirar", "no puede respirar", "falta el aire", "falta de aire", "falta aire",
	"dificultad para respirar", "ahog", "asfixi", "cant breathe", "cannot breathe", "not breathing",
	"desmay", "inconscien", "unconscious", "perdio el conocimiento", "no reacciona",
	"convuls", "ataque epileptico", "seizure",
	"sangr", "hemorrag", "bleeding", "blood",
	"infart", "ataque cardiaco", "ataque al corazon", "paro cardi", "paro respir", "heart attack",
	"derrame", "cara caida", "cara torcida", "paraliz", "no puedo mover", "stroke",
	"suicid", "quitarme la vida", "matarme", "morir", "muriendo", "muerte", "muerto", "no quiero vivir",
	"hacerme dano", "dying", "kill myself",
	"sobredosis", "overdose", "envenen", "intoxic", "poison",
	"accidente", "atropell", "fractura expuesta", "quemadura grave", "mordedura",
	"morad", "auxilio", "socorro", "help me",
	"trabajo de parto", "rompi la fuente", "rompio la fuente", "rompi fuente",
}

// Chest or heart with pain or pressure is an emergency, however phrased.
var (
	emergencyChestStems = []string{"pecho", "corazon", "toraci", "chest", "heart"}
	emergencyPainStems  = []string{"dolor", "duele", "presion", "opresion", "apret", "pain", "hurt"}
)

var emergencyForms, chestForms, painForms = compactStems(emergencyStems), compactStems(emergencyChestStems), compactStems(emergencyPainStems)

func compactStems(stems []string) []string {
	out := make([]string, 0, len(stems))
	for _, st := range stems {
		out = append(out, compactLetters(foldBot(st)))
	}
	return out
}

// BotIsEmergency reports whether text has emergency words (any of
// triggerForms).
//
// Blood tests are the one exemption: when the only emergency stem is the
// blood one ("sangr"/"blood"), the text clearly asks about an exam or a
// donation ("examen de sangre", "donar sangre", "biometría") and has no
// health-question word, it goes to the model path. Any doubt stays an
// emergency.
func BotIsEmergency(text string) bool {
	bloodOnly := false
	for _, t := range triggerForms(text) {
		if containsAny(t, chestForms) && containsAny(t, painForms) {
			return true
		}
		for i, st := range emergencyForms {
			if !strings.Contains(t, st) {
				continue
			}
			if !bloodStem[emergencyStems[i]] {
				return true
			}
			bloodOnly = true
		}
	}
	if !bloodOnly {
		return false
	}
	f := foldConfusables(foldBot(text))
	return !(bloodExamRe.MatchString(f) && !botSymptomRe.MatchString(f) && !botSymptomRe.MatchString(leetI.Replace(f)))
}

// bloodStem are the emergency stems a blood-test question may carry.
var bloodStem = map[string]bool{"sangr": true, "blood": true}

// bloodExamRe: the text asks about a blood test or donation.
var bloodExamRe = regexp.MustCompile(`\b(?:examen(?:es)?|analisis|pruebas?|muestras?|tomas?|perfil)\s+de\s+sangre\b|\bdonar\s+sangre\b|\bdonacion(?:es)?\s+de\s+sangre\b|\bbiometria\b|\bhemograma\b|\bblood\s+tests?\b`)

// confusables maps Cyrillic and Greek letters that look like Latin ones
// (lower case, after foldBot) to those Latin letters, so "dоlor" with a
// Cyrillic о still reads "dolor".
var confusables = strings.NewReplacer(
	"а", "a", "в", "b", "е", "e", "ё", "e", "к", "k", "м", "m", "н", "h", "о", "o", "р", "p", "с", "c",
	"т", "t", "у", "y", "х", "x", "і", "i", "ї", "i", "ј", "j", "ѕ", "s", "ԁ", "d", "ԛ", "q", "ԝ", "w", "ъ", "b", "ь", "b",
	"α", "a", "β", "b", "γ", "y", "ε", "e", "ζ", "z", "η", "n", "ι", "i", "κ", "k", "μ", "u", "ν", "v", "ο", "o",
	"ρ", "p", "σ", "o", "ς", "s", "τ", "t", "υ", "u", "χ", "x", "ω", "w",
)

func foldConfusables(s string) string { return confusables.Replace(s) }

// MixedScript reports whether a word of text mixes Latin letters with
// Cyrillic or Greek ones: a look-alike spelling meant to slip past the
// checks. Such a text is handed to reception.
func MixedScript(text string) bool {
	for _, w := range strings.FieldsFunc(norm.NFKD.String(text), func(r rune) bool { return !unicode.IsLetter(r) }) {
		latin, other := false, false
		for _, r := range w {
			switch {
			case unicode.Is(unicode.Latin, r):
				latin = true
			case unicode.Is(unicode.Cyrillic, r) || unicode.Is(unicode.Greek, r):
				other = true
			}
		}
		if latin && other {
			return true
		}
	}
	return false
}

func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// leet maps digits used as letters; "1" is tried as both i and l.
var (
	leetI = strings.NewReplacer("0", "o", "1", "i", "3", "e", "4", "a", "5", "s", "7", "t", "@", "a", "$", "s")
	leetL = strings.NewReplacer("0", "o", "1", "l", "3", "e", "4", "a", "5", "s", "7", "t", "@", "a", "$", "s")
)

// triggerForms are text folded (foldBot), with leetspeak digits as letters,
// reduced to letters only and with repeated letters collapsed: "D 0 L O R
// en pechooo" → "dolorenpecho".
func triggerForms(text string) []string {
	f := foldConfusables(foldBot(text))
	return []string{compactLetters(leetI.Replace(f)), compactLetters(leetL.Replace(f))}
}

// compactLetters keeps only letters and collapses runs of the same letter.
func compactLetters(s string) string {
	var b strings.Builder
	var prev rune
	for _, r := range s {
		if !unicode.IsLetter(r) || r == prev {
			continue
		}
		b.WriteRune(r)
		prev = r
	}
	return b.String()
}

// Asking for a person (or a complaint): straight to reception.
var botPersonRe = botPatterns(
	`personas?`, `humanos?`, `humanas?`, `ser humano`, `agentes?`, `recepcion`, `recepcionistas?`, `asesor\p{L}*`,
	`operador\p{L}*`, `secretari[oa]s?`, `alguien real`, `hablar con alguien`, `atencion al cliente`, `encargad[oa]`,
	`hablar con (?:el|la|un|una) (?:doctor|doctora|medico|medica|dr|dra|licenciad[oa])`, `quejas?`, `reclamos?`,
)

// Health questions (symptoms, medication, results, treatment): the
// assistant never gives medical advice, so they go to reception.
var botSymptomRe = botPatterns(
	`dolor\p{L}*`, `duele\p{L}*`, `fiebre`, `tos`, `gripe`, `resfri\p{L}*`, `vomit\p{L}*`, `diarrea`, `nausea\p{L}*`, `mareo\p{L}*`, `maread[oa]`,
	`sangrado`, `sangrando`, `sangra`, `heridas?`, `hinchaz\p{L}*`, `hinchad[oa]s?`, `inflamad[oa]s?`, `inflamacion`, `alergi\p{L}*`,
	`sarpullido`, `ronchas?`, `picazon`, `comezon`, `infeccion\p{L}*`, `infectad[oa]s?`, `sintomas?`, `presion (?:alta|baja|arterial)`,
	`glucosa`, `azucar`, `diabet\p{L}*`, `embaraz\p{L}*`, `medicamentos?`, `medicinas?`, `pastillas?`, `dosis`, `recetas?`, `recetar\p{L}*`,
	`antibiotic\p{L}*`, `ibuprofeno`, `paracetamol`, `diagnostic\p{L}*`, `que tengo`, `es grave`, `es normal`, `resultados?`,
	`tratamientos?`, `me siento`, `enferm[oa]s?`, `enfermedad\p{L}*`, `cancer`, `tumor\p{L}*`, `bulto`, `lesion\p{L}*`, `golpe\p{L}*`,
	`me cai`, `quemadura\p{L}*`, `fractura\p{L}*`, `urgente\p{L}*`, `urgencias?`, `contagi\p{L}*`, `vacuna\p{L}*`,
)

// Booking intent: the server appends the portal link to the answer.
var botBookingRe = botPatterns(`citas?`, `agendar\p{L}*`, `agendo`, `agenda`, `turnos?`, `reservar\p{L}*`, `reservas?`, `separar`)

// BotPrecheck returns the handoff reason of a patient text that must not
// reach the model ("" when it may): emergency words first (they get their
// own message), then asking for a person, health questions and texts too
// long to answer.
func BotPrecheck(text string) string {
	t := foldConfusables(foldBot(text))
	tl := leetI.Replace(t)
	switch {
	case BotIsEmergency(text):
		return whatsapp_models.BotReasonEmergency
	case MixedScript(text):
		return whatsapp_models.BotReasonSuspicious
	case botPersonRe.MatchString(t) || botPersonRe.MatchString(tl):
		return whatsapp_models.BotReasonAskedPerson
	case botSymptomRe.MatchString(t) || botSymptomRe.MatchString(tl):
		return whatsapp_models.BotReasonSymptoms
	case utf8.RuneCountInString(strings.TrimSpace(text)) > MaxBotMessageRunes:
		return whatsapp_models.BotReasonTooLong
	}
	return ""
}

// WantsBooking reports whether the patient asks about appointments.
func WantsBooking(text string) bool { return botBookingRe.MatchString(foldBot(text)) }

// Patterns of the guards (over folded text).
var (
	// Any host-like token: name.tld with an alphabetic TLD of 2+ letters
	// (linktr.ee, pagos.xyz, algo.ai), with or without scheme or path.
	guardURLRe = regexp.MustCompile(`(?:https?://|www\.)[^\s]+|[a-z0-9][a-z0-9\-]*(?:\.[a-z0-9\-]+)*\.[a-z]{2,}\b(?:/[^\s]*)?`)
	// URLs spelled out ("clinica punto com", "arroba").
	guardSpelledURLRe = regexp.MustCompile(`\bpunto\s*(?:com|co|ec|net|org|ai|io|me|xyz|dev|app|info|top|ee|ly)\b|\barroba\b|\bdoble\s*v\b|\bwww\b|\bhttp`)
	guardEmailRe      = regexp.MustCompile(`[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}`)
	// Digit runs that may be a phone: 7+ digits with separators.
	guardPhoneRe  = regexp.MustCompile(`\+?\d[\d\s().\-]{5,}\d`)
	guardNumberRe = regexp.MustCompile(`\d+(?:[.,]\d+)?`)
	// Amounts: a currency mark next to a number, or a price word before it.
	guardAmountRe    = regexp.MustCompile(`(?:\$|usd|us\$)\s*(\d+(?:[.,]\d{1,2})?)|(\d+(?:[.,]\d{1,2})?)\s*(?:\$|usd|dolar\p{L}*)`)
	guardPriceWordRe = regexp.MustCompile(`\b(?:cuesta|cuestan|precio\p{L}*|valor\p{L}*|costo\p{L}*|vale|valen|tarifa\p{L}*|cobr\p{L}*|pagar|pago)\b[^\d\n]{0,20}(\d+(?:[.,]\d{1,2})?)`)
	// Amounts and phone digits written in words.
	guardSpelledAmountRe = regexp.MustCompile(`\b(?:un|uno|una|dos|tres|cuatro|cinco|seis|siete|ocho|nueve|diez|once|doce|trece|catorce|quince|veinte|veinti\p{L}+|treinta|cuarenta|cincuenta|sesenta|setenta|ochenta|noventa|cien|ciento|doscientos|trescientos|cuatrocientos|quinientos|mil)(?:\s+y\s+\p{L}+)?\s+(?:dolar\p{L}*|usd)\b`)
	guardSpelledPhoneRe  = regexp.MustCompile(`(?:\b(?:cero|uno|dos|tres|cuatro|cinco|seis|siete|ocho|nueve)\b[\s,.\-]*){5,}`)
	guardAddressRe       = regexp.MustCompile(`\b(?:calle|av|avda|avenida|barrio|urbanizacion|ciudadela|cdla|edificio|torre|sector|esquina|km|kilometro|interseccion|diagonal|pasaje|parroquia)\b\.?\s+([^.,;:\n]{2,60})`)
	guardPlaceholderRe   = regexp.MustCompile(`\[(?:paciente|profesional|correo|telefono|identificacion|enlace|direccion)\]`)
	// Price vocabulary: allowed in a reply only when the material has it.
	guardPriceVocabRe = regexp.MustCompile(`\$|\b(?:precio|cuest|costo|valor|dolar|gratis|gratuit|descuent|tarifa|usd|promocion|oferta)\p{L}*`)
)

// addressStopwords are skipped when comparing the words after an address
// keyword with the clinic's data.
var addressStopwords = map[string]bool{
	"de": true, "del": true, "la": true, "el": true, "los": true, "las": true, "y": true, "e": true, "en": true,
	"n": true, "no": true, "nro": true, "num": true, "numero": true, "frente": true, "a": true, "al": true, "entre": true,
}

// GuardBotReply checks the model's reply against the clinic's material and
// returns "" when it may be sent, else the guard that failed (for logs and
// the test panel, never shown to the patient). Anything the patient could
// act on — a URL or host-like token (also spelled out), an email, a phone
// number (also in words), price vocabulary, an amount (also in words), a
// large number, an address — must already be in the material; amounts only
// match the material's currency-marked numbers. A reply longer than
// MaxBotReplyRunes, or carrying a redaction placeholder, fails too. Digits
// are normalized first (fullwidth, Arabic-Indic). On doubt: handoff.
func GuardBotReply(reply string, k BotKnowledge) string {
	reply = strings.TrimSpace(reply)
	if reply == "" {
		return "empty"
	}
	if utf8.RuneCountInString(reply) > MaxBotReplyRunes {
		return "length"
	}
	r := foldBot(reply)
	material := foldBot(k.MaterialText())
	materialCompact := compactSpaces(material)
	if guardPlaceholderRe.MatchString(r) {
		return "placeholder"
	}
	for _, e := range guardEmailRe.FindAllString(r, -1) {
		if !strings.Contains(material, e) {
			return "email"
		}
	}
	for _, u := range guardURLRe.FindAllString(guardEmailRe.ReplaceAllString(r, " "), -1) {
		if !strings.Contains(materialCompact, normalizeURL(u)) {
			return "url"
		}
	}
	for _, u := range guardSpelledURLRe.FindAllString(r, -1) {
		if !strings.Contains(material, u) {
			return "url"
		}
	}
	if guardSpelledPhoneRe.MatchString(r) {
		return "phone"
	}
	materialPhones := phoneDigits(material)
	for _, p := range guardPhoneRe.FindAllString(r, -1) {
		d := onlyDigits(p)
		if len(d) < 7 {
			continue
		}
		if !phoneKnown(d, materialPhones) {
			return "phone"
		}
	}
	for _, w := range guardPriceVocabRe.FindAllString(r, -1) {
		if !strings.Contains(material, priceStem(w)) {
			return "price_word"
		}
	}
	for _, m := range guardSpelledAmountRe.FindAllString(r, -1) {
		if !strings.Contains(material, m) {
			return "amount"
		}
	}
	// Allowed amounts: only the material's currency-marked numbers (never
	// its hours or street numbers).
	amounts := materialAmounts(material)
	for _, m := range guardAmountRe.FindAllStringSubmatch(r, -1) {
		n := m[1]
		if n == "" {
			n = m[2]
		}
		if !amounts[normalizeNumber(n)] {
			return "amount"
		}
	}
	for _, m := range guardPriceWordRe.FindAllStringSubmatch(r, -1) {
		if !amounts[normalizeNumber(m[1])] {
			return "amount"
		}
	}
	// A number of 3+ digits (a price without its mark, a street number, an
	// extension) that the material doesn't have.
	numbers := materialNumbers(material)
	for _, n := range guardNumberRe.FindAllString(r, -1) {
		if len(onlyDigits(n)) >= 3 && !numbers[normalizeNumber(n)] && !phoneKnown(onlyDigits(n), materialPhones) {
			return "number"
		}
	}
	materialWords := wordSet(material)
	for _, m := range guardAddressRe.FindAllStringSubmatch(r, -1) {
		checked := 0
		for _, w := range strings.FieldsFunc(m[1], func(c rune) bool { return !unicode.IsLetter(c) && !unicode.IsDigit(c) }) {
			if addressStopwords[w] {
				continue
			}
			if !materialWords[w] {
				return "address"
			}
			if checked++; checked == 3 {
				break
			}
		}
	}
	return ""
}

// foldBot lower-cases s and removes accents and other marks (ñ → n),
// whatever the Unicode form (NFKD: also fullwidth letters and digits), and
// writes any decimal digit (Arabic-Indic...) as 0-9, so patterns and
// comparisons see one spelling.
func foldBot(s string) string {
	var b strings.Builder
	for _, r := range norm.NFKD.String(s) {
		switch {
		case unicode.Is(unicode.Mn, r):
		case r > unicode.MaxASCII && unicode.IsDigit(r):
			b.WriteRune('0' + digitValue(r))
		default:
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

// digitValue is the value of a decimal digit outside ASCII: decimal digits
// come in contiguous blocks of ten starting at zero.
func digitValue(r rune) rune {
	n := rune(0)
	for n < 9 && unicode.IsDigit(r-n-1) {
		n++
	}
	return n
}

func compactSpaces(s string) string { return strings.Join(strings.Fields(s), " ") }

// normalizeURL drops the scheme, "www." and trailing punctuation.
func normalizeURL(u string) string {
	u = strings.TrimRight(u, ".,;:!?)»\"'")
	u = strings.TrimPrefix(strings.TrimPrefix(u, "https://"), "http://")
	return strings.TrimPrefix(u, "www.")
}

func onlyDigits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// phoneDigits are the digit runs of the material that may be phones.
func phoneDigits(material string) []string {
	var out []string
	for _, p := range guardPhoneRe.FindAllString(material, -1) {
		if d := onlyDigits(p); len(d) >= 7 {
			out = append(out, d)
		}
	}
	for _, n := range guardNumberRe.FindAllString(material, -1) {
		if d := onlyDigits(n); len(d) >= 7 {
			out = append(out, d)
		}
	}
	return out
}

// phoneKnown reports whether the number d is one of known, ignoring the
// country code / trunk 0 ("+593 99 123 4567" is "0991234567"): the last 9
// digits (all when shorter) must match.
func phoneKnown(d string, known []string) bool {
	tail := func(s string) string {
		if len(s) > 9 {
			return s[len(s)-9:]
		}
		return s
	}
	for _, k := range known {
		if tail(k) == tail(d) || strings.Contains(k, d) {
			return true
		}
	}
	return false
}

// materialNumbers are the numbers of the material, normalized.
func materialNumbers(material string) map[string]bool {
	out := map[string]bool{}
	for _, n := range guardNumberRe.FindAllString(material, -1) {
		out[normalizeNumber(n)] = true
	}
	return out
}

// materialAmounts are the material's currency-marked numbers, normalized.
func materialAmounts(material string) map[string]bool {
	out := map[string]bool{}
	for _, m := range guardAmountRe.FindAllStringSubmatch(material, -1) {
		n := m[1]
		if n == "" {
			n = m[2]
		}
		out[normalizeNumber(n)] = true
	}
	return out
}

// priceStem is the part of a price word the material must contain
// ("cuestan" → "cuest", "dólares" → "dolar").
func priceStem(w string) string {
	for _, st := range []string{"precio", "cuest", "costo", "valor", "dolar", "gratis", "gratuit", "descuent", "tarifa", "usd", "promocion", "oferta"} {
		if strings.HasPrefix(w, st) {
			return st
		}
	}
	return w
}

// normalizeNumber reads "30", "30.00", "30,5" alike: decimal comma to dot,
// trailing zero decimals and leading zeros dropped.
func normalizeNumber(n string) string {
	n = strings.ReplaceAll(n, ",", ".")
	if i := strings.IndexByte(n, '.'); i >= 0 {
		n = strings.TrimRight(n, "0")
		n = strings.TrimSuffix(n, ".")
	}
	n = strings.TrimLeft(n, "0")
	if n == "" || strings.HasPrefix(n, ".") {
		n = "0" + n
	}
	return n
}

func wordSet(s string) map[string]bool {
	out := map[string]bool{}
	for _, w := range strings.FieldsFunc(s, func(c rune) bool { return !unicode.IsLetter(c) && !unicode.IsDigit(c) }) {
		out[w] = true
	}
	return out
}
