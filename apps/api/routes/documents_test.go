package routes

import (
	"testing"

	"pengi-med-saas/i18n/catalog"
	i18n_messages "pengi-med-saas/i18n/messages"
)

// Settings renders each document's name as document_templates.doc.<id> with a
// dynamic key, which the literal-key tests can't see: a new printable document
// without its translation would show the raw key.
func TestPrintableDocuments_NamesAreTranslated(t *testing.T) {
	c, err := catalog.Load(i18n_messages.FS)
	if err != nil {
		t.Fatalf("load embedded catalog: %v", err)
	}
	for _, lang := range c.Languages() {
		bundle, _ := c.Bundle(lang)
		for _, doc := range printableDocuments {
			key := "document_templates.doc." + doc.ID
			if bundle[key] == "" {
				t.Errorf("%s: missing %q", lang, key)
			}
		}
	}
}
