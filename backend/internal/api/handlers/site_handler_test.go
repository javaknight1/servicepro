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

// fakeSites is an in-memory SiteServiceInterface for one customer.
type fakeSites struct {
	customer uuid.UUID
	sites    map[uuid.UUID]*models.Site
}

func (f *fakeSites) check(customerID uuid.UUID) error {
	if customerID != f.customer {
		return services.ErrCustomerNotFound
	}
	return nil
}

func (f *fakeSites) List(_ context.Context, customerID uuid.UUID, includeInactive bool) ([]models.Site, error) {
	if err := f.check(customerID); err != nil {
		return nil, err
	}
	out := []models.Site{}
	for _, s := range f.sites {
		if s.IsActive || includeInactive {
			out = append(out, *s)
		}
	}
	return out, nil
}

func (f *fakeSites) Get(_ context.Context, customerID, siteID uuid.UUID) (*models.Site, error) {
	if err := f.check(customerID); err != nil {
		return nil, err
	}
	s, ok := f.sites[siteID]
	if !ok {
		return nil, services.ErrSiteNotFound
	}
	return s, nil
}

func (f *fakeSites) Create(_ context.Context, customerID uuid.UUID, req *models.CreateSiteRequest) (*models.Site, error) {
	if err := f.check(customerID); err != nil {
		return nil, err
	}
	s := &models.Site{ID: uuid.New(), CustomerID: customerID, Name: req.Name, AddressStreet: req.AddressStreet,
		AddressCity: req.AddressCity, AddressState: req.AddressState, AddressZip: req.AddressZip, IsActive: true}
	f.sites[s.ID] = s
	return s, nil
}

func (f *fakeSites) Update(ctx context.Context, customerID, siteID uuid.UUID, req *models.UpdateSiteRequest) (*models.Site, error) {
	s, err := f.Get(ctx, customerID, siteID)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		s.Name = *req.Name
	}
	if req.IsActive != nil {
		s.IsActive = *req.IsActive
	}
	return s, nil
}

func (f *fakeSites) Delete(ctx context.Context, customerID, siteID uuid.UUID) error {
	if _, err := f.Get(ctx, customerID, siteID); err != nil {
		return err
	}
	delete(f.sites, siteID)
	return nil
}

func siteRouter(f *fakeSites) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewSiteHandler(f)
	r.GET("/customers/:id/sites", h.ListSites)
	r.POST("/customers/:id/sites", h.CreateSite)
	r.GET("/customers/:id/sites/:siteId", h.GetSite)
	r.PUT("/customers/:id/sites/:siteId", h.UpdateSite)
	r.DELETE("/customers/:id/sites/:siteId", h.DeleteSite)
	return r
}

func doJSON(t *testing.T, r *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&buf).Encode(body))
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestSiteHandler_CRUD(t *testing.T) {
	f := &fakeSites{customer: uuid.New(), sites: map[uuid.UUID]*models.Site{}}
	r := siteRouter(f)
	base := "/customers/" + f.customer.String() + "/sites"

	w := doJSON(t, r, http.MethodPost, base, map[string]any{
		"name": "Harborview Lofts", "address_street": "220 W Spring St", "address_city": "Columbus",
		"address_state": "OH", "address_zip": "43215",
	})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var created models.SiteResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	assert.Equal(t, "220 W Spring St, Columbus, OH 43215", created.AddressFull)

	w = doJSON(t, r, http.MethodGet, base, nil)
	require.Equal(t, http.StatusOK, w.Code)
	var list models.SiteListResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &list))
	assert.Equal(t, 1, list.Total)

	w = doJSON(t, r, http.MethodPut, base+"/"+created.ID.String(), map[string]any{"name": "Lofts"})
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"name":"Lofts"`)

	w = doJSON(t, r, http.MethodGet, base+"/"+created.ID.String(), nil)
	require.Equal(t, http.StatusOK, w.Code)

	w = doJSON(t, r, http.MethodDelete, base+"/"+created.ID.String(), nil)
	require.Equal(t, http.StatusNoContent, w.Code)

	w = doJSON(t, r, http.MethodGet, base+"/"+created.ID.String(), nil)
	require.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"site_not_found"`)
}

func TestSiteHandler_Errors(t *testing.T) {
	f := &fakeSites{customer: uuid.New(), sites: map[uuid.UUID]*models.Site{}}
	r := siteRouter(f)

	w := doJSON(t, r, http.MethodGet, "/customers/not-a-uuid/sites", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	w = doJSON(t, r, http.MethodGet, "/customers/"+uuid.NewString()+"/sites", nil)
	assert.Equal(t, http.StatusNotFound, w.Code, "another tenant's (or unknown) customer")
	assert.Contains(t, w.Body.String(), `"error":"customer_not_found"`)

	w = doJSON(t, r, http.MethodPost, "/customers/"+f.customer.String()+"/sites", map[string]any{"name": "No address"})
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "address_street")
}
