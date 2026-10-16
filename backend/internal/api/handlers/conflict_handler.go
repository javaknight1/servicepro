package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services/conflict"
)

// ConflictHandler handles conflict detection endpoints
type ConflictHandler struct {
	detector *conflict.ConflictDetector
}

// NewConflictHandler creates a new conflict handler
func NewConflictHandler(detector *conflict.ConflictDetector) *ConflictHandler {
	return &ConflictHandler{
		detector: detector,
	}
}

// CheckConflicts handles POST /api/v1/conflicts/check
// @Summary      Check for scheduling conflicts
// @Description  Checks for scheduling conflicts given a proposed schedule
// @Tags         conflicts
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body conflict.ConflictCheckRequest true "Conflict check request"
// @Success      200  {object}  conflict.ConflictCheckResponse
// @Failure      400  {object}  models.ErrorResponse
// @Failure      401  {object}  models.ErrorResponse
// @Failure      500  {object}  models.ErrorResponse
// @Router       /api/v1/conflicts/check [post]
func (h *ConflictHandler) CheckConflicts(c *gin.Context) {
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		apierror.WriteResponse(c, http.StatusUnauthorized, models.ErrorResponse{
			Error:   "unauthorized",
			Message: "Tenant context required",
		})
		return
	}

	var req conflict.ConflictCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.BindError(c, err)
		return
	}

	req.TenantID = tenantID

	response, err := h.detector.CheckConflicts(c.Request.Context(), &req)
	if err != nil {
		apierror.Internal(c, err, "Failed to check conflicts")
		return
	}

	c.JSON(http.StatusOK, response)
}
