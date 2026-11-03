package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// SiteHandler serves a customer's sites (service locations, T186) under
// /customers/{id}/sites. The tenant comes from the request context.
type SiteHandler struct {
	sites services.SiteServiceInterface
}

// NewSiteHandler creates a site handler.
func NewSiteHandler(sites services.SiteServiceInterface) *SiteHandler {
	return &SiteHandler{sites: sites}
}

// parseUUIDParam parses path parameter name, answering 400 when it isn't a UUID.
func parseUUIDParam(c *gin.Context, name, label string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		apierror.Write(c, http.StatusBadRequest, "invalid_id", "Invalid "+label+" ID")
		return uuid.Nil, false
	}
	return id, true
}

// ListSites handles GET /api/v1/customers/:id/sites
// @Summary List a customer's sites
// @Description The customer's service locations, by name. Inactive sites are left out unless include_inactive=true.
// @Tags sites
// @Produce json
// @Security BearerAuth
// @Param id path string true "Customer ID"
// @Param include_inactive query bool false "Include inactive sites"
// @Success 200 {object} models.SiteListResponse
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 404 {object} models.ErrorResponse "Customer not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/customers/{id}/sites [get]
func (h *SiteHandler) ListSites(c *gin.Context) {
	customerID, ok := parseUUIDParam(c, "id", "customer")
	if !ok {
		return
	}
	sites, err := h.sites.List(c.Request.Context(), customerID, c.Query("include_inactive") == "true")
	if err != nil {
		apierror.Respond(c, err, "Failed to list sites")
		return
	}
	resp := models.SiteListResponse{Sites: make([]models.SiteResponse, len(sites)), Total: len(sites)}
	for i := range sites {
		resp.Sites[i] = sites[i].ToResponse()
	}
	c.JSON(http.StatusOK, resp)
}

// GetSite handles GET /api/v1/customers/:id/sites/:siteId
// @Summary Get a customer's site
// @Description One of the customer's service locations.
// @Tags sites
// @Produce json
// @Security BearerAuth
// @Param id path string true "Customer ID"
// @Param siteId path string true "Site ID"
// @Success 200 {object} models.SiteResponse
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 404 {object} models.ErrorResponse "Customer or site not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/customers/{id}/sites/{siteId} [get]
func (h *SiteHandler) GetSite(c *gin.Context) {
	customerID, ok := parseUUIDParam(c, "id", "customer")
	if !ok {
		return
	}
	siteID, ok := parseUUIDParam(c, "siteId", "site")
	if !ok {
		return
	}
	site, err := h.sites.Get(c.Request.Context(), customerID, siteID)
	if err != nil {
		apierror.Respond(c, err, "Failed to get site")
		return
	}
	c.JSON(http.StatusOK, site.ToResponse())
}

// CreateSite handles POST /api/v1/customers/:id/sites
// @Summary Add a site to a customer
// @Description Creates a service location (address, access notes, site contact) for the customer.
// @Tags sites
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Customer ID"
// @Param request body models.CreateSiteRequest true "Site"
// @Success 201 {object} models.SiteResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 404 {object} models.ErrorResponse "Customer not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/customers/{id}/sites [post]
func (h *SiteHandler) CreateSite(c *gin.Context) {
	customerID, ok := parseUUIDParam(c, "id", "customer")
	if !ok {
		return
	}
	var req models.CreateSiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.BindError(c, err)
		return
	}
	site, err := h.sites.Create(c.Request.Context(), customerID, &req)
	if err != nil {
		apierror.Respond(c, err, "Failed to create site")
		return
	}
	c.JSON(http.StatusCreated, site.ToResponse())
}

// UpdateSite handles PUT /api/v1/customers/:id/sites/:siteId
// @Summary Update a customer's site
// @Description Changes the fields present in the body; is_active=false retires the site for new jobs.
// @Tags sites
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Customer ID"
// @Param siteId path string true "Site ID"
// @Param request body models.UpdateSiteRequest true "Fields to change"
// @Success 200 {object} models.SiteResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 404 {object} models.ErrorResponse "Customer or site not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/customers/{id}/sites/{siteId} [put]
func (h *SiteHandler) UpdateSite(c *gin.Context) {
	customerID, ok := parseUUIDParam(c, "id", "customer")
	if !ok {
		return
	}
	siteID, ok := parseUUIDParam(c, "siteId", "site")
	if !ok {
		return
	}
	var req models.UpdateSiteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.BindError(c, err)
		return
	}
	site, err := h.sites.Update(c.Request.Context(), customerID, siteID, &req)
	if err != nil {
		apierror.Respond(c, err, "Failed to update site")
		return
	}
	c.JSON(http.StatusOK, site.ToResponse())
}

// DeleteSite handles DELETE /api/v1/customers/:id/sites/:siteId
// @Summary Delete a customer's site
// @Description Soft-deletes the site. Existing jobs keep their reference; new jobs can't use it.
// @Tags sites
// @Security BearerAuth
// @Param id path string true "Customer ID"
// @Param siteId path string true "Site ID"
// @Success 204 "Deleted"
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 404 {object} models.ErrorResponse "Customer or site not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/customers/{id}/sites/{siteId} [delete]
func (h *SiteHandler) DeleteSite(c *gin.Context) {
	customerID, ok := parseUUIDParam(c, "id", "customer")
	if !ok {
		return
	}
	siteID, ok := parseUUIDParam(c, "siteId", "site")
	if !ok {
		return
	}
	if err := h.sites.Delete(c.Request.Context(), customerID, siteID); err != nil {
		apierror.Respond(c, err, "Failed to delete site")
		return
	}
	c.Status(http.StatusNoContent)
}
