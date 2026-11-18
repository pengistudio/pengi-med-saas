package billing_handlers

import (
	"errors"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"unicode/utf8"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	"pengi-med-saas/core/tenantdb"
	billing_dto "pengi-med-saas/features/billing/dto"
	billing_models "pengi-med-saas/features/billing/models"
	sri_services "pengi-med-saas/features/billing/sri/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// CustomerHandler manages the tenant's third-party buyers (billing_customers).
type CustomerHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewCustomerHandler(db *gorm.DB, logger *zap.Logger) *CustomerHandler {
	return &CustomerHandler{db: db, logger: logger}
}

func customerNotFound() envelope.Response {
	return envelope.ErrorResponse(http.StatusNotFound, "billing.customer.error.not_found", core_errors.ErrBillingCustomerNotFound)
}

func customerInvalid() envelope.Response {
	return envelope.ErrorResponse(http.StatusBadRequest, "billing.customer.error.invalid_payload", core_errors.ErrBillingCustomerInvalid)
}

func customerDuplicate() envelope.Response {
	return envelope.ErrorResponse(http.StatusConflict, "billing.customer.error.duplicate_identification", core_errors.ErrBillingCustomerDuplicate)
}

// normalizeCustomer trims the customer's fields and validates them; a non-nil
// response is the error to answer.
func normalizeCustomer(c *billing_models.Customer) *envelope.Response {
	c.Identification = strings.TrimSpace(c.Identification)
	c.Name = strings.TrimSpace(c.Name)
	c.Address = strings.TrimSpace(c.Address)
	c.Email = strings.TrimSpace(c.Email)
	c.Phone = strings.TrimSpace(c.Phone)

	if c.Name == "" || c.Identification == "" ||
		utf8.RuneCountInString(c.Name) > billing_models.CustomerNameMaxLength ||
		utf8.RuneCountInString(c.Address) > billing_models.CustomerContactMaxLength ||
		utf8.RuneCountInString(c.Email) > billing_models.CustomerContactMaxLength ||
		utf8.RuneCountInString(c.Phone) > billing_models.CustomerPhoneMaxLength {
		resp := customerInvalid()
		return &resp
	}
	if c.Email != "" {
		if addr, err := mail.ParseAddress(c.Email); err != nil || addr.Address != c.Email {
			resp := customerInvalid()
			return &resp
		}
	}
	if !sri_services.ValidateCustomerIdentification(c.Identification) {
		resp := envelope.ErrorResponse(http.StatusBadRequest, "billing.customer.error.invalid_identification", core_errors.ErrBillingCustomerInvalidID)
		return &resp
	}
	return nil
}

// identificationTaken reports whether another live customer of the tenant
// already has the identification.
func identificationTaken(db *gorm.DB, identification string, exceptID uint) (bool, error) {
	var count int64
	q := db.Model(&billing_models.Customer{}).Where("identification = ?", identification)
	if exceptID != 0 {
		q = q.Where("id <> ?", exceptID)
	}
	err := q.Count(&count).Error
	return count > 0, err
}

// isUniqueViolation reports a unique index violation (a concurrent duplicate).
func isUniqueViolation(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "duplicate key") || strings.Contains(msg, "UNIQUE constraint failed")
}

func (h *CustomerHandler) save(db *gorm.DB, customer *billing_models.Customer, create bool) *envelope.Response {
	taken, err := identificationTaken(db, customer.Identification, customer.ID)
	if err != nil {
		h.logger.Error("Failed to check customer identification", zap.Error(err))
		resp := envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
		return &resp
	}
	if taken {
		resp := customerDuplicate()
		return &resp
	}
	if create {
		err = db.Create(customer).Error
	} else {
		err = db.Save(customer).Error
	}
	if err != nil {
		if isUniqueViolation(err) {
			resp := customerDuplicate()
			return &resp
		}
		h.logger.Error("Failed to save customer", zap.Error(err))
		resp := envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
		return &resp
	}
	return nil
}

func (h *CustomerHandler) CreateCustomer(c *gin.Context) envelope.Response {
	db := tenantdb.For(c, h.db)

	var dto billing_dto.CreateCustomerDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		return customerInvalid()
	}
	customer := &billing_models.Customer{
		TenantID:       tenantdb.TenantID(c),
		Identification: dto.Identification,
		Name:           dto.Name,
		Address:        dto.Address,
		Email:          dto.Email,
		Phone:          dto.Phone,
	}
	if resp := normalizeCustomer(customer); resp != nil {
		return *resp
	}
	if resp := h.save(db, customer, true); resp != nil {
		return *resp
	}
	return envelope.SuccessResponse(customer, "billing.customer.create.success")
}

// GetAllCustomers lists the tenant's customers by name, paginated; search
// matches name or identification (case-insensitive).
func (h *CustomerHandler) GetAllCustomers(c *gin.Context) envelope.Response {
	db := tenantdb.For(c, h.db)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	baseQuery := db.Model(&billing_models.Customer{})
	if search := strings.TrimSpace(c.Query("search")); search != "" {
		like := "%" + strings.ToLower(search) + "%"
		baseQuery = baseQuery.Where("LOWER(name) LIKE ? OR LOWER(identification) LIKE ?", like, like)
	}

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		h.logger.Error("Failed to count customers", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}

	var customers []billing_models.Customer
	if err := baseQuery.Order("name ASC").Limit(limit).Offset(offset).Find(&customers).Error; err != nil {
		h.logger.Error("Failed to fetch customers", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}

	return envelope.PagedSuccessResponse(customers, int(total), page, limit, "billing.customers.fetch.success")
}

// find loads the request tenant's customer :id.
func (h *CustomerHandler) find(c *gin.Context, db *gorm.DB) (*billing_models.Customer, *envelope.Response) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		resp := customerInvalid()
		return nil, &resp
	}
	var customer billing_models.Customer
	if err := db.First(&customer, id).Error; err != nil {
		resp := customerNotFound()
		return nil, &resp
	}
	return &customer, nil
}

func (h *CustomerHandler) GetCustomerByID(c *gin.Context) envelope.Response {
	customer, resp := h.find(c, tenantdb.For(c, h.db))
	if resp != nil {
		return *resp
	}
	return envelope.SuccessResponse(customer, "billing.customer.fetch.success")
}

// UpdateCustomer changes the fields sent. Invoices already received by the SRI
// keep the buyer they were sent with (their buyer snapshot); drafts and
// rejected ones take the new data on their next send.
func (h *CustomerHandler) UpdateCustomer(c *gin.Context) envelope.Response {
	db := tenantdb.For(c, h.db)

	var dto billing_dto.UpdateCustomerDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		return customerInvalid()
	}
	customer, resp := h.find(c, db)
	if resp != nil {
		return *resp
	}
	if dto.Identification != nil {
		customer.Identification = *dto.Identification
	}
	if dto.Name != nil {
		customer.Name = *dto.Name
	}
	if dto.Address != nil {
		customer.Address = *dto.Address
	}
	if dto.Email != nil {
		customer.Email = *dto.Email
	}
	if dto.Phone != nil {
		customer.Phone = *dto.Phone
	}
	if resp := normalizeCustomer(customer); resp != nil {
		return *resp
	}
	if resp := h.save(db, customer, false); resp != nil {
		return *resp
	}
	return envelope.SuccessResponse(customer, "billing.customer.update.success")
}

// DeleteCustomer soft-deletes the customer. Its invoices keep it (and their
// buyer snapshot); its identification can be registered again.
func (h *CustomerHandler) DeleteCustomer(c *gin.Context) envelope.Response {
	db := tenantdb.For(c, h.db)
	customer, resp := h.find(c, db)
	if resp != nil {
		return *resp
	}
	if err := db.Delete(customer).Error; err != nil {
		h.logger.Error("Failed to delete customer", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "error.internal", core_errors.ErrInternal)
	}
	return envelope.SuccessResponse(customer, "billing.customer.delete.success")
}
