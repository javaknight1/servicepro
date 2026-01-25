package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

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
// GET /api/v1/membership-tiers
func (h *MembershipHandler) GetAllTiers(c *gin.Context) {
	tiers, err := h.membershipService.GetAllTiers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "fetch_failed",
			Message: "Failed to fetch membership tiers",
		})
		return
	}

	c.JSON(http.StatusOK, tiers)
}

// GetTenantMembership retrieves the current membership for a tenant
// GET /api/v1/tenants/:id/membership
func (h *MembershipHandler) GetTenantMembership(c *gin.Context) {
	idStr := c.Param("id")
	tenantID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid tenant ID format",
		})
		return
	}

	membership, err := h.membershipService.GetTenantMembership(c.Request.Context(), tenantID)
	if err != nil {
		if errors.Is(err, services.ErrNoActiveSubscription) {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "no_subscription",
				Message: "No active subscription found for this tenant",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "fetch_failed",
			Message: "Failed to fetch membership",
		})
		return
	}

	c.JSON(http.StatusOK, membership)
}

// UpdateTenantMembership updates a tenant's membership tier
// PUT /api/v1/tenants/:id/membership
func (h *MembershipHandler) UpdateTenantMembership(c *gin.Context) {
	idStr := c.Param("id")
	tenantID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid tenant ID format",
		})
		return
	}

	var req models.UpdateMembershipRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	userID, err := middleware.GetUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	membership, err := h.membershipService.UpdateTenantMembership(c.Request.Context(), tenantID, &req, userID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrMembershipTierNotFound):
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "tier_not_found",
				Message: "Membership tier not found",
			})
		case errors.Is(err, services.ErrSameTier):
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "same_tier",
				Message: "Tenant is already on this membership tier",
			})
		case errors.Is(err, services.ErrPaymentMethodRequired):
			c.JSON(http.StatusPaymentRequired, models.ErrorResponse{
				Error:   "payment_method_required",
				Message: "A payment method is required to upgrade to a paid tier. Please add a payment method first.",
			})
		default:
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "update_failed",
				Message: "Failed to update membership",
			})
		}
		return
	}

	c.JSON(http.StatusOK, membership)
}

// PreviewSubscriptionChange previews what will happen when changing subscriptions
// POST /api/v1/tenants/:id/membership/preview
func (h *MembershipHandler) PreviewSubscriptionChange(c *gin.Context) {
	idStr := c.Param("id")
	tenantID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid tenant ID format",
		})
		return
	}

	var req models.PreviewSubscriptionChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	preview, err := h.membershipService.PreviewSubscriptionChange(c.Request.Context(), tenantID, &req)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrMembershipTierNotFound):
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error:   "tier_not_found",
				Message: "Membership tier not found",
			})
		default:
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error:   "preview_failed",
				Message: "Failed to preview subscription change",
			})
		}
		return
	}

	c.JSON(http.StatusOK, preview)
}

// GetSubscriptionHistory retrieves the subscription history for a tenant
// GET /api/v1/tenants/:id/membership/history
func (h *MembershipHandler) GetSubscriptionHistory(c *gin.Context) {
	idStr := c.Param("id")
	tenantID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid tenant ID format",
		})
		return
	}

	history, err := h.membershipService.GetSubscriptionHistory(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "fetch_failed",
			Message: "Failed to fetch subscription history",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"history": history,
	})
}
