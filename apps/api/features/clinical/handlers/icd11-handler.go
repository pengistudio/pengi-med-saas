package clinical_handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"pengi-med-saas/core/config"

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
	cfg := config.Get()
	return &ICD11Handler{logger: logger, apiURL: cfg.ICD11APIURL, release: cfg.ICD11Release, language: cfg.ICD11Language}
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
		return envelope.ErrorResponse(http.StatusBadRequest, "error.invalid_request", core_errors.ErrClinicalInvalidRequest)
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
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	req.Header.Set("Api-Version", "v2")
	req.Header.Set("Accept-Language", lang)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		h.logger.Error("Failed to call ICD-11 API", zap.Error(err))
		return envelope.ErrorResponse(http.StatusServiceUnavailable, "clinical.icd11.error.unavailable", core_errors.ErrClinicalICD11Unavailable)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		h.logger.Error("Failed to read ICD-11 response", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}

	var apiResp icd11ApiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		h.logger.Error("Failed to parse ICD-11 response", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
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
