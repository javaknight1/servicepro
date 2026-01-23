package webhooks

import (
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/models"
	stripeservice "github.com/javaknight1/servicepro/backend/internal/services/stripe"
)

// =============================================================================
// Stripe Webhook Handler
// =============================================================================

// StripeWebhookHandler handles Stripe webhook requests
type StripeWebhookHandler struct {
	service *stripeservice.WebhookHandlerService
}

// NewStripeWebhookHandler creates a new Stripe webhook handler
func NewStripeWebhookHandler(service *stripeservice.WebhookHandlerService) *StripeWebhookHandler {
	return &StripeWebhookHandler{
		service: service,
	}
}

// =============================================================================
// HTTP Response Types
// =============================================================================

// WebhookSuccessResponse represents a successful webhook response
type WebhookSuccessResponse struct {
	Received  bool   `json:"received"`
	EventID   string `json:"event_id,omitempty"`
	EventType string `json:"event_type,omitempty"`
}

// WebhookErrorResponse represents an error webhook response
type WebhookErrorResponse struct {
	Received bool   `json:"received"`
	Error    string `json:"error"`
	Code     string `json:"code,omitempty"`
}

// WebhookStatsResponse represents webhook statistics response
type WebhookStatsResponse struct {
	TotalEvents     int64            `json:"total_events"`
	ProcessedEvents int64            `json:"processed_events"`
	FailedEvents    int64            `json:"failed_events"`
	PendingEvents   int64            `json:"pending_events"`
	RetryingEvents  int64            `json:"retrying_events"`
	EventsByType    map[string]int64 `json:"events_by_type"`
	AvgProcessingMs float64          `json:"avg_processing_ms"`
}

// =============================================================================
// Main Webhook Endpoint
// =============================================================================

// HandleWebhook handles incoming Stripe webhooks
// @Summary Handle Stripe webhook
// @Description Receives and processes Stripe webhook events
// @Tags webhooks
// @Accept json
// @Produce json
// @Param Stripe-Signature header string true "Stripe signature header"
// @Success 200 {object} WebhookSuccessResponse
// @Failure 400 {object} WebhookErrorResponse
// @Failure 401 {object} WebhookErrorResponse
// @Failure 500 {object} WebhookErrorResponse
// @Router /api/v1/webhooks/stripe [post]
func (h *StripeWebhookHandler) HandleWebhook(c *gin.Context) {
	// Read the raw body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logError("Failed to read request body", err)
		c.JSON(http.StatusBadRequest, WebhookErrorResponse{
			Received: false,
			Error:    "Failed to read request body",
			Code:     "INVALID_REQUEST",
		})
		return
	}
	defer c.Request.Body.Close()

	// Validate body is not empty
	if len(body) == 0 {
		h.logError("Empty request body", nil)
		c.JSON(http.StatusBadRequest, WebhookErrorResponse{
			Received: false,
			Error:    "Request body is empty",
			Code:     "EMPTY_BODY",
		})
		return
	}

	// Get Stripe signature
	signature := c.GetHeader("Stripe-Signature")
	if signature == "" {
		h.logError("Missing Stripe-Signature header", nil)
		c.JSON(http.StatusUnauthorized, WebhookErrorResponse{
			Received: false,
			Error:    "Missing Stripe-Signature header",
			Code:     "MISSING_SIGNATURE",
		})
		return
	}

	// Build headers map
	headers := make(map[string]string)
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			// Mask sensitive headers
			if strings.ToLower(key) == "stripe-signature" {
				headers[key] = "[REDACTED]"
			} else {
				headers[key] = values[0]
			}
		}
	}

	// Get client IP
	clientIP := c.ClientIP()
	if clientIP == "" {
		clientIP = c.Request.RemoteAddr
	}

	// Build input
	input := &stripeservice.WebhookInput{
		Payload:   body,
		Signature: signature,
		IPAddress: clientIP,
		UserAgent: c.GetHeader("User-Agent"),
		Headers:   headers,
	}

	// Process webhook
	result, err := h.service.ProcessWebhook(c.Request.Context(), input)
	if err != nil {
		h.handleProcessingError(c, err, result)
		return
	}

	// Success response
	c.JSON(http.StatusOK, WebhookSuccessResponse{
		Received:  true,
		EventID:   result.EventID,
		EventType: result.EventType,
	})
}

// handleProcessingError handles webhook processing errors
func (h *StripeWebhookHandler) handleProcessingError(c *gin.Context, err error, result *stripeservice.WebhookResult) {
	statusCode := http.StatusBadRequest
	errorCode := "PROCESSING_ERROR"
	errorMsg := "Webhook processing failed"

	switch err {
	case stripeservice.ErrInvalidSignature:
		statusCode = http.StatusUnauthorized
		errorCode = "INVALID_SIGNATURE"
		errorMsg = "Invalid webhook signature"

	case stripeservice.ErrEventExpired:
		statusCode = http.StatusBadRequest
		errorCode = "EVENT_EXPIRED"
		errorMsg = "Webhook event has expired"

	case stripeservice.ErrEventAlreadyProcessed:
		// Already processed is actually a success case
		c.JSON(http.StatusOK, WebhookSuccessResponse{
			Received:  true,
			EventID:   result.EventID,
			EventType: result.EventType,
		})
		return

	case stripeservice.ErrMaxRetriesExceeded:
		statusCode = http.StatusUnprocessableEntity
		errorCode = "MAX_RETRIES_EXCEEDED"
		errorMsg = "Event processing failed after maximum retries"

	default:
		statusCode = http.StatusInternalServerError
		errorCode = "INTERNAL_ERROR"
		errorMsg = "Internal server error"
	}

	h.logError(errorMsg, err)

	c.JSON(statusCode, WebhookErrorResponse{
		Received: false,
		Error:    errorMsg,
		Code:     errorCode,
	})
}

// =============================================================================
// Admin Endpoints
// =============================================================================

// GetWebhookEvents retrieves webhook events with filtering
// @Summary Get webhook events
// @Description Retrieves webhook events with optional filtering
// @Tags webhooks
// @Accept json
// @Produce json
// @Param source query string false "Event source (stripe)"
// @Param event_type query string false "Event type filter"
// @Param status query string false "Status filter"
// @Param limit query int false "Limit results"
// @Param offset query int false "Offset for pagination"
// @Success 200 {array} models.WebhookEvent
// @Failure 500 {object} WebhookErrorResponse
// @Router /api/v1/admin/webhooks/events [get]
func (h *StripeWebhookHandler) GetWebhookEvents(c *gin.Context) {
	opts := &models.WebhookEventQueryOptions{
		Source:    models.WebhookEventSource(c.Query("source")),
		EventType: c.Query("event_type"),
		Status:    models.WebhookEventStatus(c.Query("status")),
	}

	// Parse limit
	if limit := c.Query("limit"); limit != "" {
		var l int
		if _, err := parseIntParam(limit, &l); err == nil && l > 0 {
			opts.Limit = l
		}
	} else {
		opts.Limit = 100 // Default limit
	}

	// Parse offset
	if offset := c.Query("offset"); offset != "" {
		var o int
		if _, err := parseIntParam(offset, &o); err == nil && o >= 0 {
			opts.Offset = o
		}
	}

	events, err := h.service.GetWebhookEvents(c.Request.Context(), opts)
	if err != nil {
		h.logError("Failed to get webhook events", err)
		c.JSON(http.StatusInternalServerError, WebhookErrorResponse{
			Received: false,
			Error:    "Failed to retrieve webhook events",
			Code:     "DATABASE_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events,
		"count":  len(events),
		"limit":  opts.Limit,
		"offset": opts.Offset,
	})
}

// GetWebhookStats retrieves webhook statistics
// @Summary Get webhook statistics
// @Description Retrieves aggregated webhook event statistics
// @Tags webhooks
// @Produce json
// @Success 200 {object} WebhookStatsResponse
// @Failure 500 {object} WebhookErrorResponse
// @Router /api/v1/admin/webhooks/stats [get]
func (h *StripeWebhookHandler) GetWebhookStats(c *gin.Context) {
	stats, err := h.service.GetWebhookStats(c.Request.Context())
	if err != nil {
		h.logError("Failed to get webhook stats", err)
		c.JSON(http.StatusInternalServerError, WebhookErrorResponse{
			Received: false,
			Error:    "Failed to retrieve webhook statistics",
			Code:     "DATABASE_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, WebhookStatsResponse{
		TotalEvents:     stats.TotalEvents,
		ProcessedEvents: stats.ProcessedEvents,
		FailedEvents:    stats.FailedEvents,
		PendingEvents:   stats.PendingEvents,
		RetryingEvents:  stats.RetryingEvents,
		EventsByType:    stats.EventsByType,
		AvgProcessingMs: stats.AvgProcessingMs,
	})
}

// GetWebhookEvent retrieves a specific webhook event by ID
// @Summary Get webhook event by ID
// @Description Retrieves a specific webhook event
// @Tags webhooks
// @Produce json
// @Param id path string true "Event ID"
// @Success 200 {object} models.WebhookEvent
// @Failure 404 {object} WebhookErrorResponse
// @Failure 500 {object} WebhookErrorResponse
// @Router /api/v1/admin/webhooks/events/{id} [get]
func (h *StripeWebhookHandler) GetWebhookEvent(c *gin.Context) {
	idParam := c.Param("id")
	eventID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, WebhookErrorResponse{
			Received: false,
			Error:    "Invalid event ID format",
			Code:     "INVALID_ID",
		})
		return
	}

	opts := &models.WebhookEventQueryOptions{
		Limit: 1,
	}

	events, err := h.service.GetWebhookEvents(c.Request.Context(), opts)
	if err != nil {
		h.logError("Failed to get webhook event", err)
		c.JSON(http.StatusInternalServerError, WebhookErrorResponse{
			Received: false,
			Error:    "Failed to retrieve webhook event",
			Code:     "DATABASE_ERROR",
		})
		return
	}

	// Find the specific event
	for _, event := range events {
		if event.ID == eventID {
			c.JSON(http.StatusOK, event)
			return
		}
	}

	c.JSON(http.StatusNotFound, WebhookErrorResponse{
		Received: false,
		Error:    "Webhook event not found",
		Code:     "NOT_FOUND",
	})
}

// ReprocessWebhookEvent reprocesses a failed webhook event
// @Summary Reprocess webhook event
// @Description Manually reprocesses a failed webhook event
// @Tags webhooks
// @Produce json
// @Param id path string true "Event ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} WebhookErrorResponse
// @Failure 500 {object} WebhookErrorResponse
// @Router /api/v1/admin/webhooks/events/{id}/reprocess [post]
func (h *StripeWebhookHandler) ReprocessWebhookEvent(c *gin.Context) {
	idParam := c.Param("id")
	eventID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, WebhookErrorResponse{
			Received: false,
			Error:    "Invalid event ID format",
			Code:     "INVALID_ID",
		})
		return
	}

	if err := h.service.ReprocessEvent(c.Request.Context(), eventID); err != nil {
		h.logError("Failed to reprocess webhook event", err)
		c.JSON(http.StatusInternalServerError, WebhookErrorResponse{
			Received: false,
			Error:    "Failed to reprocess webhook event: " + err.Error(),
			Code:     "REPROCESS_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Event queued for reprocessing",
		"event_id": eventID.String(),
	})
}

// =============================================================================
// Health Check
// =============================================================================

// HealthCheck returns the health status of the webhook handler
// @Summary Webhook health check
// @Description Returns health status of webhook processing
// @Tags webhooks
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/webhooks/health [get]
func (h *StripeWebhookHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "stripe-webhook-handler",
	})
}

// =============================================================================
// Route Registration
// =============================================================================

// RegisterRoutes registers all webhook routes
func (h *StripeWebhookHandler) RegisterRoutes(router *gin.RouterGroup) {
	// Public webhook endpoint (no auth - Stripe needs to access it)
	webhooks := router.Group("/webhooks")
	{
		webhooks.POST("/stripe", h.HandleWebhook)
		webhooks.GET("/health", h.HealthCheck)
	}
}

// RegisterAdminRoutes registers admin-only webhook routes
func (h *StripeWebhookHandler) RegisterAdminRoutes(router *gin.RouterGroup) {
	// Admin endpoints (requires authentication)
	admin := router.Group("/admin/webhooks")
	{
		admin.GET("/events", h.GetWebhookEvents)
		admin.GET("/events/:id", h.GetWebhookEvent)
		admin.POST("/events/:id/reprocess", h.ReprocessWebhookEvent)
		admin.GET("/stats", h.GetWebhookStats)
	}
}

// =============================================================================
// Helper Functions
// =============================================================================

func (h *StripeWebhookHandler) logError(message string, err error) {
	if err != nil {
		log.Printf("[Stripe Webhook] %s: %v", message, err)
	} else {
		log.Printf("[Stripe Webhook] %s", message)
	}
}

func parseIntParam(value string, result *int) (bool, error) {
	n := 0
	for _, c := range value {
		if c < '0' || c > '9' {
			return false, nil
		}
		n = n*10 + int(c-'0')
	}
	*result = n
	return true, nil
}

// =============================================================================
// Middleware for IP Validation (Optional Security)
// =============================================================================

// StripeIPWhitelist contains Stripe's webhook IP addresses
// Note: In production, verify these IPs from Stripe's documentation
var StripeIPWhitelist = []string{
	// Stripe's webhook IPs - update from https://stripe.com/docs/ips
	"3.18.12.63",
	"3.130.192.231",
	"13.235.14.237",
	"13.235.122.149",
	"18.211.135.69",
	"35.154.171.200",
	"52.15.183.38",
	"54.88.130.119",
	"54.88.130.237",
	"54.187.174.169",
	"54.187.205.235",
	"54.187.216.72",
}

// ValidateStripeIP middleware validates the request comes from Stripe's IPs
func ValidateStripeIP(allowedIPs []string) gin.HandlerFunc {
	ipSet := make(map[string]bool)
	for _, ip := range allowedIPs {
		ipSet[ip] = true
	}

	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		// Skip validation if no IPs configured or in test mode
		if len(allowedIPs) == 0 {
			c.Next()
			return
		}

		// Check if IP is in whitelist
		if !ipSet[clientIP] {
			// Log the rejected IP for monitoring
			c.JSON(http.StatusForbidden, WebhookErrorResponse{
				Received: false,
				Error:    "Request from unauthorized IP",
				Code:     "UNAUTHORIZED_IP",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitWebhook provides rate limiting for webhook endpoints
func RateLimitWebhook(maxRequests int, window int) gin.HandlerFunc {
	// Simple in-memory rate limiter
	// In production, use Redis-based rate limiting
	return func(c *gin.Context) {
		// Placeholder - implement with your rate limiting solution
		c.Next()
	}
}
