package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// PaymentsConnectHandler lets a tenant connect its own Stripe account and see
// whether it can take invoice payments (Stripe Connect, T169).
type PaymentsConnectHandler struct {
	svc *services.PaymentsConnectService
}

// NewPaymentsConnectHandler creates the handler.
func NewPaymentsConnectHandler(svc *services.PaymentsConnectService) *PaymentsConnectHandler {
	return &PaymentsConnectHandler{svc: svc}
}

// tenantID is the tenant RequireTenantFromPath verified the caller belongs to.
func (h *PaymentsConnectHandler) tenantID(c *gin.Context) (uuid.UUID, bool) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		apierror.Write(c, http.StatusUnauthorized, "unauthorized", "User not authenticated")
		return uuid.Nil, false
	}
	return tenantID, true
}

// Connect starts or continues Stripe onboarding.
// @Summary		Connect Stripe payouts
// @Description	Creates the organization's own Stripe account on first use (Accounts v2, direct charges; Stripe collects its fees from the business) and returns a single-use Stripe-hosted onboarding link. Call it again to continue onboarding. Requires membership.billing (owners and admins).
// @Tags			Payments
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Tenant ID"
// @Success		200	{object}	models.PaymentsConnectResponse
// @Failure		401	{object}	models.ErrorResponse
// @Failure		403	{object}	models.ErrorResponse
// @Failure		502	{object}	models.ErrorResponse
// @Failure		503	{object}	models.ErrorResponse
// @Router			/tenants/{id}/payments/connect [post]
func (h *PaymentsConnectHandler) Connect(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		return
	}
	url, err := h.svc.StartOnboarding(c.Request.Context(), tenantID)
	if err != nil {
		apierror.Respond(c, err, "Failed to start Stripe onboarding")
		return
	}
	c.JSON(http.StatusOK, models.PaymentsConnectResponse{URL: url})
}

// Status reports the organization's payment readiness.
// @Summary		Get payments status
// @Description	Whether the organization's Stripe account exists, which payment methods are active, whether Stripe needs more information, and ServicePro's fees. While onboarding is unfinished the status is re-read from Stripe. Requires membership.billing.
// @Tags			Payments
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Tenant ID"
// @Success		200	{object}	models.PaymentsStatusResponse
// @Failure		401	{object}	models.ErrorResponse
// @Failure		403	{object}	models.ErrorResponse
// @Failure		500	{object}	models.ErrorResponse
// @Router			/tenants/{id}/payments/status [get]
func (h *PaymentsConnectHandler) Status(c *gin.Context) {
	tenantID, ok := h.tenantID(c)
	if !ok {
		return
	}
	status, err := h.svc.Status(c.Request.Context(), tenantID)
	if err != nil {
		apierror.Respond(c, err, "Failed to load payments status")
		return
	}
	c.JSON(http.StatusOK, status)
}
