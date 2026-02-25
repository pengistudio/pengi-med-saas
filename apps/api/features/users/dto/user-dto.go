package user_dto

import (
	company_models "pengi-med-saas/features/companies/models"
	user_models "pengi-med-saas/features/users/models"
)

type LoginDTO struct {
	UserName string `json:"user_name" form:"user_name"`
	Password string `json:"password" form:"password"`
}

type EnvironmentWithCompany struct {
	user_models.Environment
	Company company_models.Company `json:"company"`
}
