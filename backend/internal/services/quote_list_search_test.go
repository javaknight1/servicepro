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

// T228: the quotes list honours q, sort_by and sort_order, and search never
// reaches another tenant's quotes or customers.
func TestQuoteList_SearchAndSort(t *testing.T) {
	f := setupIsolation(t)
	custRepo := repository.NewCustomerRepository(f.db)
	repoA := repository.NewQuoteRepository(f.db).WithContext(f.ctxA)
	repoB := repository.NewQuoteRepository(f.db).WithContext(f.ctxB)

	acmeA := "Acme Plumbing"
	carol := &models.Customer{ID: uuid.New(), FirstName: "Carol", LastName: "Smith", Email: "carol@a.test", CompanyName: &acmeA, CustomerType: "commercial", Status: "active"}
	require.NoError(t, custRepo.WithContext(f.ctxA).Create(carol))
	acmeB := "Acme West"
	f.custB.CompanyName = &acmeB
	require.NoError(t, f.db.WithContext(f.ctxB).Save(f.custB).Error)

	mk := func(repo repository.QuoteRepositoryInterface, custID uuid.UUID, number string, total int64, age time.Duration) {
		t.Helper()
		require.NoError(t, repo.Create(&models.Quote{
			ID: uuid.New(), CustomerID: custID, QuoteNumber: number, Status: models.QuoteStatusDraft,
			ValidUntil: time.Now().Add(time.Hour), Total: decimal.NewFromInt(total), CreatedBy: f.userA,
		}))
		require.NoError(t, f.db.WithContext(f.ctxA).Exec(
			"UPDATE quotes SET created_at = ? WHERE quote_number = ?", time.Now().Add(-age), number).Error)
	}
	mk(repoA, f.custA.ID, "QUO-A-100", 300, 3*time.Hour)
	mk(repoA, carol.ID, "QUO-A-200", 100, 2*time.Hour)
	mk(repoA, carol.ID, "QUO-A-300", 200, 1*time.Hour)
	mk(repoB, f.custB.ID, "QUO-B-100", 999, 0)

	numbers := func(params models.QuoteQueryParams) []string {
		t.Helper()
		params.Page, params.PageSize = 1, 50
		quotes, total, err := repoA.List(&params)
		require.NoError(t, err)
		require.EqualValues(t, len(quotes), total)
		out := make([]string, len(quotes))
		for i, q := range quotes {
			out[i] = q.QuoteNumber
		}
		return out
	}

	t.Run("search", func(t *testing.T) {
		assert.Equal(t, []string{"QUO-A-300", "QUO-A-200"}, numbers(models.QuoteQueryParams{Q: "ACME"}), "company name, case-insensitive")
		assert.Equal(t, []string{"QUO-A-200"}, numbers(models.QuoteQueryParams{Q: "quo-a-2"}), "quote number")
		assert.Equal(t, []string{"QUO-A-100"}, numbers(models.QuoteQueryParams{Q: "alice"}), "first name")
		assert.Equal(t, []string{"QUO-A-300", "QUO-A-200"}, numbers(models.QuoteQueryParams{Q: "smith"}), "last name")
		assert.Equal(t, []string{"QUO-A-300", "QUO-A-200"}, numbers(models.QuoteQueryParams{Q: "carol smith"}), "full name")
		assert.Empty(t, numbers(models.QuoteQueryParams{Q: "%"}), "LIKE wildcards are literal")
		assert.Empty(t, numbers(models.QuoteQueryParams{Q: "QUO_A"}), "LIKE wildcards are literal")
		assert.Len(t, numbers(models.QuoteQueryParams{Q: "   "}), 3, "blank search lists everything")
	})

	t.Run("search never crosses tenants", func(t *testing.T) {
		assert.Empty(t, numbers(models.QuoteQueryParams{Q: "QUO-B"}))
		assert.Empty(t, numbers(models.QuoteQueryParams{Q: "bob"}))
		assert.Empty(t, numbers(models.QuoteQueryParams{Q: "west"}))

		quotes, total, err := repoB.List(&models.QuoteQueryParams{Q: "acme", Page: 1, PageSize: 50})
		require.NoError(t, err)
		assert.EqualValues(t, 1, total)
		require.Len(t, quotes, 1)
		assert.Equal(t, "QUO-B-100", quotes[0].QuoteNumber)
	})

	t.Run("sort", func(t *testing.T) {
		assert.Equal(t, []string{"QUO-A-300", "QUO-A-200", "QUO-A-100"}, numbers(models.QuoteQueryParams{}), "default created_at desc")
		assert.Equal(t, []string{"QUO-A-200", "QUO-A-300", "QUO-A-100"}, numbers(models.QuoteQueryParams{SortBy: "total", SortOrder: "asc"}))
		assert.Equal(t, []string{"QUO-A-100", "QUO-A-300", "QUO-A-200"}, numbers(models.QuoteQueryParams{SortBy: "total", SortOrder: "desc"}))
		assert.Equal(t, []string{"QUO-A-100", "QUO-A-200", "QUO-A-300"}, numbers(models.QuoteQueryParams{SortBy: "quote_number", SortOrder: "asc"}))
		assert.Equal(t, []string{"QUO-A-100", "QUO-A-200", "QUO-A-300"}, numbers(models.QuoteQueryParams{SortBy: "bogus; DROP TABLE quotes", SortOrder: "asc"}), "unknown column falls back to created_at")
	})
}
