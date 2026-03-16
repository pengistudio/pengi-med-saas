package billing_handlers

import (
	"net/http"
	"strconv"

	"pengi-med-saas/core/envelope"
	core_errors "pengi-med-saas/core/errors"
	billing_dto "pengi-med-saas/features/billing/dto"
	billing_models "pengi-med-saas/features/billing/models"
	tenant_middleware "pengi-med-saas/features/tenants/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CatalogItemHandler struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewCatalogItemHandler(db *gorm.DB, logger *zap.Logger) *CatalogItemHandler {
	return &CatalogItemHandler{db: db, logger: logger}
}

func (h *CatalogItemHandler) CreateCatalogItem(c *gin.Context) envelope.Response {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		return envelope.ErrorResponse(http.StatusUnauthorized, "Tenant scope not found", core_errors.ErrTenantNotFound)
	}
	tenantScope := tenant_middleware.TenantScope(c)

	var dto billing_dto.CreateCatalogItemDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		h.logger.Error("Failed to bind CreateCatalogItem DTO", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid provided payload", core_errors.ErrBillingInvalidRequest)
	}

	item := &billing_models.CatalogItem{
		TenantID:  tenantID.(uint),
		Name:      dto.Name,
		SKU:       dto.SKU,
		UnitPrice: dto.UnitPrice,
	}

	if dto.Description != nil {
		item.Description = *dto.Description
	}
	if dto.Tax != nil {
		item.Tax = *dto.Tax
	}
	if dto.TaxCode != nil {
		item.TaxCode = *dto.TaxCode
	}
	if dto.TaxPercentageCode != nil {
		item.TaxPercentageCode = *dto.TaxPercentageCode
	}
	if dto.IceTax != nil {
		item.IceTax = *dto.IceTax
	}
	if dto.IceTaxCode != nil {
		item.IceTaxCode = *dto.IceTaxCode
	}
	if dto.IceTaxPercentageCode != nil {
		item.IceTaxPercentageCode = *dto.IceTaxPercentageCode
	}

	if err := h.db.Scopes(tenantScope).Create(item).Error; err != nil {
		h.logger.Error("Failed to create CatalogItem", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Failed to create catalog item", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(item, "billing.catalog_item.create.success")
}

func (h *CatalogItemHandler) GetAllCatalogItems(c *gin.Context) envelope.Response {
	tenantScope := tenant_middleware.TenantScope(c)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	search := c.Query("search")
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	baseQuery := h.db.Scopes(tenantScope).Model(&billing_models.CatalogItem{})
	if search != "" {
		like := "%" + search + "%"
		baseQuery = baseQuery.Where("name ILIKE ? OR sku ILIKE ?", like, like)
	}

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		h.logger.Error("Failed to count catalog items", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Failed to count catalog items", core_errors.ErrInternal)
	}

	var items []billing_models.CatalogItem
	if err := baseQuery.Order("name ASC").Limit(limit).Offset(offset).Find(&items).Error; err != nil {
		h.logger.Error("Failed to fetch catalog items", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Failed to fetch catalog items", core_errors.ErrInternal)
	}

	return envelope.PagedSuccessResponse(items, int(total), page, limit, "billing.catalog_items.fetch.success")
}

func (h *CatalogItemHandler) GetCatalogItemByID(c *gin.Context) envelope.Response {
	tenantScope := tenant_middleware.TenantScope(c)
	itemID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid catalog item ID", core_errors.ErrBillingInvalidRequest)
	}

	var item billing_models.CatalogItem
	if err := h.db.Scopes(tenantScope).First(&item, itemID).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "Catalog item not found", core_errors.ErrBillingProductNotFound)
	}

	return envelope.SuccessResponse(item, "billing.catalog_item.fetch.success")
}

func (h *CatalogItemHandler) UpdateCatalogItem(c *gin.Context) envelope.Response {
	tenantScope := tenant_middleware.TenantScope(c)
	itemID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid catalog item ID", core_errors.ErrBillingInvalidRequest)
	}

	var dto billing_dto.UpdateCatalogItemDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		h.logger.Error("Failed to bind UpdateCatalogItem DTO", zap.Error(err))
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid provided payload", core_errors.ErrBillingInvalidRequest)
	}

	var item billing_models.CatalogItem
	if err := h.db.Scopes(tenantScope).First(&item, itemID).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "Catalog item not found", core_errors.ErrBillingProductNotFound)
	}

	if dto.Name != nil {
		item.Name = *dto.Name
	}
	if dto.SKU != nil {
		item.SKU = *dto.SKU
	}
	if dto.Description != nil {
		item.Description = *dto.Description
	}
	if dto.UnitPrice != nil {
		item.UnitPrice = *dto.UnitPrice
	}
	if dto.Tax != nil {
		item.Tax = *dto.Tax
	}
	if dto.TaxCode != nil {
		item.TaxCode = *dto.TaxCode
	}
	if dto.TaxPercentageCode != nil {
		item.TaxPercentageCode = *dto.TaxPercentageCode
	}
	if dto.IceTax != nil {
		item.IceTax = *dto.IceTax
	}
	if dto.IceTaxCode != nil {
		item.IceTaxCode = *dto.IceTaxCode
	}
	if dto.IceTaxPercentageCode != nil {
		item.IceTaxPercentageCode = *dto.IceTaxPercentageCode
	}

	if err := h.db.Scopes(tenantScope).Save(&item).Error; err != nil {
		h.logger.Error("Failed to update CatalogItem", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Failed to update catalog item", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(item, "billing.catalog_item.update.success")
}

func (h *CatalogItemHandler) DeleteCatalogItem(c *gin.Context) envelope.Response {
	tenantScope := tenant_middleware.TenantScope(c)
	itemID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return envelope.ErrorResponse(http.StatusBadRequest, "Invalid catalog item ID", core_errors.ErrBillingInvalidRequest)
	}

	var item billing_models.CatalogItem
	if err := h.db.Scopes(tenantScope).First(&item, itemID).Error; err != nil {
		return envelope.ErrorResponse(http.StatusNotFound, "Catalog item not found", core_errors.ErrBillingProductNotFound)
	}

	if err := h.db.Scopes(tenantScope).Delete(&item).Error; err != nil {
		h.logger.Error("Failed to delete CatalogItem", zap.Error(err))
		return envelope.ErrorResponse(http.StatusInternalServerError, "Failed to delete catalog item", core_errors.ErrInternal)
	}

	return envelope.SuccessResponse(item, "billing.catalog_item.delete.success")
}
