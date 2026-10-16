package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
)

func quoteServiceFor(f *isolationFixture) QuoteServiceInterface {
	return NewQuoteService(repository.NewQuoteRepository(f.db)).WithContext(f.ctxA)
}

func quoteRequest(customerID uuid.UUID) *models.QuoteRequest {
	return &models.QuoteRequest{
		CustomerID: customerID,
		ValidUntil: time.Now().Add(72 * time.Hour),
		Items: []models.QuoteItemRequest{
			{Description: "Labour", Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100)},
		},
	}
}

// T224 / audit 7.H6: saving a quote with its preloaded Customer reset
// customer_id, so changing a quote's customer silently reverted.
func TestUpdateQuote_KeepsCustomerChange(t *testing.T) {
	f := setupIsolation(t)
	other := &models.Customer{ID: uuid.New(), FirstName: "Carol", LastName: "A", Email: "carol@a.test", CustomerType: "residential", Status: "active"}
	require.NoError(t, repository.NewCustomerRepository(f.db).WithContext(f.ctxA).Create(other))
	svc := quoteServiceFor(f)

	created, err := svc.CreateQuote(quoteRequest(f.custA.ID), f.userA)
	require.NoError(t, err)

	updated, err := svc.UpdateQuote(created.ID, quoteRequest(other.ID), f.userA)
	require.NoError(t, err)
	assert.Equal(t, other.ID, updated.CustomerID)

	reloaded, err := svc.GetQuote(created.ID)
	require.NoError(t, err)
	assert.Equal(t, other.ID, reloaded.CustomerID, "customer change persisted")

	_, err = svc.UpdateQuote(created.ID, quoteRequest(f.custB.ID), f.userA)
	assert.Error(t, err, "can't move a quote to another tenant's customer")
}
