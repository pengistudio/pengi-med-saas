package backoffice_dto

type LoginDTO struct {
	UserName string `json:"user_name" form:"user_name"`
	Password string `json:"password" form:"password"`
}
