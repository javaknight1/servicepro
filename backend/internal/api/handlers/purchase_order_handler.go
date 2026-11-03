package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// PurchaseOrderHandler serves a customer's purchase orders (T187) under
// /customers/{id}/purchase-orders. The tenant comes from the request context.
type PurchaseOrderHandler struct {
	pos services.PurchaseOrderServiceInterface
}

// NewPurchaseOrderHandler creates a purchase-order handler.
func NewPurchaseOrderHandler(pos services.PurchaseOrderServiceInterface) *PurchaseOrderHandler {
	return &PurchaseOrderHandler{pos: pos}
}

// ListPurchaseOrders handles GET /api/v1/customers/:id/purchase-orders
// @Summary List a customer's purchase orders
// @Description The customer's POs, newest first, each with what invoices have billed against it (issued invoices with that PO number: sent, not draft or cancelled) and what remains.
// @Tags purchase-orders
// @Produce json
// @Security BearerAuth
// @Param id path string true "Customer ID"
// @Success 200 {object} models.PurchaseOrderListResponse
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 404 {object} models.ErrorResponse "Customer not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/customers/{id}/purchase-orders [get]
func (h *PurchaseOrderHandler) ListPurchaseOrders(c *gin.Context) {
	customerID, ok := parseUUIDParam(c, "id", "customer")
	if !ok {
		return
	}
	pos, err := h.pos.List(c.Request.Context(), customerID)
	if err != nil {
		apierror.Respond(c, err, "Failed to list purchase orders")
		return
	}
	c.JSON(http.StatusOK, models.PurchaseOrderListResponse{PurchaseOrders: pos, Total: len(pos)})
}

// CreatePurchaseOrder handles POST /api/v1/customers/:id/purchase-orders
// @Summary Record a customer purchase order
// @Description Records a PO number with the amount it authorises and an optional expiry. Invoices for the customer carrying that PO number count against it.
// @Tags purchase-orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Customer ID"
// @Param request body models.CreatePurchaseOrderRequest true "Purchase order"
// @Success 201 {object} models.PurchaseOrderResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 404 {object} models.ErrorResponse "Customer not found"
// @Failure 409 {object} models.ErrorResponse "The customer already has a PO with that number (purchase_order_exists)"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/customers/{id}/purchase-orders [post]
func (h *PurchaseOrderHandler) CreatePurchaseOrder(c *gin.Context) {
	customerID, ok := parseUUIDParam(c, "id", "customer")
	if !ok {
		return
	}
	userID, exists := c.Get("user_id")
	uid, isUUID := userID.(uuid.UUID)
	if !exists || !isUUID {
		apierror.Write(c, http.StatusUnauthorized, "unauthorized", "User not authenticated")
		return
	}
	var req models.CreatePurchaseOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.BindError(c, err)
		return
	}
	po, err := h.pos.Create(c.Request.Context(), customerID, &req, uid)
	if err != nil {
		apierror.Respond(c, err, "Failed to create purchase order")
		return
	}
	c.JSON(http.StatusCreated, po)
}

// UpdatePurchaseOrder handles PUT /api/v1/customers/:id/purchase-orders/:poId
// @Summary Update a customer purchase order
// @Description Changes the fields present in the body; clear_expires_on removes the expiry.
// @Tags purchase-orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Customer ID"
// @Param poId path string true "Purchase order ID"
// @Param request body models.UpdatePurchaseOrderRequest true "Fields to change"
// @Success 200 {object} models.PurchaseOrderResponse
// @Failure 400 {object} models.ErrorResponse "Invalid request"
// @Failure 404 {object} models.ErrorResponse "Customer or purchase order not found"
// @Failure 409 {object} models.ErrorResponse "The customer already has a PO with that number (purchase_order_exists)"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/customers/{id}/purchase-orders/{poId} [put]
func (h *PurchaseOrderHandler) UpdatePurchaseOrder(c *gin.Context) {
	customerID, ok := parseUUIDParam(c, "id", "customer")
	if !ok {
		return
	}
	poID, ok := parseUUIDParam(c, "poId", "purchase order")
	if !ok {
		return
	}
	var req models.UpdatePurchaseOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		apierror.BindError(c, err)
		return
	}
	po, err := h.pos.Update(c.Request.Context(), customerID, poID, &req)
	if err != nil {
		apierror.Respond(c, err, "Failed to update purchase order")
		return
	}
	c.JSON(http.StatusOK, po)
}

// DeletePurchaseOrder handles DELETE /api/v1/customers/:id/purchase-orders/:poId
// @Summary Delete a customer purchase order
// @Description Stops tracking the PO. Invoices keep their PO number.
// @Tags purchase-orders
// @Security BearerAuth
// @Param id path string true "Customer ID"
// @Param poId path string true "Purchase order ID"
// @Success 204 "Deleted"
// @Failure 400 {object} models.ErrorResponse "Invalid ID"
// @Failure 404 {object} models.ErrorResponse "Customer or purchase order not found"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/customers/{id}/purchase-orders/{poId} [delete]
func (h *PurchaseOrderHandler) DeletePurchaseOrder(c *gin.Context) {
	customerID, ok := parseUUIDParam(c, "id", "customer")
	if !ok {
		return
	}
	poID, ok := parseUUIDParam(c, "poId", "purchase order")
	if !ok {
		return
	}
	if err := h.pos.Delete(c.Request.Context(), customerID, poID); err != nil {
		apierror.Respond(c, err, "Failed to delete purchase order")
		return
	}
	c.Status(http.StatusNoContent)
}
