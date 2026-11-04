package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// JobAttachmentHandler serves a job's photos and files (T174) under
// /jobs/{id}/attachments. The tenant and the technician row filter come from
// the request context, so a job the caller can't open is a 404 here too.
type JobAttachmentHandler struct {
	attachments services.JobAttachmentServiceInterface
}

// NewJobAttachmentHandler creates a job-attachment handler.
func NewJobAttachmentHandler(attachments services.JobAttachmentServiceInterface) *JobAttachmentHandler {
	return &JobAttachmentHandler{attachments: attachments}
}

func attachmentIDs(c *gin.Context) (jobID, attachmentID uuid.UUID, ok bool) {
	if jobID, ok = parseUUIDParam(c, "id", "job"); !ok {
		return
	}
	attachmentID, ok = parseUUIDParam(c, "attachmentId", "attachment")
	return
}

func callerID(c *gin.Context) (uuid.UUID, bool) {
	id, err := getUserIDFromContext(c)
	if err != nil {
		apierror.Write(c, http.StatusUnauthorized, "unauthorized", "User not authenticated")
		return uuid.Nil, false
	}
	return id, true
}

// UploadAttachment handles POST /api/v1/jobs/:id/attachments
// @Summary Attach a photo or file to a job
// @Description Multipart upload (field "file", optional "caption"), at most 15 MB. The type is sniffed from the bytes: JPEG, PNG and WebP photos are re-encoded as JPEG (EXIF stripped, orientation applied, longest side 2048 px) with a 400 px thumbnail; PDFs are stored as uploaded. Photos declaring more than 12000 px a side or 40 megapixels are refused before decoding. Technicians can attach only to jobs assigned to them; other jobs are 404. A job holds at most 100 attachments.
// @Tags jobs
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job ID"
// @Param file formData file true "Photo (JPEG, PNG, WebP) or PDF"
// @Param caption formData string false "Caption, up to 500 characters"
// @Success 201 {object} models.JobAttachmentResponse
// @Failure 400 {object} models.ErrorResponse "No file, unreadable photo (invalid_image), too many pixels (image_too_large), caption too long"
// @Failure 404 {object} models.ErrorResponse "Job not found"
// @Failure 409 {object} models.ErrorResponse "The job already has 100 attachments (attachment_limit)"
// @Failure 413 {object} models.ErrorResponse "Larger than 15 MB (file_too_large)"
// @Failure 415 {object} models.ErrorResponse "Not a JPEG, PNG, WebP or PDF (unsupported_file_type)"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs/{id}/attachments [post]
func (h *JobAttachmentHandler) UploadAttachment(c *gin.Context) {
	jobID, ok := parseUUIDParam(c, "id", "job")
	if !ok {
		return
	}
	userID, ok := callerID(c)
	if !ok {
		return
	}
	// Cap the body before the multipart form is parsed, so an oversized
	// upload is refused instead of being spooled to disk.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, services.MaxAttachmentBytes+64<<10)
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			apierror.Respond(c, services.ErrAttachmentTooLarge, "")
			return
		}
		apierror.Write(c, http.StatusBadRequest, "file_required", "Choose a photo or file to upload")
		return
	}
	defer file.Close()

	var caption *string
	if values, ok := c.Request.MultipartForm.Value["caption"]; ok && len(values) > 0 {
		caption = &values[0]
	}
	att, err := h.attachments.Upload(c.Request.Context(), jobID, userID, header.Filename, file, caption)
	if err != nil {
		apierror.Respond(c, err, "Failed to upload attachment")
		return
	}
	c.JSON(http.StatusCreated, att)
}

// ListAttachments handles GET /api/v1/jobs/:id/attachments
// @Summary List a job's photos and files
// @Description Oldest first. Each carries presigned links (url, thumb_url for photos) valid for about an hour, and API paths (file_path, thumb_path) that redirect to fresh ones.
// @Tags jobs
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job ID"
// @Success 200 {object} models.JobAttachmentListResponse
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 404 {object} models.ErrorResponse "Job not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs/{id}/attachments [get]
func (h *JobAttachmentHandler) ListAttachments(c *gin.Context) {
	jobID, ok := parseUUIDParam(c, "id", "job")
	if !ok {
		return
	}
	list, err := h.attachments.List(c.Request.Context(), jobID)
	if err != nil {
		apierror.Respond(c, err, "Failed to list attachments")
		return
	}
	c.JSON(http.StatusOK, models.JobAttachmentListResponse{Attachments: list, Total: len(list)})
}

func (h *JobAttachmentHandler) redirect(c *gin.Context, thumb bool) {
	jobID, attachmentID, ok := attachmentIDs(c)
	if !ok {
		return
	}
	url, err := h.attachments.Link(c.Request.Context(), jobID, attachmentID, thumb)
	if err != nil {
		apierror.Respond(c, err, "Failed to load attachment")
		return
	}
	// The link lives an hour; let the browser reuse the redirect for a while.
	c.Header("Cache-Control", "private, max-age=600")
	c.Redirect(http.StatusFound, url)
}

// GetAttachmentFile handles GET /api/v1/jobs/:id/attachments/:attachmentId/file
// @Summary Download a job attachment
// @Description Redirects to a presigned link (about an hour) to the stored photo or file.
// @Tags jobs
// @Security BearerAuth
// @Param id path string true "Job ID"
// @Param attachmentId path string true "Attachment ID"
// @Success 302
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 404 {object} models.ErrorResponse "Job or attachment not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs/{id}/attachments/{attachmentId}/file [get]
func (h *JobAttachmentHandler) GetAttachmentFile(c *gin.Context) { h.redirect(c, false) }

// GetAttachmentThumb handles GET /api/v1/jobs/:id/attachments/:attachmentId/thumb
// @Summary Download a job photo's thumbnail
// @Description Redirects to a presigned link (about an hour) to the 400 px thumbnail. Files (PDF) have none: 404.
// @Tags jobs
// @Security BearerAuth
// @Param id path string true "Job ID"
// @Param attachmentId path string true "Attachment ID"
// @Success 302
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 404 {object} models.ErrorResponse "Job, attachment or thumbnail not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs/{id}/attachments/{attachmentId}/thumb [get]
func (h *JobAttachmentHandler) GetAttachmentThumb(c *gin.Context) { h.redirect(c, true) }

// UpdateAttachment handles PATCH /api/v1/jobs/:id/attachments/:attachmentId
// @Summary Caption a job attachment
// @Description Sets the caption; an empty one clears it. Technicians can change only attachments they added (403 attachment_not_yours).
// @Tags jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job ID"
// @Param attachmentId path string true "Attachment ID"
// @Param request body models.UpdateJobAttachmentRequest true "Caption"
// @Success 200 {object} models.JobAttachmentResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 403 {object} models.ErrorResponse "Someone else's attachment (attachment_not_yours)"
// @Failure 404 {object} models.ErrorResponse "Job or attachment not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs/{id}/attachments/{attachmentId} [patch]
func (h *JobAttachmentHandler) UpdateAttachment(c *gin.Context) {
	jobID, attachmentID, ok := attachmentIDs(c)
	if !ok {
		return
	}
	userID, ok := callerID(c)
	if !ok {
		return
	}
	var req models.UpdateJobAttachmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.BindError(c, err)
		return
	}
	att, err := h.attachments.UpdateCaption(c.Request.Context(), jobID, attachmentID, userID, &req)
	if err != nil {
		apierror.Respond(c, err, "Failed to update attachment")
		return
	}
	c.JSON(http.StatusOK, att)
}

// DeleteAttachment handles DELETE /api/v1/jobs/:id/attachments/:attachmentId
// @Summary Delete a job attachment
// @Description Removes the attachment and its stored file. Technicians can delete only attachments they added (403 attachment_not_yours).
// @Tags jobs
// @Security BearerAuth
// @Param id path string true "Job ID"
// @Param attachmentId path string true "Attachment ID"
// @Success 204 "Deleted"
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 403 {object} models.ErrorResponse "Someone else's attachment (attachment_not_yours)"
// @Failure 404 {object} models.ErrorResponse "Job or attachment not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs/{id}/attachments/{attachmentId} [delete]
func (h *JobAttachmentHandler) DeleteAttachment(c *gin.Context) {
	jobID, attachmentID, ok := attachmentIDs(c)
	if !ok {
		return
	}
	userID, ok := callerID(c)
	if !ok {
		return
	}
	if err := h.attachments.Delete(c.Request.Context(), jobID, attachmentID, userID); err != nil {
		apierror.Respond(c, err, "Failed to delete attachment")
		return
	}
	c.Status(http.StatusNoContent)
}
