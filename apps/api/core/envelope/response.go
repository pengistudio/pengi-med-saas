package envelope

import (
	"fmt"
	"net/http"
	core_errors "pengi-med-saas/core/errors"
)

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func New(code int, message string, data interface{}) Response {
	return Response{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

func SuccessResponse(data any, message string) Response {
	return Response{
		Code:    http.StatusOK,
		Message: message,
		Data:    data,
	}
}

// NotModified answers a conditional request whose ETag still matches; Handle
// sends it with no body. The action sets the ETag header itself.
func NotModified() Response {
	return Response{Code: http.StatusNotModified}
}

type PagedData struct {
	Items      any `json:"items"`
	Total      int `json:"total"`
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalPages int `json:"total_pages"`
}

func PagedSuccessResponse(items any, total int, page int, limit int, message string) Response {
	totalPages := 0
	if limit > 0 {
		totalPages = (total + limit - 1) / limit
	}
	return Response{
		Code:    http.StatusOK,
		Message: message,
		Data: PagedData{
			Items:      items,
			Total:      total,
			Page:       page,
			Limit:      limit,
			TotalPages: totalPages,
		},
	}
}

func ErrorResponse(code int, message string, data core_errors.AppError) Response {
	return Response{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

func (r Response) Unwrap() error {
	if r.Code > 399 {
		appErr, ok := r.Data.(core_errors.AppError)
		if !ok {
			return fmt.Errorf("description: %s, invalid error data format", r.Message)
		}
		return fmt.Errorf("description: %s, metadata: %v", r.Message, appErr)
	}
	return nil
}

func (r Response) Error() string {
	return fmt.Sprintf("description: %s, metadata: %v", r.Message, r.Data)
}
