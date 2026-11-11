// Package reports_templates embeds the default report templates (revenue,
// receivables, appointments, clinical) into the binary, so they can never be
// missing from a deployed image.
package reports_templates

import "embed"

//go:embed *.html
var FS embed.FS
