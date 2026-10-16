package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	apperrors "github.com/javaknight1/servicepro/backend/internal/errors"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// TaxRateHandler serves the tenant's tax rates (T237).
type TaxRateHandler struct {
	service *services.TaxRateService
}

// NewTaxRateHandler creates a tax rate handler.
func NewTaxRateHandler(service *services.TaxRateService) *TaxRateHandler {
	return &TaxRateHandler{service: service}
}

// CreateTaxRateRequest is the body of POST /tax-rates.
type CreateTaxRateRequest struct {
	Name        string           `json:"name" binding:"required,max=100"`
	Description string           `json:"description"`
	Rate        *decimal.Decimal `json:"rate" binding:"required"` // fraction: 0.0825 = 8.25%
	TaxType     models.TaxType   `json:"tax_type"`                // default sales_tax
	Region      string           `json:"region" binding:"max=100"`
	IsCompound  bool             `json:"is_compound"`
	IsActive    *bool            `json:"is_active"` // default true
}

// UpdateTaxRateRequest is the body of PUT /tax-rates/{id}; omitted fields are unchanged.
type UpdateTaxRateRequest struct {
	Name        *string          `json:"name" binding:"omitempty,max=100"`
	Description *string          `json:"description"`
	Rate        *decimal.Decimal `json:"rate"`
	TaxType     *models.TaxType  `json:"tax_type"`
	Region      *string          `json:"region" binding:"omitempty,max=100"`
	IsCompound  *bool            `json:"is_compound"`
	IsActive    *bool            `json:"is_active"`
}

// errInvalidTaxRateID is the 400 for a malformed path ID.
var errInvalidTaxRateID = apperrors.BadRequest("invalid_id", "Invalid tax rate ID")

func taxRatePathID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		apierror.Respond(c, errInvalidTaxRateID, "")
		return uuid.Nil, false
	}
	return id, true
}

func taxRateTenant(c *gin.Context) (uuid.UUID, bool) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		apierror.Internal(c, err, "Tenant context missing")
		return uuid.Nil, false
	}
	return tenantID, true
}

// ListTaxRates godoc
// @Summary List tax rates
// @Description The tenant's own tax rates plus the global defaults (tenant_id omitted), ordered by name.
// @Tags tax-rates
// @Produce json
// @Param active_only query boolean false "Only active rates"
// @Success 200 {array} models.TaxRate
// @Failure 401 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/tax-rates [get]
// @Security BearerAuth
func (h *TaxRateHandler) ListTaxRates(c *gin.Context) {
	tenantID, ok := taxRateTenant(c)
	if !ok {
		return
	}
	rates, err := h.service.List(c.Request.Context(), tenantID, c.Query("active_only") == "true")
	if err != nil {
		apierror.Respond(c, err, "Failed to list tax rates")
		return
	}
	c.JSON(http.StatusOK, rates)
}

// GetTaxRate godoc
// @Summary Get a tax rate
// @Description One of the tenant's tax rates or a global default. Another tenant's rate is 404.
// @Tags tax-rates
// @Produce json
// @Param id path string true "Tax rate ID"
// @Success 200 {object} models.TaxRate
// @Failure 400 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/v1/tax-rates/{id} [get]
// @Security BearerAuth
func (h *TaxRateHandler) GetTaxRate(c *gin.Context) {
	tenantID, ok := taxRateTenant(c)
	if !ok {
		return
	}
	id, ok := taxRatePathID(c)
	if !ok {
		return
	}
	rate, err := h.service.Get(c.Request.Context(), tenantID, id)
	if err != nil {
		apierror.Respond(c, err, "Failed to get tax rate")
		return
	}
	c.JSON(http.StatusOK, rate)
}

// CreateTaxRate godoc
// @Summary Create a tax rate
// @Description Adds a tax rate owned by the tenant. rate is a fraction between 0 and 1 (0.0825 = 8.25%).
// @Tags tax-rates
// @Accept json
// @Produce json
// @Param request body CreateTaxRateRequest true "Tax rate"
// @Success 201 {object} models.TaxRate
// @Failure 400 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Router /api/v1/tax-rates [post]
// @Security BearerAuth
func (h *TaxRateHandler) CreateTaxRate(c *gin.Context) {
	tenantID, ok := taxRateTenant(c)
	if !ok {
		return
	}
	var req CreateTaxRateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.BindError(c, err)
		return
	}
	in := services.TaxRateInput{
		Name:        &req.Name,
		Description: &req.Description,
		Rate:        req.Rate,
		Region:      &req.Region,
		IsCompound:  &req.IsCompound,
		IsActive:    req.IsActive,
	}
	if req.TaxType != "" {
		in.TaxType = &req.TaxType
	}
	rate, err := h.service.Create(c.Request.Context(), tenantID, in)
	if err != nil {
		apierror.Respond(c, err, "Failed to create tax rate")
		return
	}
	c.JSON(http.StatusCreated, rate)
}

// UpdateTaxRate godoc
// @Summary Update a tax rate
// @Description Changes the given fields of one of the tenant's own tax rates. Global defaults are read-only (403 tax_rate_read_only); another tenant's rate is 404.
// @Tags tax-rates
// @Accept json
// @Produce json
// @Param id path string true "Tax rate ID"
// @Param request body UpdateTaxRateRequest true "Fields to change"
// @Success 200 {object} models.TaxRate
// @Failure 400 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Router /api/v1/tax-rates/{id} [put]
// @Security BearerAuth
func (h *TaxRateHandler) UpdateTaxRate(c *gin.Context) {
	tenantID, ok := taxRateTenant(c)
	if !ok {
		return
	}
	id, ok := taxRatePathID(c)
	if !ok {
		return
	}
	var req UpdateTaxRateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.BindError(c, err)
		return
	}
	rate, err := h.service.Update(c.Request.Context(), tenantID, id, services.TaxRateInput{
		Name:        req.Name,
		Description: req.Description,
		Rate:        req.Rate,
		TaxType:     req.TaxType,
		Region:      req.Region,
		IsCompound:  req.IsCompound,
		IsActive:    req.IsActive,
	})
	if err != nil {
		apierror.Respond(c, err, "Failed to update tax rate")
		return
	}
	c.JSON(http.StatusOK, rate)
}

// DeleteTaxRate godoc
// @Summary Delete a tax rate
// @Description Deletes one of the tenant's own tax rates. Global defaults are read-only (403); a rate used by invoices is 409 tax_rate_in_use (deactivate it instead).
// @Tags tax-rates
// @Param id path string true "Tax rate ID"
// @Success 204
// @Failure 400 {object} models.ErrorResponse
// @Failure 403 {object} models.ErrorResponse
// @Failure 404 {object} models.ErrorResponse
// @Failure 409 {object} models.ErrorResponse
// @Router /api/v1/tax-rates/{id} [delete]
// @Security BearerAuth
func (h *TaxRateHandler) DeleteTaxRate(c *gin.Context) {
	tenantID, ok := taxRateTenant(c)
	if !ok {
		return
	}
	id, ok := taxRatePathID(c)
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), tenantID, id); err != nil {
		apierror.Respond(c, err, "Failed to delete tax rate")
		return
	}
	c.Status(http.StatusNoContent)
}
