package i18n_messages

import "embed"

// FS contains the embedded JSON message files (messages_es.json, messages_en.json).
//go:embed messages_es.json messages_en.json
var FS embed.FS
