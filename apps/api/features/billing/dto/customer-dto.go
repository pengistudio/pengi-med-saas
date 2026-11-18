package billing_dto

// CreateCustomerDTO creates a third-party buyer. Strings are trimmed; name and
// identification are required.
type CreateCustomerDTO struct {
	Identification string `json:"identification" binding:"required"`
	Name           string `json:"name" binding:"required"`
	Address        string `json:"address"`
	Email          string `json:"email"`
	Phone          string `json:"phone"`
}

// UpdateCustomerDTO changes only the fields sent; name and identification can
// not be blanked.
type UpdateCustomerDTO struct {
	Identification *string `json:"identification"`
	Name           *string `json:"name"`
	Address        *string `json:"address"`
	Email          *string `json:"email"`
	Phone          *string `json:"phone"`
}
