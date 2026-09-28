package catalog_test

import (
	"reflect"
	"strings"
	"testing"
	"testing/fstest"

	"pengi-med-saas/i18n/catalog"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func newTestCatalog() *catalog.Catalog {
	return catalog.New(map[string]map[string]string{
		"es": {"greeting": "Hola", "only.es": "Solo en español"},
		"en": {"greeting": "Hello"},
	})
}

func TestLoad_ReadsEveryMessagesFile(t *testing.T) {
	fsys := fstest.MapFS{
		"messages_es.json": {Data: []byte(`[{"key": "a", "value": "uno"}, {"key": "b", "value": "dos"}]`)},
		"messages_en.json": {Data: []byte(`[{"key": "a", "value": "one"}, {"key": "b", "value": "two"}]`)},
		"README.md":        {Data: []byte("ignored")},
	}
	c, err := catalog.Load(fsys)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := c.Languages(); !reflect.DeepEqual(got, []string{"en", "es"}) {
		t.Errorf("Languages() = %v", got)
	}
	if got := c.Translate("en", "b"); got != "two" {
		t.Errorf("Translate(en, b) = %q", got)
	}
	if got := c.Keys("es"); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Errorf("Keys(es) = %v", got)
	}
}

func TestLoad_Errors(t *testing.T) {
	cases := map[string]fstest.MapFS{
		"invalid json": {
			"messages_es.json": {Data: []byte(`[{"key": "a", "value": "uno"`)},
		},
		"duplicate key": {
			"messages_es.json": {Data: []byte(`[{"key": "a", "value": "uno"}, {"key": "a", "value": "otro"}]`)},
		},
		"empty key": {
			"messages_es.json": {Data: []byte(`[{"key": "", "value": "uno"}]`)},
		},
		"no default language": {
			"messages_en.json": {Data: []byte(`[{"key": "a", "value": "one"}]`)},
		},
	}
	for name, fsys := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := catalog.Load(fsys); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}

func TestTranslate_FallsBackToDefaultLanguageThenKey(t *testing.T) {
	c := newTestCatalog()
	cases := []struct{ lang, key, want string }{
		{"en", "greeting", "Hello"},
		{"es", "greeting", "Hola"},
		{"en", "only.es", "Solo en español"},
		{"fr", "greeting", "Hola"},
		{"en", "missing.key", "missing.key"},
		{"en", "", ""},
	}
	for _, tc := range cases {
		if got := c.Translate(tc.lang, tc.key); got != tc.want {
			t.Errorf("Translate(%q, %q) = %q, want %q", tc.lang, tc.key, got, tc.want)
		}
	}
}

func TestTranslate_WarnsOncePerMissingKey(t *testing.T) {
	core, logs := observer.New(zap.WarnLevel)
	c := newTestCatalog().WithLogger(zap.New(core))

	c.Translate("en", "missing.key")
	c.Translate("es", "missing.key")
	c.Translate("en", "other.missing")
	c.Translate("en", "only.es") // present in the default language: no warning
	c.Translate("en", "")        // nothing to translate: no warning

	if got := logs.Len(); got != 2 {
		t.Fatalf("got %d warnings, want 2: %v", got, logs.All())
	}
}

func TestBundle_ReturnsFlatMapAndStableHash(t *testing.T) {
	c := newTestCatalog()
	en, enHash := c.Bundle("en")
	if !reflect.DeepEqual(en, map[string]string{"greeting": "Hello"}) {
		t.Errorf("Bundle(en) = %v", en)
	}
	es, esHash := c.Bundle("es")
	if len(es) != 2 {
		t.Errorf("Bundle(es) = %v", es)
	}
	if enHash == "" || enHash == esHash {
		t.Errorf("hashes must be non-empty and differ per language: en=%q es=%q", enHash, esHash)
	}
	if strings.ContainsAny(enHash, `" `) {
		t.Errorf("hash must be usable inside an ETag: %q", enHash)
	}

	// Same content → same hash, whatever the map order.
	_, again := catalog.New(map[string]map[string]string{"en": {"greeting": "Hello"}, "es": {}}).Bundle("en")
	if again != enHash {
		t.Errorf("hash not stable: %q vs %q", again, enHash)
	}
	// Different content → different hash.
	_, changed := catalog.New(map[string]map[string]string{"en": {"greeting": "Hi"}, "es": {}}).Bundle("en")
	if changed == enHash {
		t.Error("hash did not change with content")
	}

	// The returned map is a copy.
	en["greeting"] = "mutated"
	if c.Translate("en", "greeting") != "Hello" {
		t.Error("Bundle leaked the internal map")
	}
}

func TestBundle_UnsupportedLanguageUsesDefault(t *testing.T) {
	c := newTestCatalog()
	fr, frHash := c.Bundle("fr")
	es, esHash := c.Bundle("es")
	if !reflect.DeepEqual(fr, es) || frHash != esHash {
		t.Errorf("Bundle(fr) should be the default language bundle")
	}
}

func TestResolveLanguage(t *testing.T) {
	c := newTestCatalog()
	cases := []struct {
		name, query, header, want string
	}{
		{"default", "", "", "es"},
		{"query wins", "en", "es-EC,es;q=0.9", "en"},
		{"unsupported query falls through to header", "fr", "en-US,en;q=0.9", "en"},
		{"region tag", "", "es-EC,es;q=0.9", "es"},
		{"first supported", "", "fr-FR,fr;q=0.9,en;q=0.8", "en"},
		{"q ordering", "", "es;q=0.2,en;q=0.9", "en"},
		{"q=0 is excluded", "", "en;q=0,fr", "es"},
		{"case insensitive", "", "EN-us", "en"},
		{"wildcard ignored", "", "*", "es"},
		{"plain", "", "en", "en"},
		{"garbage", "", ";;,q=,", "es"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := c.ResolveLanguage(tc.query, tc.header); got != tc.want {
				t.Errorf("ResolveLanguage(%q, %q) = %q, want %q", tc.query, tc.header, got, tc.want)
			}
		})
	}
}
