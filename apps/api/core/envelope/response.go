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
	if message == "" {
		message = http.StatusText(code)
	}

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
