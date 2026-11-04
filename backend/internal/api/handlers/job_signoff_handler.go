package handlers

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// maxSignoffBodyBytes caps POST /jobs/{id}/signoff: the 512 KB PNG as
// base64 plus the signer's details.
const maxSignoffBodyBytes = 800 << 10

// JobSignoffHandler serves a job's customer sign-off and service report
// (T176) under /jobs/{id}/signoff. The tenant and the technician row filter
// come from the request context, so a job the caller can't open is a 404.
type JobSignoffHandler struct {
	signoffs services.JobSignoffServiceInterface
}

// NewJobSignoffHandler creates a job sign-off handler.
func NewJobSignoffHandler(signoffs services.JobSignoffServiceInterface) *JobSignoffHandler {
	return &JobSignoffHandler{signoffs: signoffs}
}

// GetSignoff handles GET /api/v1/jobs/:id/signoff
// @Summary Get a job's customer sign-off
// @Description The sign-off (null when the job hasn't been signed) with a presigned link to the signature, whether and to whom the service report was emailed, and suggested_recipients: who the report goes to (the customer's email and the site contact unless the customer opted out of email, and the signer's email).
// @Tags jobs
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job ID"
// @Success 200 {object} models.JobSignoffEnvelope
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 404 {object} models.ErrorResponse "Job not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs/{id}/signoff [get]
func (h *JobSignoffHandler) GetSignoff(c *gin.Context) {
	jobID, ok := parseUUIDParam(c, "id", "job")
	if !ok {
		return
	}
	env, err := h.signoffs.Get(c.Request.Context(), jobID)
	if err != nil {
		apierror.Respond(c, err, "Failed to load the sign-off")
		return
	}
	c.JSON(http.StatusOK, env)
}

// CaptureSignoff handles POST /api/v1/jobs/:id/signoff
// @Summary Record the customer's signature on a job
// @Description The customer signs on the technician's phone: signer name (optional title and email) and the signature as a PNG data URL. The type is sniffed from the bytes (PNG only), at most 512 KB and 100 x 40 to 3000 x 3000 pixels (checked before decoding); a canvas without ink is refused. The visit must be started or done (409 job_not_started). Signing again replaces the sign-off until the service report has been emailed, then it is locked (409 signoff_locked). The time, IP and device are recorded. Technicians can sign off only jobs assigned to them; other jobs are 404.
// @Tags jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job ID"
// @Param request body models.CaptureJobSignoffRequest true "Signer and signature"
// @Success 201 {object} models.JobSignoffResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request, unreadable signature (invalid_signature), wrong size (signature_dimensions), empty canvas (signature_empty)"
// @Failure 404 {object} models.ErrorResponse "Job not found"
// @Failure 409 {object} models.ErrorResponse "Visit not started (job_not_started) or report already emailed (signoff_locked)"
// @Failure 413 {object} models.ErrorResponse "Signature over 512 KB (signature_too_large)"
// @Failure 415 {object} models.ErrorResponse "Not a PNG (unsupported_signature_type)"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs/{id}/signoff [post]
func (h *JobSignoffHandler) CaptureSignoff(c *gin.Context) {
	jobID, ok := parseUUIDParam(c, "id", "job")
	if !ok {
		return
	}
	userID, ok := callerID(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSignoffBodyBytes)
	var req models.CaptureJobSignoffRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			apierror.Respond(c, services.ErrSignatureTooLarge, "")
			return
		}
		apierror.BindError(c, err)
		return
	}
	so, err := h.signoffs.Capture(c.Request.Context(), jobID, userID, &req, middleware.ClientIP(c), c.Request.UserAgent())
	if err != nil {
		apierror.Respond(c, err, "Failed to save the sign-off")
		return
	}
	c.JSON(http.StatusCreated, so)
}

func (h *JobSignoffHandler) redirect(c *gin.Context, link func(*gin.Context) (string, error), cache string) {
	url, err := link(c)
	if err != nil {
		apierror.Respond(c, err, "Failed to load the file")
		return
	}
	c.Header("Cache-Control", cache)
	c.Redirect(http.StatusFound, url)
}

// GetSignature handles GET /api/v1/jobs/:id/signoff/signature
// @Summary Download the customer's signature
// @Description Redirects to a presigned link (about an hour) to the signature PNG.
// @Tags jobs
// @Security BearerAuth
// @Param id path string true "Job ID"
// @Success 302
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 404 {object} models.ErrorResponse "Job or sign-off not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs/{id}/signoff/signature [get]
func (h *JobSignoffHandler) GetSignature(c *gin.Context) {
	jobID, ok := parseUUIDParam(c, "id", "job")
	if !ok {
		return
	}
	h.redirect(c, func(c *gin.Context) (string, error) {
		return h.signoffs.SignatureLink(c.Request.Context(), jobID)
	}, "private, max-age=600")
}

// GetReport handles GET /api/v1/jobs/:id/signoff/report
// @Summary Download the service report PDF
// @Description Redirects to a presigned link (about an hour) to the service report: once emailed, the copy the customer received; before that, one built from the job as it is now (work done, notes, materials without prices, time on site, photos, signature).
// @Tags jobs
// @Security BearerAuth
// @Param id path string true "Job ID"
// @Success 302
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 404 {object} models.ErrorResponse "Job or sign-off not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs/{id}/signoff/report [get]
func (h *JobSignoffHandler) GetReport(c *gin.Context) {
	jobID, ok := parseUUIDParam(c, "id", "job")
	if !ok {
		return
	}
	// A draft changes as photos and notes are added: don't cache it long.
	h.redirect(c, func(c *gin.Context) (string, error) {
		return h.signoffs.ReportLink(c.Request.Context(), jobID)
	}, "private, no-store")
}

// EmailReport handles POST /api/v1/jobs/:id/signoff/email
// @Summary Email the service report to the customer
// @Description Emails the service report PDF to suggested_recipients, from the business. Sends at most once: a repeated request after a successful send returns sent=false and sends nothing; {"resend": true} sends the stored report again. A concurrent send is 409 email_in_progress; no address (or the customer opted out and the signer gave none) is 409 no_recipients.
// @Tags jobs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Job ID"
// @Param request body models.EmailJobReportRequest false "Resend"
// @Success 200 {object} models.EmailJobReportResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 404 {object} models.ErrorResponse "Job or sign-off not found"
// @Failure 409 {object} models.ErrorResponse "No recipients (no_recipients) or already sending (email_in_progress)"
// @Failure 502 {object} models.ErrorResponse "The email provider refused it (email_failed)"
// @Failure 503 {object} models.ErrorResponse "Email not configured (email_unavailable)"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/jobs/{id}/signoff/email [post]
func (h *JobSignoffHandler) EmailReport(c *gin.Context) {
	jobID, ok := parseUUIDParam(c, "id", "job")
	if !ok {
		return
	}
	var req models.EmailJobReportRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		apierror.BindError(c, err)
		return
	}
	res, err := h.signoffs.EmailReport(c.Request.Context(), jobID, req.Resend)
	if err != nil {
		apierror.Respond(c, err, "Failed to email the service report")
		return
	}
	c.JSON(http.StatusOK, res)
}
