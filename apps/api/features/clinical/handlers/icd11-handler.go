package clinical_handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type ICD11Handler struct {
	logger   *zap.Logger
	apiURL   string
	release  string
	language string
}

func NewICD11Handler(logger *zap.Logger) *ICD11Handler {
	apiURL := os.Getenv("ICD11_API_URL")
	if apiURL == "" {
		apiURL = "http://icd-api:80"
	}
	release := os.Getenv("ICD11_RELEASE")
	if release == "" {
		release = "2026-01"
	}
	language := os.Getenv("ICD11_LANGUAGE")
	if language == "" {
		language = "es"
	}
	return &ICD11Handler{logger: logger, apiURL: apiURL, release: release, language: language}
}

type icd11SearchResult struct {
	Code  string `json:"code"`
	Title string `json:"title"`
}

type icd11ApiResponse struct {
	DestinationEntities []struct {
		TheCode string `json:"theCode"`
		Title   string `json:"title"`
	} `json:"destinationEntities"`
}

func (h *ICD11Handler) Search(c *gin.Context) envelope.Response {
	q := c.Query("q")
	if q == "" {
		return envelope.ErrorResponse(http.StatusBadRequest, "query param 'q' is required", core_errors.ErrAuthInvalidRequest)
	}

	lang := c.Query("lang")
	if lang == "" {
		lang = h.language
	}

	searchURL := h.apiURL + "/icd/release/11/" + h.release + "/mms/search"
	params := url.Values{}
	params.Set("q", q)
	params.Set("flatResults", "true")
	params.Set("highlightingEnabled", "false")

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, searchURL+"?"+params.Encode(), nil)
	if err != nil {
		h.logger.Error("Failed to build ICD-11 request", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "icd11 request error", core_errors.ErrInternal)
	}
	req.Header.Set("Api-Version", "v2")
	req.Header.Set("Accept-Language", lang)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		h.logger.Error("Failed to call ICD-11 API", zap.Error(err))
		return envelope.ErrorResponse(http.StatusServiceUnavailable, "icd11 unavailable", core_errors.ErrInternal)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.logger.Error("Failed to read ICD-11 response", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "icd11 read error", core_errors.ErrInternal)
	}

	var apiResp icd11ApiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		h.logger.Error("Failed to parse ICD-11 response", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "icd11 parse error", core_errors.ErrInternal)
	}

	results := make([]icd11SearchResult, 0, len(apiResp.DestinationEntities))
	for _, e := range apiResp.DestinationEntities {
		if e.TheCode != "" {
			results = append(results, icd11SearchResult{
				Code:  e.TheCode,
				Title: e.Title,
			})
		}
	}

	return envelope.SuccessResponse(results, "clinical.icd11.search.success")
}
