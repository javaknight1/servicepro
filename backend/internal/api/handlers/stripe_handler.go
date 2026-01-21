package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	stripeService "github.com/javaknight1/servicepro/backend/internal/services/stripe"
)

// StripeHandler handles Stripe-related HTTP requests
type StripeHandler struct {
	client         *stripeService.Client
	webhookHandler *stripeService.WebhookHandler
}

// NewStripeHandler creates a new Stripe handler
func NewStripeHandler(client *stripeService.Client, webhookHandler *stripeService.WebhookHandler) *StripeHandler {
	return &StripeHandler{
		client:         client,
		webhookHandler: webhookHandler,
	}
}

// CreatePaymentIntentRequest represents the request to create a payment intent
type CreatePaymentIntentRequest struct {
	Amount              float64           `json:"amount" binding:"required,gt=0"`
	Currency            string            `json:"currency" binding:"required,len=3"`
	CustomerID          *string           `json:"customer_id"`
	PaymentMethodID     *string           `json:"payment_method_id"`
	Description         *string           `json:"description"`
	Metadata            map[string]string `json:"metadata"`
	StatementDescriptor *string           `json:"statement_descriptor"`
	ReceiptEmail        *string           `json:"receipt_email"`
	SetupFutureUsage    *string           `json:"setup_future_usage"`
	CaptureMethod       *string           `json:"capture_method"`
	ConfirmationMethod  *string           `json:"confirmation_method"`
	ReturnURL           *string           `json:"return_url"`
}

// CreatePaymentIntent godoc
// @Summary Create a payment intent
// @Description Creates a new Stripe payment intent for processing a payment
// @Tags stripe
// @Accept json
// @Produce json
// @Param request body CreatePaymentIntentRequest true "Payment intent details"
// @Success 200 {object} stripeService.PaymentIntentResult
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /stripe/payment-intents [post]
func (h *StripeHandler) CreatePaymentIntent(c *gin.Context) {
	var req CreatePaymentIntentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	// Convert amount from dollars to cents
	amountCents := stripeService.ConvertDecimalToCents(decimal.NewFromFloat(req.Amount))

	// Generate idempotency key
	idempotencyKey := uuid.New().String()

	params := &stripeService.PaymentIntentParams{
		Amount:              amountCents,
		Currency:            req.Currency,
		CustomerID:          req.CustomerID,
		PaymentMethodID:     req.PaymentMethodID,
		Description:         req.Description,
		Metadata:            req.Metadata,
		StatementDescriptor: req.StatementDescriptor,
		ReceiptEmail:        req.ReceiptEmail,
		SetupFutureUsage:    req.SetupFutureUsage,
		CaptureMethod:       req.CaptureMethod,
		ConfirmationMethod:  req.ConfirmationMethod,
		ReturnURL:           req.ReturnURL,
		IdempotencyKey:      &idempotencyKey,
	}

	result, err := h.client.CreatePaymentIntent(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create payment intent", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetPaymentIntent godoc
// @Summary Get a payment intent
// @Description Retrieves a payment intent by ID
// @Tags stripe
// @Accept json
// @Produce json
// @Param id path string true "Payment Intent ID"
// @Success 200 {object} stripeService.PaymentIntentResult
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /stripe/payment-intents/{id} [get]
func (h *StripeHandler) GetPaymentIntent(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payment intent ID is required"})
		return
	}

	result, err := h.client.GetPaymentIntent(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get payment intent", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ConfirmPaymentIntentRequest represents the request to confirm a payment intent
type ConfirmPaymentIntentRequest struct {
	PaymentMethodID *string `json:"payment_method_id"`
}

// ConfirmPaymentIntent godoc
// @Summary Confirm a payment intent
// @Description Confirms a payment intent
// @Tags stripe
// @Accept json
// @Produce json
// @Param id path string true "Payment Intent ID"
// @Param request body ConfirmPaymentIntentRequest false "Confirmation details"
// @Success 200 {object} stripeService.PaymentIntentResult
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /stripe/payment-intents/{id}/confirm [post]
func (h *StripeHandler) ConfirmPaymentIntent(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payment intent ID is required"})
		return
	}

	var req ConfirmPaymentIntentRequest
	_ = c.ShouldBindJSON(&req) // Optional body

	result, err := h.client.ConfirmPaymentIntent(c.Request.Context(), id, req.PaymentMethodID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to confirm payment intent", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// CancelPaymentIntentRequest represents the request to cancel a payment intent
type CancelPaymentIntentRequest struct {
	Reason *string `json:"reason"`
}

// CancelPaymentIntent godoc
// @Summary Cancel a payment intent
// @Description Cancels a payment intent
// @Tags stripe
// @Accept json
// @Produce json
// @Param id path string true "Payment Intent ID"
// @Param request body CancelPaymentIntentRequest false "Cancellation details"
// @Success 200 {object} stripeService.PaymentIntentResult
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /stripe/payment-intents/{id}/cancel [post]
func (h *StripeHandler) CancelPaymentIntent(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payment intent ID is required"})
		return
	}

	var req CancelPaymentIntentRequest
	_ = c.ShouldBindJSON(&req) // Optional body

	result, err := h.client.CancelPaymentIntent(c.Request.Context(), id, req.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel payment intent", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// CapturePaymentIntentRequest represents the request to capture a payment intent
type CapturePaymentIntentRequest struct {
	AmountToCapture *float64 `json:"amount_to_capture"`
}

// CapturePaymentIntent godoc
// @Summary Capture a payment intent
// @Description Captures a payment intent (for manual capture)
// @Tags stripe
// @Accept json
// @Produce json
// @Param id path string true "Payment Intent ID"
// @Param request body CapturePaymentIntentRequest false "Capture details"
// @Success 200 {object} stripeService.PaymentIntentResult
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /stripe/payment-intents/{id}/capture [post]
func (h *StripeHandler) CapturePaymentIntent(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payment intent ID is required"})
		return
	}

	var req CapturePaymentIntentRequest
	_ = c.ShouldBindJSON(&req) // Optional body

	var amountCents *int64
	if req.AmountToCapture != nil {
		cents := stripeService.ConvertDecimalToCents(decimal.NewFromFloat(*req.AmountToCapture))
		amountCents = &cents
	}

	result, err := h.client.CapturePaymentIntent(c.Request.Context(), id, amountCents)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to capture payment intent", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// CreateCustomerRequest represents the request to create a customer
type CreateCustomerRequest struct {
	Email           *string           `json:"email"`
	Name            *string           `json:"name"`
	Phone           *string           `json:"phone"`
	Description     *string           `json:"description"`
	Metadata        map[string]string `json:"metadata"`
	PaymentMethodID *string           `json:"payment_method_id"`
}

// CreateCustomer godoc
// @Summary Create a customer
// @Description Creates a new Stripe customer
// @Tags stripe
// @Accept json
// @Produce json
// @Param request body CreateCustomerRequest true "Customer details"
// @Success 200 {object} stripeService.CustomerResult
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /stripe/customers [post]
func (h *StripeHandler) CreateCustomer(c *gin.Context) {
	var req CreateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	idempotencyKey := uuid.New().String()

	params := &stripeService.CustomerParams{
		Email:           req.Email,
		Name:            req.Name,
		Phone:           req.Phone,
		Description:     req.Description,
		Metadata:        req.Metadata,
		PaymentMethodID: req.PaymentMethodID,
		IdempotencyKey:  &idempotencyKey,
	}

	result, err := h.client.CreateCustomer(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create customer", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetCustomer godoc
// @Summary Get a customer
// @Description Retrieves a customer by ID
// @Tags stripe
// @Accept json
// @Produce json
// @Param id path string true "Customer ID"
// @Success 200 {object} stripeService.CustomerResult
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /stripe/customers/{id} [get]
func (h *StripeHandler) GetCustomer(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Customer ID is required"})
		return
	}

	result, err := h.client.GetCustomer(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get customer", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// CreateRefundRequest represents the request to create a refund
type CreateRefundRequest struct {
	ChargeID             *string           `json:"charge_id"`
	PaymentIntentID      *string           `json:"payment_intent_id"`
	Amount               *float64          `json:"amount"`
	Reason               *string           `json:"reason"`
	Metadata             map[string]string `json:"metadata"`
	RefundApplicationFee *bool             `json:"refund_application_fee"`
	ReverseTransfer      *bool             `json:"reverse_transfer"`
}

// CreateRefund godoc
// @Summary Create a refund
// @Description Creates a refund for a charge or payment intent
// @Tags stripe
// @Accept json
// @Produce json
// @Param request body CreateRefundRequest true "Refund details"
// @Success 200 {object} stripeService.RefundResult
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Security BearerAuth
// @Router /stripe/refunds [post]
func (h *StripeHandler) CreateRefund(c *gin.Context) {
	var req CreateRefundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request", "details": err.Error()})
		return
	}

	if req.ChargeID == nil && req.PaymentIntentID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Either charge_id or payment_intent_id is required"})
		return
	}

	idempotencyKey := uuid.New().String()

	params := &stripeService.RefundParams{
		ChargeID:             req.ChargeID,
		PaymentIntentID:      req.PaymentIntentID,
		Reason:               req.Reason,
		Metadata:             req.Metadata,
		RefundApplicationFee: req.RefundApplicationFee,
		ReverseTransfer:      req.ReverseTransfer,
		IdempotencyKey:       &idempotencyKey,
	}

	if req.Amount != nil {
		cents := stripeService.ConvertDecimalToCents(decimal.NewFromFloat(*req.Amount))
		params.Amount = &cents
	}

	result, err := h.client.CreateRefund(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create refund", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// HandleWebhook handles incoming Stripe webhooks
func (h *StripeHandler) HandleWebhook(c *gin.Context) {
	h.webhookHandler.HandleHTTPWebhook(c.Writer, c.Request)
}

// GetWebhookStats godoc
// @Summary Get webhook statistics
// @Description Returns webhook processing statistics
// @Tags stripe
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security BearerAuth
// @Router /stripe/webhooks/stats [get]
func (h *StripeHandler) GetWebhookStats(c *gin.Context) {
	stats := map[string]interface{}{
		"processed_events_count": h.webhookHandler.GetProcessedEventsCount(),
	}

	c.JSON(http.StatusOK, stats)
}
