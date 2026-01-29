package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// PublicInvoiceHandler handles public invoice payment endpoints
type PublicInvoiceHandler struct {
	invoicePaymentService *services.InvoicePaymentService
	invoiceService        *services.InvoiceService
}

// NewPublicInvoiceHandler creates a new public invoice handler
func NewPublicInvoiceHandler(
	invoicePaymentService *services.InvoicePaymentService,
	invoiceService *services.InvoiceService,
) *PublicInvoiceHandler {
	return &PublicInvoiceHandler{
		invoicePaymentService: invoicePaymentService,
		invoiceService:        invoiceService,
	}
}

// PayInvoice handles the payment initiation for an invoice
// GET /public/invoices/pay/:token
// This endpoint validates the token, creates a Stripe Checkout session,
// and redirects the user to Stripe's hosted checkout page
func (h *PublicInvoiceHandler) PayInvoice(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment token is required"})
		return
	}

	ctx := c.Request.Context()

	// Create checkout session (this also validates the token)
	result, err := h.invoicePaymentService.CreateCheckoutSessionForInvoice(ctx, token)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorMessage := "failed to create payment session"

		switch {
		case errors.Is(err, services.ErrInvalidPaymentToken):
			statusCode = http.StatusNotFound
			errorMessage = "invalid payment link"
		case errors.Is(err, services.ErrPaymentTokenExpired):
			statusCode = http.StatusGone
			errorMessage = "this payment link has expired"
		case errors.Is(err, services.ErrPaymentTokenVoided):
			statusCode = http.StatusGone
			errorMessage = "this payment link is no longer valid"
		case errors.Is(err, services.ErrInvoiceNotPayable):
			statusCode = http.StatusConflict
			errorMessage = "this invoice cannot be paid at this time"
		case errors.Is(err, services.ErrCheckoutSessionCreationFailed):
			statusCode = http.StatusServiceUnavailable
			errorMessage = "payment service temporarily unavailable"
		}

		c.JSON(statusCode, gin.H{"error": errorMessage})
		return
	}

	// Redirect to Stripe Checkout
	c.Redirect(http.StatusSeeOther, result.URL)
}

// GetInvoiceByToken retrieves invoice details for a payment token
// GET /public/invoices/details/:token
// This endpoint returns invoice details for display on a payment page
func (h *PublicInvoiceHandler) GetInvoiceByToken(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment token is required"})
		return
	}

	ctx := c.Request.Context()

	// Get invoice by token
	invoice, err := h.invoicePaymentService.GetInvoiceByToken(ctx, token)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorMessage := "failed to retrieve invoice"

		if errors.Is(err, services.ErrInvalidPaymentToken) {
			statusCode = http.StatusNotFound
			errorMessage = "invoice not found"
		}

		c.JSON(statusCode, gin.H{"error": errorMessage})
		return
	}

	// Return a sanitized view of the invoice for public display
	response := h.buildPublicInvoiceResponse(invoice)

	c.JSON(http.StatusOK, response)
}

// ValidatePaymentToken validates a payment token without initiating payment
// GET /public/invoices/validate/:token
// Returns the validation status and basic invoice info
func (h *PublicInvoiceHandler) ValidatePaymentToken(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment token is required"})
		return
	}

	ctx := c.Request.Context()

	// Validate the token
	invoice, err := h.invoicePaymentService.ValidateToken(ctx, token)
	if err != nil {
		errorMessage := "invalid token"
		switch {
		case errors.Is(err, services.ErrInvalidPaymentToken):
			errorMessage = "invalid payment link"
		case errors.Is(err, services.ErrPaymentTokenExpired):
			errorMessage = "payment link has expired"
		case errors.Is(err, services.ErrPaymentTokenVoided):
			errorMessage = "payment link is no longer valid"
		case errors.Is(err, services.ErrInvoiceNotPayable):
			errorMessage = "invoice cannot be paid"
		}

		c.JSON(http.StatusOK, gin.H{
			"valid": false,
			"error": errorMessage,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":          true,
		"invoice_number": invoice.InvoiceNumber,
		"total_amount":   invoice.TotalAmount.StringFixed(2),
		"currency":       "USD",
	})
}

// PublicInvoiceResponse is a sanitized invoice response for public display
type PublicInvoiceResponse struct {
	InvoiceNumber string                      `json:"invoice_number"`
	IssueDate     string                      `json:"issue_date"`
	DueDate       string                      `json:"due_date"`
	Status        string                      `json:"status"`
	Subtotal      string                      `json:"subtotal"`
	TaxAmount     string                      `json:"tax_amount"`
	TotalAmount   string                      `json:"total_amount"`
	AmountPaid    string                      `json:"amount_paid"`
	AmountDue     string                      `json:"amount_due"`
	Currency      string                      `json:"currency"`
	CustomerName  string                      `json:"customer_name,omitempty"`
	Lines         []PublicInvoiceLineResponse `json:"lines"`
}

// PublicInvoiceLineResponse is a sanitized invoice line for public display
type PublicInvoiceLineResponse struct {
	Description string `json:"description"`
	Quantity    string `json:"quantity"`
	UnitPrice   string `json:"unit_price"`
	Total       string `json:"total"`
}

// buildPublicInvoiceResponse builds a sanitized invoice response for public display
func (h *PublicInvoiceHandler) buildPublicInvoiceResponse(invoice *models.Invoice) *PublicInvoiceResponse {
	// Build line items
	lines := make([]PublicInvoiceLineResponse, 0, len(invoice.Lines))
	for _, line := range invoice.Lines {
		lineTotal := line.Quantity.Mul(line.UnitPrice)
		lines = append(lines, PublicInvoiceLineResponse{
			Description: line.Description,
			Quantity:    line.Quantity.StringFixed(2),
			UnitPrice:   line.UnitPrice.StringFixed(2),
			Total:       lineTotal.StringFixed(2),
		})
	}

	// Calculate amount due
	amountDue := invoice.TotalAmount.Sub(invoice.AmountPaid)

	// Get customer name
	customerName := ""
	if invoice.Customer != nil {
		if invoice.Customer.CompanyName != nil && *invoice.Customer.CompanyName != "" {
			customerName = *invoice.Customer.CompanyName
		} else {
			customerName = invoice.Customer.FirstName + " " + invoice.Customer.LastName
		}
	}

	return &PublicInvoiceResponse{
		InvoiceNumber: invoice.InvoiceNumber,
		IssueDate:     invoice.IssueDate.Format("2006-01-02"),
		DueDate:       invoice.DueDate.Format("2006-01-02"),
		Status:        string(invoice.Status),
		Subtotal:      invoice.Subtotal.StringFixed(2),
		TaxAmount:     invoice.TaxAmount.StringFixed(2),
		TotalAmount:   invoice.TotalAmount.StringFixed(2),
		AmountPaid:    invoice.AmountPaid.StringFixed(2),
		AmountDue:     amountDue.StringFixed(2),
		Currency:      "USD",
		CustomerName:  customerName,
		Lines:         lines,
	}
}
