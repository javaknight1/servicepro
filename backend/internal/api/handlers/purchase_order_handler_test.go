package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

// fakePOs is an in-memory PurchaseOrderServiceInterface for one customer.
type fakePOs struct {
	customer uuid.UUID
	pos      map[uuid.UUID]*models.PurchaseOrderResponse
}

func (f *fakePOs) List(_ context.Context, customerID uuid.UUID) ([]models.PurchaseOrderResponse, error) {
	if customerID != f.customer {
		return nil, services.ErrCustomerNotFound
	}
	out := []models.PurchaseOrderResponse{}
	for _, p := range f.pos {
		out = append(out, *p)
	}
	return out, nil
}

func (f *fakePOs) Create(_ context.Context, customerID uuid.UUID, req *models.CreatePurchaseOrderRequest, _ uuid.UUID) (*models.PurchaseOrderResponse, error) {
	if customerID != f.customer {
		return nil, services.ErrCustomerNotFound
	}
	for _, p := range f.pos {
		if p.PONumber == req.PONumber {
			return nil, services.ErrPurchaseOrderExists
		}
	}
	p := &models.PurchaseOrderResponse{ID: uuid.New(), CustomerID: customerID, PONumber: req.PONumber, Amount: req.Amount, Remaining: req.Amount}
	f.pos[p.ID] = p
	return p, nil
}

func (f *fakePOs) Update(_ context.Context, customerID, poID uuid.UUID, req *models.UpdatePurchaseOrderRequest) (*models.PurchaseOrderResponse, error) {
	p, ok := f.pos[poID]
	if customerID != f.customer || !ok {
		return nil, services.ErrPurchaseOrderNotFound
	}
	if req.Amount != nil {
		p.Amount = *req.Amount
	}
	return p, nil
}

func (f *fakePOs) Delete(_ context.Context, customerID, poID uuid.UUID) error {
	if _, ok := f.pos[poID]; customerID != f.customer || !ok {
		return services.ErrPurchaseOrderNotFound
	}
	delete(f.pos, poID)
	return nil
}

func poRouter(f *fakePOs) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", uuid.New()) })
	h := NewPurchaseOrderHandler(f)
	r.GET("/customers/:id/purchase-orders", h.ListPurchaseOrders)
	r.POST("/customers/:id/purchase-orders", h.CreatePurchaseOrder)
	r.PUT("/customers/:id/purchase-orders/:poId", h.UpdatePurchaseOrder)
	r.DELETE("/customers/:id/purchase-orders/:poId", h.DeletePurchaseOrder)
	return r
}

func poDo(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func TestPurchaseOrderHandler(t *testing.T) {
	cust := uuid.New()
	f := &fakePOs{customer: cust, pos: map[uuid.UUID]*models.PurchaseOrderResponse{}}
	r := poRouter(f)
	base := "/customers/" + cust.String() + "/purchase-orders"

	w := poDo(r, http.MethodPost, base, `{"po_number":"PO-1","amount":"1200.00"}`)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var created models.PurchaseOrderResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	assert.Equal(t, "PO-1", created.PONumber)
	assert.Equal(t, "1200", created.Amount.String())

	w = poDo(r, http.MethodPost, base, `{"po_number":"PO-1","amount":"5"}`)
	assert.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "purchase_order_exists")

	w = poDo(r, http.MethodPost, base, `{"amount":"5"}`)
	assert.Equal(t, http.StatusBadRequest, w.Code, "po_number is required")

	w = poDo(r, http.MethodGet, base, "")
	require.Equal(t, http.StatusOK, w.Code)
	var list models.PurchaseOrderListResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &list))
	assert.Equal(t, 1, list.Total)

	w = poDo(r, http.MethodPut, base+"/"+created.ID.String(), `{"amount":"900"}`)
	require.Equal(t, http.StatusOK, w.Code)

	// Another customer (or tenant): 404; a bad ID: 400.
	w = poDo(r, http.MethodGet, "/customers/"+uuid.NewString()+"/purchase-orders", "")
	assert.Equal(t, http.StatusNotFound, w.Code)
	w = poDo(r, http.MethodPut, base+"/"+uuid.NewString(), `{"amount":"1"}`)
	assert.Equal(t, http.StatusNotFound, w.Code)
	w = poDo(r, http.MethodDelete, base+"/not-a-uuid", "")
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = poDo(r, http.MethodDelete, base+"/"+created.ID.String(), "")
	assert.Equal(t, http.StatusNoContent, w.Code)
}
