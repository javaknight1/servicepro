package services

import (
	"sync"
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

// Audit 7.M6: accept and reject were check-then-update, so both could
// succeed. Now the second one loses.
func TestAcceptRejectQuote_OnlyOneWins(t *testing.T) {
	f := setupIsolation(t)
	svc := quoteServiceFor(f)
	created, err := svc.CreateQuote(quoteRequest(f.custA.ID), f.userA)
	require.NoError(t, err)

	assert.ErrorIs(t, svc.AcceptQuote(created.ID), ErrQuoteCannotBeAccepted, "drafts can't be accepted")
	require.NoError(t, svc.SendQuote(created.ID))
	require.NoError(t, svc.AcceptQuote(created.ID))
	assert.ErrorIs(t, svc.RejectQuote(created.ID), ErrQuoteCannotBeRejected)
	assert.ErrorIs(t, svc.AcceptQuote(created.ID), ErrQuoteCannotBeAccepted)

	got, err := svc.GetQuote(created.ID)
	require.NoError(t, err)
	assert.Equal(t, models.QuoteStatusAccepted, got.Status)

	assert.ErrorIs(t, NewQuoteService(repository.NewQuoteRepository(f.db)).WithContext(f.ctxB).AcceptQuote(created.ID), ErrQuoteNotFound)
}

// Concurrent accept/decline on Postgres: exactly one succeeds.
func TestPostgres_AcceptRejectQuote_Concurrent(t *testing.T) {
	db := setupPostgres(t)
	tn := seedPGTenant(t, db)
	svc := NewQuoteService(repository.NewQuoteRepository(db)).WithContext(tn.ctx)
	created, err := svc.CreateQuote(quoteRequest(tn.customerID), tn.userID)
	require.NoError(t, err)
	require.NoError(t, svc.SendQuote(created.ID))

	const workers = 8
	errs := make([]error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				errs[i] = svc.AcceptQuote(created.ID)
			} else {
				errs[i] = svc.RejectQuote(created.ID)
			}
		}(i)
	}
	wg.Wait()
	ok := 0
	for _, err := range errs {
		if err == nil {
			ok++
		}
	}
	assert.Equal(t, 1, ok)
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
