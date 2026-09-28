// Package catalog is the Message catalog: every user-facing text of the API
// and the web apps, keyed by i18n key, per language.
//
// The texts live in the JSON files embedded in the binary (i18n/messages);
// there is no table and no hot reload, so changing a text needs a deploy
// (docs/adr/0003-catalogo-de-mensajes-en-el-binario.md). main builds one
// Catalog with Load and hands it to the i18n middleware and handler; tests
// build one from a map with New.
package catalog

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
	"sync"

	"go.uber.org/zap"
)

// DefaultLanguage is the language used when the requested one is unsupported
// and the fallback for keys missing in the requested language.
const DefaultLanguage = "es"

// Catalog holds the messages of every supported language. It is immutable
// after construction and safe for concurrent use.
type Catalog struct {
	messages map[string]map[string]string // lang -> key -> value
	hashes   map[string]string            // lang -> content hash
	logger   *zap.Logger
	warned   sync.Map // missing key -> struct{}
}

// Load reads every messages_<lang>.json file at the root of fsys. Each file
// is a JSON array of {"key", "value"} objects. It fails on invalid JSON, an
// empty or duplicate key within a file, or a missing default language.
func Load(fsys fs.FS) (*Catalog, error) {
	files, err := fs.Glob(fsys, "messages_*.json")
	if err != nil {
		return nil, err
	}
	messages := make(map[string]map[string]string, len(files))
	for _, name := range files {
		lang := strings.TrimSuffix(strings.TrimPrefix(path.Base(name), "messages_"), ".json")
		data, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", name, err)
		}
		var entries []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}
		if err := json.Unmarshal(data, &entries); err != nil {
			return nil, fmt.Errorf("parse %s: %w", name, err)
		}
		bundle := make(map[string]string, len(entries))
		for i, e := range entries {
			if e.Key == "" {
				return nil, fmt.Errorf("%s: entry %d has an empty key", name, i)
			}
			if _, dup := bundle[e.Key]; dup {
				return nil, fmt.Errorf("%s: duplicate key %q", name, e.Key)
			}
			bundle[e.Key] = e.Value
		}
		messages[lang] = bundle
	}
	if _, ok := messages[DefaultLanguage]; !ok {
		return nil, fmt.Errorf("no messages_%s.json: the default language is required", DefaultLanguage)
	}
	return New(messages), nil
}

// New builds a Catalog from an in-memory map lang -> key -> value. The map is
// copied.
func New(messages map[string]map[string]string) *Catalog {
	c := &Catalog{
		messages: make(map[string]map[string]string, len(messages)),
		hashes:   make(map[string]string, len(messages)),
		logger:   zap.NewNop(),
	}
	for lang, bundle := range messages {
		copied := make(map[string]string, len(bundle))
		for k, v := range bundle {
			copied[k] = v
		}
		c.messages[lang] = copied
		c.hashes[lang] = hashBundle(copied)
	}
	return c
}

// WithLogger sets the logger that reports missing keys and returns c.
func (c *Catalog) WithLogger(logger *zap.Logger) *Catalog {
	if logger != nil {
		c.logger = logger
	}
	return c
}

// Translate returns the text of key in lang, falling back to the default
// language and then to the key itself. A missing key is logged once.
func (c *Catalog) Translate(lang, key string) string {
	if key == "" {
		return ""
	}
	if v, ok := c.messages[lang][key]; ok {
		return v
	}
	if v, ok := c.messages[DefaultLanguage][key]; ok {
		return v
	}
	if _, already := c.warned.LoadOrStore(key, struct{}{}); !already {
		c.logger.Warn("i18n key missing from the message catalog", zap.String("key", key), zap.String("lang", lang))
	}
	return key
}

// Bundle returns a copy of every message of lang (the default language when
// lang is unsupported) and a stable hash of its content, usable as an ETag.
func (c *Catalog) Bundle(lang string) (map[string]string, string) {
	lang = c.supportedOrDefault(lang)
	src := c.messages[lang]
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out, c.hashes[lang]
}

// Hash returns the content hash of lang's bundle (the default language when
// lang is unsupported) without copying it.
func (c *Catalog) Hash(lang string) string {
	return c.hashes[c.supportedOrDefault(lang)]
}

// Keys returns the keys of lang, sorted; nil when lang is unsupported.
func (c *Catalog) Keys(lang string) []string {
	bundle, ok := c.messages[lang]
	if !ok {
		return nil
	}
	keys := make([]string, 0, len(bundle))
	for k := range bundle {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Languages returns the supported languages, sorted.
func (c *Catalog) Languages() []string {
	langs := make([]string, 0, len(c.messages))
	for lang := range c.messages {
		langs = append(langs, lang)
	}
	sort.Strings(langs)
	return langs
}

// Supports reports whether lang has a messages file.
func (c *Catalog) Supports(lang string) bool {
	_, ok := c.messages[lang]
	return ok
}

func (c *Catalog) supportedOrDefault(lang string) string {
	if c.Supports(lang) {
		return lang
	}
	return DefaultLanguage
}

// hashBundle is the hex-encoded prefix of sha256 over the sorted key/value
// pairs, so it only changes when the content does.
func hashBundle(bundle map[string]string) string {
	keys := make([]string, 0, len(bundle))
	for k := range bundle {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	h := sha256.New()
	for _, k := range keys {
		h.Write([]byte(k))
		h.Write([]byte{0})
		h.Write([]byte(bundle[k]))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}
