package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services"
)

func TestRunAgreementVisits(t *testing.T) {
	gin.SetMode(gin.TestMode)
	jobs := NewInternalJobsHandler("s3cret", nil)
	r := gin.New()
	r.POST("/internal/jobs/agreement-visits", jobs.Authorize, RunAgreementVisits(func(context.Context, time.Time) (*services.AgreementVisitsResult, error) {
		return &services.AgreementVisitsResult{Agreements: 2, Created: 9}, nil
	}))
	r.POST("/internal/jobs/broken", jobs.Authorize, RunAgreementVisits(func(context.Context, time.Time) (*services.AgreementVisitsResult, error) {
		return &services.AgreementVisitsResult{Failed: 1}, errors.New("db down")
	}))
	do := func(path, token string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, path, nil)
		if token != "" {
			req.Header.Set(InternalJobsTokenHeader, token)
		}
		r.ServeHTTP(w, req)
		return w
	}

	assert.Equal(t, http.StatusUnauthorized, do("/internal/jobs/agreement-visits", "").Code)
	w := do("/internal/jobs/agreement-visits", "s3cret")
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"created":9`)
	assert.Contains(t, w.Body.String(), `"job":"agreement-visits"`)
	w = do("/internal/jobs/broken", "s3cret")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.False(t, strings.Contains(w.Body.String(), "db down"), "no internals leak")
}

// fakeAgreements answers every lookup with not-found except one ID.
type fakeAgreements struct {
	known      uuid.UUID
	lastDays   int
	lastFilter *models.AgreementListFilter
}

func (f *fakeAgreements) get(id uuid.UUID) (*models.ServiceAgreement, error) {
	if id != f.known {
		return nil, services.ErrAgreementNotFound
	}
	return &models.ServiceAgreement{ID: id, Title: "Nightly", Frequency: models.FrequencyWeekdays,
		StartDate: time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC), VisitStartMinutes: 18 * 60, Status: models.AgreementStatusActive}, nil
}

func (f *fakeAgreements) List(_ context.Context, filter *models.AgreementListFilter) ([]models.ServiceAgreement, error) {
	f.lastFilter = filter
	a, _ := f.get(f.known)
	return []models.ServiceAgreement{*a}, nil
}
func (f *fakeAgreements) Get(_ context.Context, id uuid.UUID) (*models.ServiceAgreement, error) {
	return f.get(id)
}
func (f *fakeAgreements) Create(context.Context, *models.CreateAgreementRequest, uuid.UUID) (*models.ServiceAgreement, error) {
	return f.get(f.known)
}
func (f *fakeAgreements) Update(_ context.Context, id uuid.UUID, _ *models.UpdateAgreementRequest) (*models.ServiceAgreement, error) {
	return f.get(id)
}
func (f *fakeAgreements) SetStatus(_ context.Context, id uuid.UUID, s models.AgreementStatus) (*models.ServiceAgreement, error) {
	a, err := f.get(id)
	if a != nil {
		a.Status = s
	}
	return a, err
}
func (f *fakeAgreements) Delete(_ context.Context, id uuid.UUID) error {
	_, err := f.get(id)
	return err
}
func (f *fakeAgreements) Preview(_ context.Context, id uuid.UUID, days int) (*models.AgreementPreviewResponse, error) {
	f.lastDays = days
	if _, err := f.get(id); err != nil {
		return nil, err
	}
	return &models.AgreementPreviewResponse{AgreementID: id, Timezone: "UTC", Visits: []models.AgreementVisitPreview{}}, nil
}

func TestAgreementHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	f := &fakeAgreements{known: uuid.New()}
	h := NewAgreementHandler(f)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", uuid.New()) })
	r.GET("/agreements", h.ListAgreements)
	r.POST("/agreements", h.CreateAgreement)
	r.GET("/agreements/:id", h.GetAgreement)
	r.DELETE("/agreements/:id", h.DeleteAgreement)
	r.POST("/agreements/:id/pause", h.PauseAgreement)
	r.GET("/agreements/:id/preview", h.PreviewAgreement)
	known := "/agreements/" + f.known.String()

	w := doJSON(t, r, http.MethodGet, "/agreements", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"visit_start_time":"18:00"`)
	assert.Contains(t, w.Body.String(), `"start_date":"2026-11-02"`)

	// Filters: IDs in the query used to fail Gin's binding (400).
	customerID, siteID := uuid.New(), uuid.New()
	w = doJSON(t, r, http.MethodGet, "/agreements?customer_id="+customerID.String()+"&site_id="+siteID.String()+"&status=paused", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotNil(t, f.lastFilter.CustomerID)
	assert.Equal(t, customerID, *f.lastFilter.CustomerID)
	assert.Equal(t, siteID, *f.lastFilter.SiteID)
	assert.Equal(t, models.AgreementStatusPaused, *f.lastFilter.Status)
	w = doJSON(t, r, http.MethodGet, "/agreements?customer_id=nope&status=gone", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"customer_id"`)
	assert.Contains(t, w.Body.String(), `"status"`)

	w = doJSON(t, r, http.MethodPost, "/agreements", map[string]any{"title": "x"})
	assert.Equal(t, http.StatusBadRequest, w.Code, "missing required fields")

	w = doJSON(t, r, http.MethodPost, known+"/pause", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"status":"paused"`)

	w = doJSON(t, r, http.MethodGet, "/agreements/"+uuid.NewString(), nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"agreement_not_found"`)

	w = doJSON(t, r, http.MethodGet, known+"/preview?days=30", nil)
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 30, f.lastDays)
	w = doJSON(t, r, http.MethodGet, known+"/preview?days=abc", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)

	assert.Equal(t, http.StatusNoContent, doJSON(t, r, http.MethodDelete, known, nil).Code)
}
