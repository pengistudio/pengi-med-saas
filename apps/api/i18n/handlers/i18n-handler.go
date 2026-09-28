package i18n_handlers

import (
	"strings"

	"pengi-med-saas/core/envelope"
	"pengi-med-saas/i18n/catalog"

	"github.com/gin-gonic/gin"
)

type MessageHandler struct {
	messages *catalog.Catalog
}

func NewMessageHandler(messages *catalog.Catalog) *MessageHandler {
	return &MessageHandler{messages: messages}
}

// GetAllMessages returns every message of the request language as a flat
// {key: value} map. The bundle's content hash travels in the ETag header; a
// request whose If-None-Match still matches gets 304 with no body.
func (h *MessageHandler) GetAllMessages(c *gin.Context) envelope.Response {
	lang := h.messages.ResolveLanguage(c.Query("lang"), c.GetHeader("Accept-Language"))
	hash := h.messages.Hash(lang)

	c.Header("ETag", `"`+hash+`"`)
	c.Header("Cache-Control", "no-cache")
	c.Header("Vary", "Accept-Language")

	if etagMatches(c.GetHeader("If-None-Match"), hash) {
		return envelope.NotModified()
	}

	bundle, _ := h.messages.Bundle(lang)
	return envelope.SuccessResponse(bundle, "i18n.messages.fetch.success")
}

// compressionSuffixes are appended to a strong ETag by a compressing proxy
// (Caddy's `encode` turns "<hash>" into "<hash>-gzip" / "<hash>-zstd"), so the
// browser may send them back in If-None-Match.
var compressionSuffixes = []string{"-gzip", "-zstd", "-br"}

// etagMatches reports whether an If-None-Match header ("*" or a list of
// possibly weak entity tags) matches hash.
func etagMatches(ifNoneMatch, hash string) bool {
	for _, tag := range strings.Split(ifNoneMatch, ",") {
		tag = strings.TrimSpace(tag)
		if tag == "*" {
			return true
		}
		tag = strings.Trim(strings.TrimPrefix(tag, "W/"), `"`)
		for _, suffix := range compressionSuffixes {
			tag = strings.TrimSuffix(tag, suffix)
		}
		if tag != "" && tag == hash {
			return true
		}
	}
	return false
}
