package pdfrender

import (
	"bytes"
	"fmt"
	"html"
	"html/template"
	"regexp"
	"strings"
)

// Rule names the check an uploaded template failed.
type Rule string

const (
	// RuleParse: the template is not valid html/template syntax.
	RuleParse Rule = "parse"
	// RuleExternalURL: the template loads a resource from the network (http(s)://
	// or protocol-relative //). The PDF converter blocks them; inline them as
	// data: URIs instead.
	RuleExternalURL Rule = "external_url"
	// RuleExecute: the template fails with the document's data, usually a field
	// the document doesn't have.
	RuleExecute Rule = "execute"
	// RuleRequired: the output lacks a value the document must show (the
	// electronic signature, the RIDE access key).
	RuleRequired Rule = "required"
)

// ValidationError is why a template was rejected. Detail is what to show the
// user next to the rule's message: the parse error, the offending reference,
// the failing field.
type ValidationError struct {
	Rule   Rule
	Detail string
	Err    error
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("pdfrender: template rejected (%s): %s", e.Rule, e.Detail)
}

func (e *ValidationError) Unwrap() error { return e.Err }

// externalRef matches a network reference in an attribute (src=, href=, ...),
// a CSS url( or an @import: http://, https:// or protocol-relative //.
var externalRef = regexp.MustCompile(`(?i)(?:\b(?:src|href|srcset|poster|action|background)\s*=\s*["']?\s*|url\(\s*["']?\s*|@import\s+["']?\s*)(?:https?:)?//[^\s"')>]*`)

// failingField extracts "<.Field>" from a text/template execution error.
var failingField = regexp.MustCompile(`at <([^>]+)>`)

// Validate checks that src is a template that will render doc: it parses, loads
// nothing from the network, executes with doc.Sample (and every Variant), and
// its output contains every doc.Required value. Errors are *ValidationError.
func Validate(doc Document, src []byte) error {
	tmpl, err := template.New(doc.File).Parse(string(src))
	if err != nil {
		return &ValidationError{Rule: RuleParse, Detail: strings.TrimPrefix(err.Error(), "template: "), Err: err}
	}
	if ref := externalRef.Find(src); ref != nil {
		return &ValidationError{Rule: RuleExternalURL, Detail: truncate(string(ref), 120)}
	}

	var out bytes.Buffer
	if err := tmpl.Execute(&out, doc.Sample); err != nil {
		return executeError(err)
	}
	for _, v := range doc.Variants {
		if err := tmpl.Execute(&bytes.Buffer{}, v); err != nil {
			return executeError(err)
		}
	}

	// html/template escapes values (a "+" in a data URI becomes "&#43;"):
	// compare against the text the PDF will show.
	text := html.UnescapeString(out.String())
	for _, req := range doc.Required {
		if !strings.Contains(text, req.Value) {
			return &ValidationError{Rule: RuleRequired, Detail: req.Field}
		}
	}
	return nil
}

func executeError(err error) *ValidationError {
	detail := strings.TrimPrefix(err.Error(), "template: ")
	if m := failingField.FindStringSubmatch(err.Error()); m != nil {
		detail = m[1]
	}
	return &ValidationError{Rule: RuleExecute, Detail: detail, Err: err}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
