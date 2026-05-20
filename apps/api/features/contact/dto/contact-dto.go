package contact_dto

type ContactDTO struct {
	Name    string `json:"name" binding:"required,min=2,max=100"`
	Email   string `json:"email" binding:"required,email"`
	Message string `json:"message" binding:"required,min=10,max=2000"`
}
