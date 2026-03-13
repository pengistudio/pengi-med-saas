package dto

import "time"

type CreatePatientDTO struct {
	Document    string     `json:"document" binding:"required"`
	Phone       string     `json:"phone"`
	Email       string     `json:"email"`
	FirstName   string     `json:"first_name" binding:"required"`
	LastName    string     `json:"last_name" binding:"required"`
	BirthDate   *time.Time `json:"birth_date"`
	Institution string     `json:"institution" binding:"required"`
	Gender      string     `json:"gender"`
	Notes       string     `json:"notes"`
	Insurance   string     `json:"insurance"`
	Medic       string     `json:"medic"`
	Diagnosis   string     `json:"diagnosis"`
	APP         string     `json:"app"`
	APF         string     `json:"apf"`
	APQX        string     `json:"apqx"`
	Allergies   string     `json:"allergies"`
}

type UpdatePatientDTO struct {
	Document    *string    `json:"document"`
	Phone       *string    `json:"phone"`
	Email       *string    `json:"email"`
	FirstName   *string    `json:"first_name"`
	LastName    *string    `json:"last_name"`
	BirthDate   *time.Time `json:"birth_date"`
	Institution *string    `json:"institution"`
	Gender      *string    `json:"gender"`
	Notes       *string    `json:"notes"`
	Insurance   *string    `json:"insurance"`
	Medic       *string    `json:"medic"`
	Diagnosis   *string    `json:"diagnosis"`
	APP         *string    `json:"app"`
	APF         *string    `json:"apf"`
	APQX        *string    `json:"apqx"`
	Allergies   *string    `json:"allergies"`
}

type DeletePatientsDTO struct {
	IdList []uint `json:"id_list" binding:"required"`
}
