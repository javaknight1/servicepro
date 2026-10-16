package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// MembershipHandler handles membership-related HTTP requests
type MembershipHandler struct {
	membershipService *services.MembershipService
}

// NewMembershipHandler creates a new membership handler
func NewMembershipHandler(membershipService *services.MembershipService) *MembershipHandler {
	return &MembershipHandler{
		membershipService: membershipService,
	}
}

// GetAllTiers retrieves all available membership tiers
// @Summary		Get membership tiers
// @Description	Retrieves all available membership tiers and their features
// @Tags			Membership
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Success		200	{array}		models.MembershipTier
// @Failure		500	{object}	models.ErrorResponse
// @Router			/membership-tiers [get]
func (h *MembershipHandler) GetAllTiers(c *gin.Context) {
	tiers, err := h.membershipService.GetAllTiers(c.Request.Context())
	if err != nil {
		apierror.Internal(c, err, "Failed to fetch membership tiers")
		return
	}

	c.JSON(http.StatusOK, tiers)
}

// GetTenantMembership retrieves the current membership for a tenant
// @Summary		Get tenant membership
// @Description	Retrieves the current membership subscription for a tenant
// @Tags			Membership
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Tenant ID"
// @Success		200	{object}	models.TenantSubscriptionResponse
// @Failure		400	{object}	models.ErrorResponse
// @Failure		404	{object}	models.ErrorResponse
// @Failure		500	{object}	models.ErrorResponse
// @Router			/tenants/{id}/membership [get]
func (h *MembershipHandler) GetTenantMembership(c *gin.Context) {
	idStr := c.Param("id")
	tenantID, err := uuid.Parse(idStr)
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid tenant ID format",
		})
		return
	}

	membership, err := h.membershipService.GetTenantMembership(c.Request.Context(), tenantID)
	if err != nil {
		if errors.Is(err, services.ErrNoActiveSubscription) {
			apierror.WriteResponse(c, http.StatusNotFound, models.ErrorResponse{
				Error:   "no_subscription",
				Message: "No active subscription found for this tenant",
			})
			return
		}
		apierror.Internal(c, err, "Failed to fetch membership")
		return
	}

	c.JSON(http.StatusOK, membership)
}

// UpdateTenantMembership updates a tenant's membership tier
// @Summary		Update tenant membership
// @Description	Updates the membership tier for a tenant (upgrade/downgrade)
// @Tags			Membership
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id		path		string							true	"Tenant ID"
// @Param			request	body		models.UpdateMembershipRequest	true	"Membership update request"
// @Success		200		{object}	models.TenantSubscriptionResponse
// @Failure		400		{object}	models.ErrorResponse
// @Failure		401		{object}	models.ErrorResponse
// @Failure		402		{object}	models.ErrorResponse
// @Failure		404		{object}	models.ErrorResponse
// @Failure		409		{object}	models.ErrorResponse	"Already on this tier"
// @Failure		500		{object}	models.ErrorResponse
// @Router			/tenants/{id}/membership [put]
func (h *MembershipHandler) UpdateTenantMembership(c *gin.Context) {
	idStr := c.Param("id")
	tenantID, err := uuid.Parse(idStr)
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid tenant ID format",
		})
		return
	}

	var req models.UpdateMembershipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.BindError(c, err)
		return
	}

	userID, err := middleware.GetUserID(c)
	if err != nil {
		apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	membership, err := h.membershipService.UpdateTenantMembership(c.Request.Context(), tenantID, &req, userID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrMembershipTierNotFound):
			apierror.WriteResponse(c, http.StatusNotFound, models.ErrorResponse{
				Error:   "tier_not_found",
				Message: "Membership tier not found",
			})
		case errors.Is(err, services.ErrSameTier):
			apierror.WriteResponse(c, http.StatusConflict, models.ErrorResponse{
				Error:   "same_tier",
				Message: "Tenant is already on this membership tier",
			})
		case errors.Is(err, services.ErrPaymentMethodRequired):
			apierror.WriteResponse(c, http.StatusPaymentRequired, models.ErrorResponse{
				Error:   "payment_method_required",
				Message: "A payment method is required to upgrade to a paid tier. Please add a payment method first.",
			})
		default:
			apierror.Internal(c, err, "Failed to update membership")
		}
		return
	}

	c.JSON(http.StatusOK, membership)
}

// PreviewSubscriptionChange previews what will happen when changing subscriptions
// @Summary		Preview subscription change
// @Description	Previews the prorated charges/credits for a subscription change
// @Tags			Membership
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id		path		string									true	"Tenant ID"
// @Param			request	body		models.PreviewSubscriptionChangeRequest	true	"Preview request"
// @Success		200		{object}	models.PreviewSubscriptionChangeResponse
// @Failure		400		{object}	models.ErrorResponse
// @Failure		404		{object}	models.ErrorResponse
// @Failure		500		{object}	models.ErrorResponse
// @Router			/tenants/{id}/membership/preview [post]
func (h *MembershipHandler) PreviewSubscriptionChange(c *gin.Context) {
	idStr := c.Param("id")
	tenantID, err := uuid.Parse(idStr)
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid tenant ID format",
		})
		return
	}

	var req models.PreviewSubscriptionChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.BindError(c, err)
		return
	}

	preview, err := h.membershipService.PreviewSubscriptionChange(c.Request.Context(), tenantID, &req)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrMembershipTierNotFound):
			apierror.WriteResponse(c, http.StatusNotFound, models.ErrorResponse{
				Error:   "tier_not_found",
				Message: "Membership tier not found",
			})
		default:
			apierror.Internal(c, err, "Failed to preview subscription change")
		}
		return
	}

	c.JSON(http.StatusOK, preview)
}

// GetSubscriptionHistory retrieves the subscription history for a tenant
// @Summary		Get subscription history
// @Description	Retrieves the subscription change history for a tenant
// @Tags			Membership
// @Accept			json
// @Produce		json
// @Security		BearerAuth
// @Param			id	path		string	true	"Tenant ID"
// @Success		200	{object}	map[string]interface{}
// @Failure		400	{object}	models.ErrorResponse
// @Failure		500	{object}	models.ErrorResponse
// @Router			/tenants/{id}/membership/history [get]
func (h *MembershipHandler) GetSubscriptionHistory(c *gin.Context) {
	idStr := c.Param("id")
	tenantID, err := uuid.Parse(idStr)
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid tenant ID format",
		})
		return
	}

	history, err := h.membershipService.GetSubscriptionHistory(c.Request.Context(), tenantID)
	if err != nil {
		apierror.Internal(c, err, "Failed to fetch subscription history")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"history": history,
	})
}
