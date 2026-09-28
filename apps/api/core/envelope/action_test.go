package envelope_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"

	"github.com/gin-gonic/gin"
)

// serve runs action behind a translator that knows only the given keys and
// returns the key itself otherwise (as the message catalog does).
func serve(t *testing.T, known map[string]string, action envelope.Action) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("translator", func(key string) string {
			if v, ok := known[key]; ok {
				return v
			}
			return key
		})
	})
	r.GET("/", envelope.Handle(action))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	var body map[string]any
	if w.Body.Len() > 0 {
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode %q: %v", w.Body.String(), err)
		}
	}
	return w, body
}

func TestHandle_TranslatesMessageAndErrorCode(t *testing.T) {
	appErr := core_errors.NewAppError("E-X-001", "English default.")
	_, body := serve(t, map[string]string{"x.fail": "Falló", "E-X-001": "Traducido."}, func(*gin.Context) envelope.Response {
		return envelope.ErrorResponse(http.StatusBadRequest, "x.fail", appErr)
	})
	if body["message"] != "Falló" {
		t.Errorf("message = %v", body["message"])
	}
	data := body["data"].(map[string]any)
	if data["error_message"] != "Traducido." || data["error_code"] != "E-X-001" {
		t.Errorf("data = %v", data)
	}
}

func TestHandle_KeepsEnglishDefaultWhenErrorCodeIsUntranslated(t *testing.T) {
	appErr := core_errors.NewAppError("E-X-404", "English default.")
	_, body := serve(t, nil, func(*gin.Context) envelope.Response {
		return envelope.ErrorResponse(http.StatusBadRequest, "x.fail", appErr)
	})
	data := body["data"].(map[string]any)
	if data["error_message"] != "English default." {
		t.Errorf("error_message = %v, want the NewAppError default", data["error_message"])
	}
}

func TestHandle_NotModifiedHasNoBody(t *testing.T) {
	w, _ := serve(t, nil, func(*gin.Context) envelope.Response { return envelope.NotModified() })
	if w.Code != http.StatusNotModified || w.Body.Len() != 0 {
		t.Errorf("status = %d, body = %q", w.Code, w.Body.String())
	}
}
