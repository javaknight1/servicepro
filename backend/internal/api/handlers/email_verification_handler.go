package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// EmailVerificationHandler handles email verification endpoints
type EmailVerificationHandler struct {
	verificationService services.EmailVerificationServiceInterface
}

// NewEmailVerificationHandler creates a new email verification handler
func NewEmailVerificationHandler(verificationService services.EmailVerificationServiceInterface) *EmailVerificationHandler {
	return &EmailVerificationHandler{
		verificationService: verificationService,
	}
}

// VerifyEmail handles GET /api/v1/auth/verify
// @Summary Verify email address
// @Description Verify a user's email address using a verification token
// @Tags auth
// @Produce json
// @Param token query string true "Verification token"
// @Success 200 {object} models.VerifyEmailResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 401 {object} models.ErrorResponse "Invalid or expired token"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/auth/verify [get]
func (h *EmailVerificationHandler) VerifyEmail(c *gin.Context) {
	// Get token from query parameter
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Verification token is required",
		})
		return
	}

	req := models.VerifyEmailRequest{Token: token}

	// Verify email
	err := h.verificationService.VerifyEmail(req.Token)
	if err != nil {
		// Handle different error types
		if errors.Is(err, services.ErrInvalidVerificationToken) {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{
				Error:   "invalid_token",
				Message: "Invalid or expired verification token",
			})
			return
		}

		// Internal server error
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "internal_error",
			Message: "An error occurred while verifying your email",
		})
		return
	}

	// Return success response
	c.JSON(http.StatusOK, models.VerifyEmailResponse{
		Message: "Your email has been successfully verified. You can now access all features.",
	})
}

// ResendVerificationEmail handles POST /api/v1/auth/resend-verification
// @Summary Resend verification email
// @Description Resend the verification email to a user
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.ResendVerificationRequest true "Email address"
// @Success 200 {object} models.ResendVerificationResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request or already verified"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/auth/resend-verification [post]
func (h *EmailVerificationHandler) ResendVerificationEmail(c *gin.Context) {
	var req models.ResendVerificationRequest

	// Bind and validate request
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Valid email address is required",
		})
		return
	}

	// Resend verification email
	err := h.verificationService.ResendVerificationEmail(req.Email)
	if err != nil {
		// Handle different error types
		if errors.Is(err, services.ErrEmailAlreadyVerified) {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error:   "already_verified",
				Message: "Email address is already verified",
			})
			return
		}

		// For security, don't reveal if email exists or internal errors
		// Always return success
		c.JSON(http.StatusOK, models.ResendVerificationResponse{
			Message: "If an account with that email exists and is unverified, a verification email has been sent.",
		})
		return
	}

	// Return success response
	c.JSON(http.StatusOK, models.ResendVerificationResponse{
		Message: "If an account with that email exists and is unverified, a verification email has been sent.",
	})
}

// HandleSESBounce handles AWS SES bounce notifications (webhook)
// POST /api/v1/webhooks/ses-bounce
// @Summary Handle AWS SES bounce notifications
// @Description Processes bounce notifications from AWS SES
// @Tags webhooks
// @Accept json
// @Produce json
// @Param notification body models.SESBounceNotification true "SES Bounce Notification"
// @Success 200 {string} string "OK"
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/webhooks/ses-bounce [post]
func (h *EmailVerificationHandler) HandleSESBounce(c *gin.Context) {
	var notification models.SESBounceNotification

	// Parse the notification
	if err := c.ShouldBindJSON(&notification); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid bounce notification format",
		})
		return
	}

	// Process each bounced recipient
	for _, recipient := range notification.Bounce.BouncedRecipients {
		// Handle the bounce
		if err := h.verificationService.HandleBounce(recipient.EmailAddress); err != nil {
			// Log error but don't fail the request
			// AWS SES expects a 200 OK response
			c.String(http.StatusOK, "OK")
			return
		}
	}

	c.String(http.StatusOK, "OK")
}

// HandleSESNotification handles general AWS SES notifications
// This handles the initial SNS subscription confirmation
// POST /api/v1/webhooks/ses-notification
func (h *EmailVerificationHandler) HandleSESNotification(c *gin.Context) {
	// Read raw body
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Failed to read notification body",
		})
		return
	}

	// Parse as generic map to check type
	var notification map[string]interface{}
	if err := json.Unmarshal(body, &notification); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid JSON format",
		})
		return
	}

	// Check notification type
	notificationType, ok := notification["Type"].(string)
	if !ok {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "invalid_request",
			Message: "Missing notification type",
		})
		return
	}

	// Handle subscription confirmation
	if notificationType == "SubscriptionConfirmation" {
		// In production, you should verify the signature and confirm the subscription
		// by making a GET request to the SubscribeURL
		subscribeURL, ok := notification["SubscribeURL"].(string)
		if ok {
			// Log the subscription URL for manual confirmation or auto-confirm
			// For security, validate the request came from AWS SNS before confirming
			c.String(http.StatusOK, "Subscription confirmation URL: "+subscribeURL)
			return
		}
	}

	// Handle bounce notifications
	if notificationType == "Notification" {
		var bounceNotification models.SESBounceNotification
		if err := json.Unmarshal(body, &bounceNotification); err == nil {
			// Process bounce
			h.HandleSESBounce(c)
			return
		}
	}

	c.String(http.StatusOK, "OK")
}
