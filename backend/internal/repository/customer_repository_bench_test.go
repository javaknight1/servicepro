package repository

import (
	"fmt"
	"testing"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// setupBenchRepo seeds a fresh tenant (removed when the benchmark ends) and
// returns a customer repository scoped to it.
func setupBenchRepo(b *testing.B) CustomerRepositoryInterface {
	b.Helper()
	db := openTestDB(b)
	tt := createTestTenant(b, db)
	return NewCustomerRepository(db).WithContext(tt.Ctx)
}

// benchCustomer returns a valid customer whose email and phone derive from i.
func benchCustomer(email string, i int) *models.Customer {
	return &models.Customer{
		FirstName:            "Bench",
		LastName:             "Mark",
		Email:                email,
		PhonePrimary:         fmt.Sprintf("512-555-%04d", i%10000),
		BillingAddressStreet: "123 Test St",
		BillingAddressCity:   "Austin",
		BillingAddressState:  "TX",
		BillingAddressZip:    "78701",
		CustomerType:         models.CustomerTypeResidential,
		Status:               models.CustomerStatusActive,
	}
}

// seedBenchData creates sample customers in the repository's tenant
func seedBenchData(b *testing.B, repo CustomerRepositoryInterface, count int) {
	b.Helper()

	for i := 0; i < count; i++ {
		companyName := fmt.Sprintf("Company %d", i%10)
		customer := &models.Customer{
			FirstName:            fmt.Sprintf("First%d", i),
			LastName:             fmt.Sprintf("Last%d", i),
			Email:                fmt.Sprintf("bench%d@example.com", i),
			PhonePrimary:         fmt.Sprintf("512-555-%04d", i%10000),
			CompanyName:          &companyName,
			BillingAddressStreet: fmt.Sprintf("%d Main St", i),
			BillingAddressCity:   []string{"Austin", "Houston", "Dallas", "San Antonio"}[i%4],
			BillingAddressState:  "TX",
			BillingAddressZip:    fmt.Sprintf("78%03d", i%1000),
			CustomerType: []models.CustomerType{
				models.CustomerTypeResidential,
				models.CustomerTypeCommercial,
			}[i%2],
			Status: models.CustomerStatusActive,
		}

		if err := repo.Create(customer); err != nil {
			b.Fatalf("Failed to seed data: %v", err)
		}
	}
}

// BenchmarkFullTextSearch benchmarks the full-text search performance
func BenchmarkFullTextSearch(b *testing.B) {
	repo := setupBenchRepo(b)
	seedBenchData(b, repo, 1000)

	benchmarks := []struct {
		name      string
		query     string
		limit     int
		offset    int
		sortBy    string
		sortOrder string
	}{
		{
			name:      "Simple Query 10 Results",
			query:     "company",
			limit:     10,
			offset:    0,
			sortBy:    "rank",
			sortOrder: "DESC",
		},
		{
			name:      "Simple Query 50 Results",
			query:     "company",
			limit:     50,
			offset:    0,
			sortBy:    "rank",
			sortOrder: "DESC",
		},
		{
			name:      "Complex Query",
			query:     "company first last",
			limit:     20,
			offset:    0,
			sortBy:    "rank",
			sortOrder: "DESC",
		},
		{
			name:      "Paginated Results",
			query:     "company",
			limit:     20,
			offset:    100,
			sortBy:    "rank",
			sortOrder: "DESC",
		},
		{
			name:      "Sort By Name",
			query:     "company",
			limit:     20,
			offset:    0,
			sortBy:    "last_name",
			sortOrder: "ASC",
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _, err := repo.FullTextSearch(
					bm.query,
					"",
					"",
					"",
					"",
					bm.limit,
					bm.offset,
					bm.sortBy,
					bm.sortOrder,
				)
				if err != nil {
					b.Fatalf("Search failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkFullTextSearchWithFilters benchmarks search with various filters
func BenchmarkFullTextSearchWithFilters(b *testing.B) {
	repo := setupBenchRepo(b)
	seedBenchData(b, repo, 1000)

	benchmarks := []struct {
		name         string
		query        string
		customerType models.CustomerType
		status       models.CustomerStatus
		city         string
		state        string
	}{
		{
			name:         "Type Filter",
			query:        "company",
			customerType: models.CustomerTypeCommercial,
			status:       "",
			city:         "",
			state:        "",
		},
		{
			name:         "Status Filter",
			query:        "company",
			customerType: "",
			status:       models.CustomerStatusActive,
			city:         "",
			state:        "",
		},
		{
			name:         "City Filter",
			query:        "company",
			customerType: "",
			status:       "",
			city:         "Austin",
			state:        "",
		},
		{
			name:         "State Filter",
			query:        "company",
			customerType: "",
			status:       "",
			city:         "",
			state:        "TX",
		},
		{
			name:         "Multiple Filters",
			query:        "company",
			customerType: models.CustomerTypeCommercial,
			status:       models.CustomerStatusActive,
			city:         "Austin",
			state:        "TX",
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _, err := repo.FullTextSearch(
					bm.query,
					bm.customerType,
					bm.status,
					bm.city,
					bm.state,
					20,
					0,
					"rank",
					"DESC",
				)
				if err != nil {
					b.Fatalf("Search with filters failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkAutocomplete benchmarks the autocomplete performance
func BenchmarkAutocomplete(b *testing.B) {
	repo := setupBenchRepo(b)
	seedBenchData(b, repo, 1000)

	benchmarks := []struct {
		name   string
		prefix string
		limit  int
	}{
		{
			name:   "Short Prefix 10 Results",
			prefix: "fi",
			limit:  10,
		},
		{
			name:   "Short Prefix 50 Results",
			prefix: "fi",
			limit:  50,
		},
		{
			name:   "Medium Prefix",
			prefix: "first",
			limit:  10,
		},
		{
			name:   "Long Prefix",
			prefix: "first1",
			limit:  10,
		},
		{
			name:   "Company Search",
			prefix: "comp",
			limit:  10,
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := repo.Autocomplete(bm.prefix, bm.limit)
				if err != nil {
					b.Fatalf("Autocomplete failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkSearch compares basic search vs full-text search
func BenchmarkSearchComparison(b *testing.B) {
	repo := setupBenchRepo(b)
	seedBenchData(b, repo, 1000)

	query := "company"

	b.Run("Basic LIKE Search", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := repo.Search(query)
			if err != nil {
				b.Fatalf("Basic search failed: %v", err)
			}
		}
	})

	b.Run("Full-Text Search", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _, err := repo.FullTextSearch(query, "", "", "", "", 50, 0, "rank", "DESC")
			if err != nil {
				b.Fatalf("Full-text search failed: %v", err)
			}
		}
	})
}

// BenchmarkGetByID benchmarks ID lookup performance
func BenchmarkGetByID(b *testing.B) {
	repo := setupBenchRepo(b)

	customer := benchCustomer("benchmark@example.com", 0)
	if err := repo.Create(customer); err != nil {
		b.Fatalf("Failed to create test customer: %v", err)
	}

	id := customer.ID

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := repo.GetByID(id)
		if err != nil {
			b.Fatalf("GetByID failed: %v", err)
		}
	}
}

// BenchmarkGetByEmail benchmarks email lookup performance
func BenchmarkGetByEmail(b *testing.B) {
	repo := setupBenchRepo(b)

	if err := repo.Create(benchCustomer("benchmark@example.com", 0)); err != nil {
		b.Fatalf("Failed to create test customer: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := repo.GetByEmail("benchmark@example.com")
		if err != nil {
			b.Fatalf("GetByEmail failed: %v", err)
		}
	}
}

// BenchmarkList benchmarks paginated list performance
func BenchmarkList(b *testing.B) {
	repo := setupBenchRepo(b)
	seedBenchData(b, repo, 1000)

	benchmarks := []struct {
		name     string
		pageSize int
	}{
		{
			name:     "Page Size 10",
			pageSize: 10,
		},
		{
			name:     "Page Size 20",
			pageSize: 20,
		},
		{
			name:     "Page Size 50",
			pageSize: 50,
		},
		{
			name:     "Page Size 100",
			pageSize: 100,
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			filter := &models.CustomerListFilter{
				Page:      1,
				PageSize:  bm.pageSize,
				SortBy:    "created_at",
				SortOrder: "DESC",
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _, err := repo.List(filter)
				if err != nil {
					b.Fatalf("List failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkCreate benchmarks customer creation
func BenchmarkCreate(b *testing.B) {
	repo := setupBenchRepo(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		customer := benchCustomer(fmt.Sprintf("bench-create-%d@example.com", i), i)
		if err := repo.Create(customer); err != nil {
			b.Fatalf("Create failed: %v", err)
		}
	}
}

// BenchmarkUpdate benchmarks customer updates
func BenchmarkUpdate(b *testing.B) {
	repo := setupBenchRepo(b)

	customer := benchCustomer("bench-update@example.com", 0)
	if err := repo.Create(customer); err != nil {
		b.Fatalf("Failed to create test customer: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		customer.FirstName = fmt.Sprintf("Updated%d", i)
		if err := repo.Update(customer); err != nil {
			b.Fatalf("Update failed: %v", err)
		}
	}
}

// BenchmarkSearchStats benchmarks statistics retrieval
func BenchmarkSearchStats(b *testing.B) {
	repo := setupBenchRepo(b)
	seedBenchData(b, repo, 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := repo.GetSearchStats()
		if err != nil {
			b.Fatalf("GetSearchStats failed: %v", err)
		}
	}
}
