// Package cash_templates embeds the default cash document templates (the
// closing report of a cash session) into the binary, so they can never be
// missing from a deployed image.
package cash_templates

import "embed"

//go:embed *.html
var FS embed.FS
