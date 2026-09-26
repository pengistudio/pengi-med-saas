// Package billing_templates embeds the default billing document templates (the
// RIDE) into the binary, so they can never be missing from a deployed image.
package billing_templates

import "embed"

//go:embed *.html
var FS embed.FS
