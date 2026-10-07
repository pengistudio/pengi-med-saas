package backoffice_dto

// LoginDTO requires both fields: an empty password must never reach the
// credential check (admins created before the CreateUser fix have the hash of
// an empty password).
type LoginDTO struct {
	UserName string `json:"user_name" form:"user_name" binding:"required"`
	Password string `json:"password" form:"password" binding:"required"`
}
