// Package clinical_templates embeds the default clinical document templates
// (prescription, medical report, medical certificate) into the binary, so they
// can never be missing from a deployed image.
package clinical_templates

import "embed"

//go:embed *.html
var FS embed.FS
