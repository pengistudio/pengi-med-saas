package core_errors

type AppError struct {
	ErrorCode    string `json:"error_code"`
	ErrorMessage string `json:"error_message"`
	// Detail is optional untranslated context for the user (e.g. which field
	// of an uploaded template failed). See WithDetail.
	Detail string `json:"detail,omitempty"`
}

// WithDetail returns a copy of e carrying detail.
func (e AppError) WithDetail(detail string) AppError {
	e.Detail = detail
	return e
}

func NewAppError(code string, message string) AppError {
	return AppError{
		ErrorCode:    code,
		ErrorMessage: message,
	}
}
