package i18n_handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"pengi-med-saas/core/envelope"
	"pengi-med-saas/i18n/catalog"
	i18n_handlers "pengi-med-saas/i18n/handlers"
	i18n_middleware "pengi-med-saas/i18n/middleware"

	"github.com/gin-gonic/gin"
)

func newRouter(t *testing.T) (*gin.Engine, *catalog.Catalog) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	messages := catalog.New(map[string]map[string]string{
		"es": {"i18n.messages.fetch.success": "Mensajes cargados", "greeting": "Hola"},
		"en": {"i18n.messages.fetch.success": "Messages loaded", "greeting": "Hello"},
	})
	r := gin.New()
	r.Use(i18n_middleware.I18nMiddleware(messages))
	h := i18n_handlers.NewMessageHandler(messages)
	r.GET("/i18n/messages", envelope.Handle(h.GetAllMessages))
	return r, messages
}

func get(r *gin.Engine, target string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

type messagesBody struct {
	Code    int               `json:"code"`
	Message string            `json:"message"`
	Data    map[string]string `json:"data"`
}

func decode(t *testing.T, w *httptest.ResponseRecorder) messagesBody {
	t.Helper()
	var body messagesBody
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %q: %v", w.Body.String(), err)
	}
	return body
}

func TestGetAllMessages_ReturnsFlatMapWithETag(t *testing.T) {
	r, messages := newRouter(t)
	w := get(r, "/i18n/messages?lang=en", nil)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	body := decode(t, w)
	want := map[string]string{"i18n.messages.fetch.success": "Messages loaded", "greeting": "Hello"}
	if !reflect.DeepEqual(body.Data, want) {
		t.Errorf("data = %v, want %v", body.Data, want)
	}
	if body.Message != "Messages loaded" {
		t.Errorf("message = %q", body.Message)
	}
	_, hash := messages.Bundle("en")
	if got := w.Header().Get("ETag"); got != `"`+hash+`"` {
		t.Errorf("ETag = %q, want %q", got, `"`+hash+`"`)
	}
	if got := w.Header().Get("Cache-Control"); got != "no-cache" {
		t.Errorf("Cache-Control = %q", got)
	}
}

func TestGetAllMessages_ResolvesLanguage(t *testing.T) {
	r, _ := newRouter(t)
	cases := []struct {
		name    string
		target  string
		headers map[string]string
		want    string
	}{
		{"default", "/i18n/messages", nil, "Hola"},
		{"accept-language with region and q", "/i18n/messages", map[string]string{"Accept-Language": "en-US,en;q=0.9,es;q=0.8"}, "Hello"},
		{"query wins over header", "/i18n/messages?lang=es", map[string]string{"Accept-Language": "en-US"}, "Hola"},
		{"unsupported falls back to es", "/i18n/messages?lang=fr", nil, "Hola"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := decode(t, get(r, tc.target, tc.headers))
			if body.Data["greeting"] != tc.want {
				t.Errorf("greeting = %q, want %q", body.Data["greeting"], tc.want)
			}
		})
	}
}

func TestGetAllMessages_NotModifiedWhenETagMatches(t *testing.T) {
	r, _ := newRouter(t)
	etag := get(r, "/i18n/messages?lang=en", nil).Header().Get("ETag")

	hash := etag[1 : len(etag)-1]
	for _, ifNoneMatch := range []string{
		etag,
		"W/" + etag,
		`"other", ` + etag,
		"*",
		// Caddy's `encode` appends the encoding to a strong ETag.
		`"` + hash + `-gzip"`,
		`W/"` + hash + `-zstd"`,
		`"other", "` + hash + `-br"`,
	} {
		w := get(r, "/i18n/messages?lang=en", map[string]string{"If-None-Match": ifNoneMatch})
		if w.Code != http.StatusNotModified {
			t.Errorf("If-None-Match %s: status = %d, want 304", ifNoneMatch, w.Code)
		}
		if w.Body.Len() != 0 {
			t.Errorf("If-None-Match %s: 304 must have no body, got %q", ifNoneMatch, w.Body.String())
		}
		if got := w.Header().Get("ETag"); got != etag {
			t.Errorf("If-None-Match %s: ETag = %q", ifNoneMatch, got)
		}
	}
}

func TestGetAllMessages_FullResponseWhenETagIsStale(t *testing.T) {
	r, _ := newRouter(t)
	esETag := get(r, "/i18n/messages?lang=es", nil).Header().Get("ETag")

	for _, ifNoneMatch := range []string{`"stale"`, esETag, `""`, `"stale-gzip"`} {
		w := get(r, "/i18n/messages?lang=en", map[string]string{"If-None-Match": ifNoneMatch})
		if w.Code != http.StatusOK {
			t.Errorf("If-None-Match %s: status = %d, want 200", ifNoneMatch, w.Code)
		}
		if decode(t, w).Data["greeting"] != "Hello" {
			t.Errorf("If-None-Match %s: wrong bundle", ifNoneMatch)
		}
	}
}
