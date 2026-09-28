package catalog_test

// Tests over the real catalog embedded in the binary (i18n/messages):
//
//   - parity: es and en have the same keys and the same placeholders per key
//     (duplicate keys within a file already make Load fail);
//   - coverage: every AppError code in core/errors/codes.go has a translation;
//   - static keys: every message literal passed to envelope.* is a key of the
//     catalog, except the legacy sentences listed in testdata/legacy_messages.txt.

import (
	"bufio"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"pengi-med-saas/i18n/catalog"
	i18n_messages "pengi-med-saas/i18n/messages"
)

// apiRoot is apps/api, relative to this package's directory.
const apiRoot = "../.."

const legacyBaselinePath = "testdata/legacy_messages.txt"

func loadEmbedded(t *testing.T) *catalog.Catalog {
	t.Helper()
	c, err := catalog.Load(i18n_messages.FS)
	if err != nil {
		t.Fatalf("load embedded catalog: %v", err)
	}
	return c
}

func TestEmbedded_LanguagesAreEsAndEn(t *testing.T) {
	got := loadEmbedded(t).Languages()
	if strings.Join(got, ",") != "en,es" {
		t.Fatalf("Languages() = %v, want [en es]", got)
	}
}

func TestEmbedded_ParityBetweenLanguages(t *testing.T) {
	c := loadEmbedded(t)
	es, _ := c.Bundle("es")
	en, _ := c.Bundle("en")

	for key := range es {
		if _, ok := en[key]; !ok {
			t.Errorf("key %q is in messages_es.json but not in messages_en.json", key)
		}
	}
	for key := range en {
		if _, ok := es[key]; !ok {
			t.Errorf("key %q is in messages_en.json but not in messages_es.json", key)
		}
	}
	for key, esValue := range es {
		enValue, ok := en[key]
		if !ok {
			continue
		}
		if a, b := placeholders(esValue), placeholders(enValue); a != b {
			t.Errorf("key %q: placeholders differ: es %s, en %s", key, a, b)
		}
	}
}

var placeholderRE = regexp.MustCompile(`\{\{\s*[\w.]+\s*\}\}|\{\s*[\w.]+\s*\}`)

func placeholders(s string) string {
	set := map[string]bool{}
	for _, p := range placeholderRE.FindAllString(s, -1) {
		set[strings.ReplaceAll(p, " ", "")] = true
	}
	out := make([]string, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	sort.Strings(out)
	return "[" + strings.Join(out, " ") + "]"
}

func TestEmbedded_EveryErrorCodeIsTranslated(t *testing.T) {
	c := loadEmbedded(t)
	codes := appErrorCodes(t)
	if len(codes) < 50 {
		t.Fatalf("found only %d error codes in core/errors/codes.go; did the parser break?", len(codes))
	}
	for _, code := range codes {
		for _, lang := range c.Languages() {
			if !hasKey(c, lang, code) {
				t.Errorf("error code %s has no translation in messages_%s.json", code, lang)
			}
		}
	}
}

// appErrorCodes returns the first argument of every NewAppError("E-...", ...)
// call in core/errors/codes.go.
func appErrorCodes(t *testing.T) []string {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(apiRoot, "core/errors/codes.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var codes []string
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if ident, ok := call.Fun.(*ast.Ident); !ok || ident.Name != "NewAppError" || len(call.Args) == 0 {
			return true
		}
		if code, ok := stringLiteral(call.Args[0]); ok {
			codes = append(codes, code)
		}
		return true
	})
	return codes
}

func hasKey(c *catalog.Catalog, lang, key string) bool {
	bundle, _ := c.Bundle(lang)
	_, ok := bundle[key]
	return ok
}

// messageArg is the index of the message argument of each envelope
// constructor that takes one.
var messageArg = map[string]int{
	"SuccessResponse":      1,
	"ErrorResponse":        1,
	"New":                  1,
	"PagedSuccessResponse": 4,
}

type messageUse struct {
	file    string // relative to apps/api, slash-separated
	line    int
	literal string
}

// TestEmbedded_EnvelopeMessagesAreCatalogKeys parses every non-test .go file
// under apps/api and checks the message argument of envelope.SuccessResponse,
// ErrorResponse, New and PagedSuccessResponse:
//
//   - a literal without spaces is a key and must exist in every language;
//   - a literal with spaces is a legacy sentence, allowed only while it is
//     listed in testdata/legacy_messages.txt ("file:literal" per line). An
//     entry that is no longer in the code also fails, so the list only shrinks;
//   - any other argument (a variable, a call, a concatenation) cannot be
//     checked statically and is skipped; at the time of writing there are
//     none (the count is logged with -v).
func TestEmbedded_EnvelopeMessagesAreCatalogKeys(t *testing.T) {
	c := loadEmbedded(t)
	uses, skipped := envelopeMessages(t)
	if len(uses) < 100 {
		t.Fatalf("found only %d envelope messages; did the parser break?", len(uses))
	}
	t.Logf("%d literal messages checked, %d non-literal arguments skipped", len(uses), skipped)

	baseline := readBaseline(t)
	seen := map[string]bool{}
	var newSentences []string

	for _, u := range uses {
		if strings.ContainsAny(u.literal, " \t\n") {
			entry := u.file + ":" + u.literal
			if !baseline[entry] {
				if !seen[entry] {
					newSentences = append(newSentences, entry)
				}
				t.Errorf("%s:%d: message %q is a sentence, not an i18n key; add a key to messages_es.json and messages_en.json and use it", u.file, u.line, u.literal)
			}
			seen[entry] = true
			continue
		}
		for _, lang := range c.Languages() {
			if !hasKey(c, lang, u.literal) {
				t.Errorf("%s:%d: i18n key %q is missing from messages_%s.json", u.file, u.line, u.literal, lang)
			}
		}
	}

	var stale []string
	for entry := range baseline {
		if !seen[entry] {
			stale = append(stale, entry)
		}
	}
	sort.Strings(stale)
	for _, entry := range stale {
		t.Errorf("%s: %q is no longer in the code; delete that line", legacyBaselinePath, entry)
	}
	if len(newSentences) > 0 {
		sort.Strings(newSentences)
		t.Logf("new sentences (use an i18n key instead of adding them to the baseline):\n%s", strings.Join(newSentences, "\n"))
	}
}

func envelopeMessages(t *testing.T) (uses []messageUse, skipped int) {
	t.Helper()
	fset := token.NewFileSet()
	err := filepath.WalkDir(apiRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "vendor", "node_modules", "testdata", "tmp", "storage":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return err
		}
		alias := envelopeImportName(file)
		if alias == "" {
			return nil
		}
		rel, _ := filepath.Rel(apiRoot, path)
		rel = filepath.ToSlash(rel)
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != alias {
				return true
			}
			idx, ok := messageArg[sel.Sel.Name]
			if !ok || idx >= len(call.Args) {
				return true
			}
			lit, ok := stringLiteral(call.Args[idx])
			if !ok {
				skipped++
				return true
			}
			if lit == "" && sel.Sel.Name == "New" {
				return true // envelope.New falls back to the HTTP status text
			}
			uses = append(uses, messageUse{file: rel, line: fset.Position(call.Pos()).Line, literal: lit})
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return uses, skipped
}

// envelopeImportName returns the name under which file imports core/envelope,
// or "" when it does not import it.
func envelopeImportName(file *ast.File) string {
	for _, imp := range file.Imports {
		path, _ := strconv.Unquote(imp.Path.Value)
		if path != "pengi-med-saas/core/envelope" {
			continue
		}
		if imp.Name != nil {
			return imp.Name.Name
		}
		return "envelope"
	}
	return ""
}

func stringLiteral(expr ast.Expr) (string, bool) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(lit.Value)
	return s, err == nil
}

// readBaseline reads testdata/legacy_messages.txt: one "file:literal" per
// line (file relative to apps/api); blank lines and lines starting with # are
// ignored.
func readBaseline(t *testing.T) map[string]bool {
	t.Helper()
	f, err := os.Open(legacyBaselinePath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	entries := map[string]bool{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.Contains(line, ".go:") {
			t.Fatalf("%s: malformed line %q, want file.go:literal", legacyBaselinePath, line)
		}
		if entries[line] {
			t.Errorf("%s: duplicate line %q", legacyBaselinePath, line)
		}
		entries[line] = true
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(fmt.Errorf("read %s: %w", legacyBaselinePath, err))
	}
	return entries
}
