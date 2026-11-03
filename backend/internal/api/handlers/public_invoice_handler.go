package handlers

import (
	"errors"
	"net/http"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"

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
// @Summary		Pay invoice
// @Description	Validates the payment token, creates a Stripe Checkout session on the business's own Stripe account (a direct charge with ServicePro's fee: bank 1% capped at $10, card 0.4%), and redirects to Stripe's hosted checkout page. Without a method, bank is used when the business accepts it.
// @Tags			Public
// @Accept			json
// @Produce		json
// @Param			token	path	string	true	"Payment token"
// @Param			method	query	string	false	"Payment method"	Enums(bank, card)
// @Success		303
// @Failure		400	{object}	models.ErrorResponse
// @Failure		404	{object}	models.ErrorResponse
// @Failure		409	{object}	models.ErrorResponse
// @Failure		410	{object}	models.ErrorResponse
// @Failure		500	{object}	models.ErrorResponse
// @Failure		503	{object}	models.ErrorResponse
// @Router			/public/invoices/pay/{token} [get]
func (h *PublicInvoiceHandler) PayInvoice(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: "payment token is required"})
		return
	}

	// Public, token-authenticated route: there is no tenant session. The
	// unguessable payment token selects the invoice across tenants.
	ctx := tenancy.WithSystem(c.Request.Context())

	// Create checkout session (this also validates the token)
	result, err := h.invoicePaymentService.CreateCheckoutSessionForInvoice(ctx, token, c.Query("method"))
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
		case errors.Is(err, services.ErrInvoicePaymentDisabled):
			statusCode = http.StatusConflict
			errorMessage = "online payment isn't available for this invoice. Please contact the business to pay"
		case errors.Is(err, services.ErrPaymentPending):
			statusCode = http.StatusConflict
			errorMessage = "a bank payment for this invoice is already processing"
		case errors.Is(err, services.ErrPaymentMethodUnavailable):
			statusCode = http.StatusConflict
			errorMessage = "this business can't take that payment method yet. Please choose another way to pay"
		case errors.Is(err, services.ErrInvalidPaymentMethod):
			statusCode = http.StatusBadRequest
			errorMessage = "unknown payment method"
		case errors.Is(err, services.ErrCheckoutSessionCreationFailed):
			statusCode = http.StatusServiceUnavailable
			errorMessage = "payment service temporarily unavailable"
		}

		if statusCode == http.StatusInternalServerError {
			apierror.Internal(c, err, sentenceCase(errorMessage))
			return
		}
		apierror.WriteResponse(c, statusCode, models.ErrorResponse{Error: sentenceCase(errorMessage)})
		return
	}

	// Redirect to Stripe Checkout
	c.Redirect(http.StatusSeeOther, result.URL)
}

// GetInvoiceByToken retrieves invoice details for a payment token
// @Summary		Get invoice by token
// @Description	Retrieves invoice details for the public pay page (mockup 17): the invoice, the business that sent it (name, city/state), and the methods it can be paid with now, each with the fee the business pays ServicePro for it (bank 1% capped at $10, card 0.4%). pay_methods is empty when the invoice can't be paid online.
// @Tags			Public
// @Accept			json
// @Produce		json
// @Param			token	path		string	true	"Payment token"
// @Success		200		{object}	PublicInvoiceResponse
// @Failure		400		{object}	models.ErrorResponse
// @Failure		404		{object}	models.ErrorResponse
// @Failure		500		{object}	models.ErrorResponse
// @Router			/public/invoices/details/{token} [get]
func (h *PublicInvoiceHandler) GetInvoiceByToken(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: "payment token is required"})
		return
	}

	// Public, token-authenticated route: there is no tenant session. The
	// unguessable payment token selects the invoice across tenants.
	ctx := tenancy.WithSystem(c.Request.Context())

	page, err := h.invoicePaymentService.GetPayPage(ctx, token)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorMessage := "failed to retrieve invoice"

		if errors.Is(err, services.ErrInvalidPaymentToken) {
			statusCode = http.StatusNotFound
			errorMessage = "invoice not found"
		}

		if statusCode == http.StatusInternalServerError {
			apierror.Internal(c, err, sentenceCase(errorMessage))
			return
		}
		apierror.WriteResponse(c, statusCode, models.ErrorResponse{Error: sentenceCase(errorMessage)})
		return
	}

	// Return a sanitized view of the invoice for public display
	response := h.buildPublicInvoiceResponse(page.Invoice)
	response.BusinessName = page.BusinessName
	response.BusinessLocation = page.BusinessLocation
	response.PaymentPending = page.PaymentPending
	response.PayMethods = make([]PublicPayMethodResponse, 0, len(page.Methods))
	for _, m := range page.Methods {
		response.PayMethods = append(response.PayMethods, PublicPayMethodResponse{
			Method: m.Method, BusinessFee: m.BusinessFee.StringFixed(2),
		})
	}

	c.JSON(http.StatusOK, response)
}

// ValidatePaymentToken validates a payment token without initiating payment
// @Summary		Validate payment token
// @Description	Validates a payment token without initiating payment, returns validation status and basic invoice info
// @Tags			Public
// @Accept			json
// @Produce		json
// @Param			token	path		string	true	"Payment token"
// @Success		200		{object}	map[string]interface{}
// @Failure		400		{object}	models.ErrorResponse
// @Router			/public/invoices/validate/{token} [get]
func (h *PublicInvoiceHandler) ValidatePaymentToken(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: "payment token is required"})
		return
	}

	// Public, token-authenticated route: there is no tenant session. The
	// unguessable payment token selects the invoice across tenants.
	ctx := tenancy.WithSystem(c.Request.Context())

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
	PONumber      string                      `json:"po_number,omitempty"`
	Lines         []PublicInvoiceLineResponse `json:"lines"`
	// BusinessName and BusinessLocation ("Columbus, OH", may be empty)
	// describe the business that sent the invoice (T170).
	BusinessName     string `json:"business_name"`
	BusinessLocation string `json:"business_location,omitempty"`
	// PayMethods are the ways this invoice can be paid online right now,
	// bank first (preferred for B2B); empty when it can't be.
	PayMethods []PublicPayMethodResponse `json:"pay_methods"`
	// PaymentPending: the customer's bank payment is still settling
	// (pay_methods is then empty).
	PaymentPending bool `json:"payment_pending"`
}

// PublicPayMethodResponse is one way to pay on the pay page.
type PublicPayMethodResponse struct {
	// Method is passed to /public/invoices/pay/{token}?method=.
	Method string `json:"method" enums:"bank,card"`
	// BusinessFee is ServicePro's fee the business pays for this method
	// (the customer pays no fee).
	BusinessFee string `json:"business_fee" example:"10.00"`
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
		Currency:      models.NormalizeCurrency(invoice.Currency),
		CustomerName:  customerName,
		PONumber:      invoice.PONumber,
		Lines:         lines,
	}
}
