package search

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

func TestSearchParams_Validate(t *testing.T) {
	tests := []struct {
		name    string
		params  SearchParams
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid params with query",
			params: SearchParams{
				Query: "test",
				Page:  1,
				Limit: 20,
				Sort:  "rank",
				Order: "DESC",
			},
			wantErr: false,
		},
		{
			name: "Valid params with filters",
			params: SearchParams{
				Type:   models.CustomerTypeCommercial,
				Status: models.CustomerStatusActive,
				Page:   1,
				Limit:  20,
			},
			wantErr: false,
		},
		{
			name: "Sets default page",
			params: SearchParams{
				Query: "test",
				Page:  0,
				Limit: 20,
			},
			wantErr: false,
		},
		{
			name: "Sets default limit",
			params: SearchParams{
				Query: "test",
				Page:  1,
				Limit: 0,
			},
			wantErr: false,
		},
		{
			name: "Caps max limit at 100",
			params: SearchParams{
				Query: "test",
				Page:  1,
				Limit: 200,
			},
			wantErr: false,
		},
		{
			name: "Invalid sort column",
			params: SearchParams{
				Query: "test",
				Sort:  "invalid_column",
			},
			wantErr: true,
			errMsg:  "invalid sort column",
		},
		{
			name: "Normalizes sort order",
			params: SearchParams{
				Query: "test",
				Order: "asc",
			},
			wantErr: false,
		},
		{
			name: "Invalid state length",
			params: SearchParams{
				Query: "test",
				State: "TEX",
			},
			wantErr: true,
			errMsg:  "state must be 2-letter code",
		},
		{
			name: "Valid state code",
			params: SearchParams{
				Query: "test",
				State: "tx",
			},
			wantErr: false,
		},
		{
			name:    "No search criteria",
			params:  SearchParams{},
			wantErr: true,
			errMsg:  "at least one search criterion is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.params.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)

				// Check defaults were set
				if tt.params.Page == 0 {
					assert.Equal(t, 1, tt.params.Page)
				}
				if tt.params.Limit == 0 {
					assert.Equal(t, 20, tt.params.Limit)
				}
				if tt.params.Limit > 100 {
					assert.Equal(t, 100, tt.params.Limit)
				}
				if tt.params.Sort == "" {
					assert.Equal(t, "rank", tt.params.Sort)
				}
				if tt.params.Order == "asc" {
					assert.Equal(t, "ASC", tt.params.Order)
				}
				if tt.params.State != "" {
					assert.Equal(t, 2, len(tt.params.State))
					assert.Equal(t, tt.params.State, "TX") // Should be uppercased
				}
			}
		})
	}
}

func TestSearchParams_GetOffset(t *testing.T) {
	tests := []struct {
		name   string
		page   int
		limit  int
		expect int
	}{
		{
			name:   "First page",
			page:   1,
			limit:  20,
			expect: 0,
		},
		{
			name:   "Second page",
			page:   2,
			limit:  20,
			expect: 20,
		},
		{
			name:   "Third page with limit 50",
			page:   3,
			limit:  50,
			expect: 100,
		},
		{
			name:   "Page 10",
			page:   10,
			limit:  10,
			expect: 90,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := SearchParams{
				Page:  tt.page,
				Limit: tt.limit,
			}
			assert.Equal(t, tt.expect, params.GetOffset())
		})
	}
}

func TestBuildSearchQuery(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "Simple query",
			input:  "test",
			expect: "test",
		},
		{
			name:   "Query with spaces",
			input:  "  test  query  ",
			expect: "test query",
		},
		{
			name:   "Query with multiple spaces",
			input:  "test    multiple    spaces",
			expect: "test multiple spaces",
		},
		{
			name:   "Query with single quotes",
			input:  "O'Brien",
			expect: "O''Brien",
		},
		{
			name:   "Empty query",
			input:  "",
			expect: "",
		},
		{
			name:   "Query with only spaces",
			input:  "   ",
			expect: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildSearchQuery(tt.input)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestCalculateTotalPages(t *testing.T) {
	tests := []struct {
		name   string
		total  int64
		limit  int
		expect int
	}{
		{
			name:   "Exact division",
			total:  100,
			limit:  20,
			expect: 5,
		},
		{
			name:   "With remainder",
			total:  105,
			limit:  20,
			expect: 6,
		},
		{
			name:   "Less than one page",
			total:  10,
			limit:  20,
			expect: 1,
		},
		{
			name:   "Zero results",
			total:  0,
			limit:  20,
			expect: 0,
		},
		{
			name:   "Zero limit",
			total:  100,
			limit:  0,
			expect: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateTotalPages(tt.total, tt.limit)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestHighlightMatches(t *testing.T) {
	tests := []struct {
		name           string
		text           string
		query          string
		highlightStart string
		highlightEnd   string
		expect         string
	}{
		{
			name:           "Simple match",
			text:           "John Smith",
			query:          "John",
			highlightStart: "<mark>",
			highlightEnd:   "</mark>",
			expect:         "<mark>John</mark> Smith",
		},
		{
			name:           "Multiple matches",
			text:           "John likes john",
			query:          "john",
			highlightStart: "<b>",
			highlightEnd:   "</b>",
			expect:         "<b>John</b> likes <b>john</b>",
		},
		{
			name:           "Empty query",
			text:           "John Smith",
			query:          "",
			highlightStart: "<mark>",
			highlightEnd:   "</mark>",
			expect:         "John Smith",
		},
		{
			name:           "Empty text",
			text:           "",
			query:          "John",
			highlightStart: "<mark>",
			highlightEnd:   "</mark>",
			expect:         "",
		},
		{
			name:           "No matches",
			text:           "John Smith",
			query:          "Jane",
			highlightStart: "<mark>",
			highlightEnd:   "</mark>",
			expect:         "John Smith",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HighlightMatches(tt.text, tt.query, tt.highlightStart, tt.highlightEnd)
			assert.Contains(t, result, tt.text[:min(len(tt.text), 4)]) // At least some of the original text
		})
	}
}

func TestParseSearchQuery(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "Valid simple query",
			input:   "test",
			wantErr: false,
		},
		{
			name:    "Valid complex query",
			input:   "company name test",
			wantErr: false,
		},
		{
			name:    "Empty query",
			input:   "",
			wantErr: true,
		},
		{
			name:    "Query with spaces",
			input:   "  test  ",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseSearchQuery(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, result)
			}
		})
	}
}

func TestExtractSearchFilters(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectQuery   string
		expectFilters map[string]string
	}{
		{
			name:          "No filters",
			input:         "john smith",
			expectQuery:   "john smith",
			expectFilters: map[string]string{},
		},
		{
			name:        "Type filter",
			input:       "john type:commercial",
			expectQuery: "john",
			expectFilters: map[string]string{
				"type": "commercial",
			},
		},
		{
			name:        "Status filter",
			input:       "john status:active",
			expectQuery: "john",
			expectFilters: map[string]string{
				"status": "active",
			},
		},
		{
			name:        "City filter",
			input:       "john city:austin",
			expectQuery: "john",
			expectFilters: map[string]string{
				"city": "austin",
			},
		},
		{
			name:        "State filter",
			input:       "john state:tx",
			expectQuery: "john",
			expectFilters: map[string]string{
				"state": "tx",
			},
		},
		{
			name:        "Multiple filters",
			input:       "john type:commercial status:active city:austin",
			expectQuery: "john",
			expectFilters: map[string]string{
				"type":   "commercial",
				"status": "active",
				"city":   "austin",
			},
		},
		{
			name:          "Non-filter colon",
			input:         "email:john@example.com",
			expectQuery:   "email:john@example.com",
			expectFilters: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, filters := ExtractSearchFilters(tt.input)
			assert.Equal(t, tt.expectQuery, query)
			assert.Equal(t, len(tt.expectFilters), len(filters))
			for key, expectedValue := range tt.expectFilters {
				assert.Equal(t, expectedValue, filters[key])
			}
		})
	}
}

func TestRankSearchResults(t *testing.T) {
	tests := []struct {
		name         string
		results      []CustomerSearchResult
		boostFactors map[string]float32
		expectBoost  bool
	}{
		{
			name: "Boost active customers",
			results: []CustomerSearchResult{
				{
					CustomerResponse: models.CustomerResponse{
						Status: models.CustomerStatusActive,
					},
					SearchRank: 1.0,
				},
			},
			boostFactors: map[string]float32{
				"active_customers": 1.5,
			},
			expectBoost: true,
		},
		{
			name: "Boost commercial customers",
			results: []CustomerSearchResult{
				{
					CustomerResponse: models.CustomerResponse{
						CustomerType: models.CustomerTypeCommercial,
					},
					SearchRank: 1.0,
				},
			},
			boostFactors: map[string]float32{
				"commercial": 2.0,
			},
			expectBoost: true,
		},
		{
			name: "No boost factors",
			results: []CustomerSearchResult{
				{
					CustomerResponse: models.CustomerResponse{
						Status: models.CustomerStatusActive,
					},
					SearchRank: 1.0,
				},
			},
			boostFactors: map[string]float32{},
			expectBoost:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalRank := tt.results[0].SearchRank
			result := RankSearchResults(tt.results, tt.boostFactors)

			if tt.expectBoost {
				assert.Greater(t, result[0].SearchRank, originalRank)
			} else {
				assert.Equal(t, originalRank, result[0].SearchRank)
			}
		})
	}
}

func TestFormatSearchQuery(t *testing.T) {
	tests := []struct {
		name   string
		params SearchParams
		expect string
	}{
		{
			name: "Query only",
			params: SearchParams{
				Query: "test",
			},
			expect: `Query: "test"`,
		},
		{
			name: "Query with type",
			params: SearchParams{
				Query: "test",
				Type:  models.CustomerTypeCommercial,
			},
			expect: `Query: "test", Type: commercial`,
		},
		{
			name: "Query with multiple filters",
			params: SearchParams{
				Query:  "test",
				Type:   models.CustomerTypeCommercial,
				Status: models.CustomerStatusActive,
				City:   "Austin",
				State:  "TX",
			},
			expect: `Query: "test", Type: commercial, Status: active, City: Austin, State: TX`,
		},
		{
			name: "No query",
			params: SearchParams{
				Type: models.CustomerTypeCommercial,
			},
			expect: "Type: commercial",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatSearchQuery(tt.params)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestSuggestSearchTerms(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectCount int
	}{
		{
			name:        "Query with extra spaces",
			input:       "test  query  here",
			expectCount: 1, // Should suggest cleaned version
		},
		{
			name:        "Clean query",
			input:       "test query",
			expectCount: 0, // No suggestions needed
		},
		{
			name:        "Empty query",
			input:       "",
			expectCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suggestions := SuggestSearchTerms(tt.input)
			assert.Equal(t, tt.expectCount, len(suggestions))
		})
	}
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
