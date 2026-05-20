package company_models

import "gorm.io/gorm"

type SubscriptionPayment struct {
	gorm.Model
	CompanyID       uint         `json:"company_id"`
	SubscriptionID  uint         `json:"subscription_id"`
	Subscription    Subscription `gorm:"foreignKey:SubscriptionID" json:"subscription"`
	DlocalPaymentID string       `json:"dlocal_payment_id"`
	OrderID         string       `gorm:"uniqueIndex" json:"order_id"`
	Amount          float64      `json:"amount"`
	Currency        string       `json:"currency"`
	Status          string       `json:"status"`
	CheckoutURL     string       `json:"checkout_url"`
	TargetPlanCode  string       `json:"target_plan_code"`
	Months          int          `gorm:"default:1" json:"months"`
}
