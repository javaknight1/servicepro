package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// JobApprovalHandler records the customer's approver OK-ing extra work
// (T187, not-to-exceed rule).
type JobApprovalHandler struct {
	approvals services.JobApprovalServiceInterface
}

// NewJobApprovalHandler creates a job approval handler.
func NewJobApprovalHandler(approvals services.JobApprovalServiceInterface) *JobApprovalHandler {
	return &JobApprovalHandler{approvals: approvals}
}

// RecordApproval handles POST /api/v1/jobs/:id/approval
// @Summary Record the approver's OK for a job
// @Description Office staff record that the customer's approver OK'd this work (V1: no customer-facing approval flow). Extra work estimated above the customer's not-to-exceed limit can only be scheduled once an approval for at least its estimated cost exists. Approver defaults to the customer's billing approver, amount to the job's estimated cost, approved_at to now.
// @Tags jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job ID"
// @Param request body models.RecordJobApprovalRequest true "Approval"
// @Success 201 {object} models.JobApproval
// @Failure 400 {object} models.ErrorResponse "Invalid request, or no approver email"
// @Failure 401 {object} models.ErrorResponse "Unauthorized"
// @Failure 404 {object} models.ErrorResponse "Job not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs/{id}/approval [post]
func (h *JobApprovalHandler) RecordApproval(c *gin.Context) {
	jobID, ok := parseUUIDParam(c, "id", "job")
	if !ok {
		return
	}
	userID, ok := c.Get("user_id")
	uid, isUUID := userID.(uuid.UUID)
	if !ok || !isUUID {
		apierror.Write(c, http.StatusUnauthorized, "unauthorized", "User not authenticated")
		return
	}
	var req models.RecordJobApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.BindError(c, err)
		return
	}
	approval, err := h.approvals.Record(c.Request.Context(), jobID, &req, uid)
	if err != nil {
		apierror.Respond(c, err, "Failed to record approval")
		return
	}
	c.JSON(http.StatusCreated, approval)
}

// ListApprovals handles GET /api/v1/jobs/:id/approvals
// @Summary List a job's approvals
// @Description The approver OKs recorded for the job, newest first.
// @Tags jobs
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job ID"
// @Success 200 {object} models.JobApprovalListResponse
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 404 {object} models.ErrorResponse "Job not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs/{id}/approvals [get]
func (h *JobApprovalHandler) ListApprovals(c *gin.Context) {
	jobID, ok := parseUUIDParam(c, "id", "job")
	if !ok {
		return
	}
	approvals, err := h.approvals.List(c.Request.Context(), jobID)
	if err != nil {
		apierror.Respond(c, err, "Failed to list approvals")
		return
	}
	c.JSON(http.StatusOK, models.JobApprovalListResponse{Approvals: approvals})
}
