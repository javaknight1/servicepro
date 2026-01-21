package search

import (
	"fmt"
	"strings"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// SearchParams holds all search parameters
type SearchParams struct {
	Query  string                // Search query
	Type   models.CustomerType   // Filter by customer type
	Status models.CustomerStatus // Filter by status
	City   string                // Filter by city
	State  string                // Filter by state
	Page   int                   // Page number (1-indexed)
	Limit  int                   // Results per page
	Sort   string                // Sort column
	Order  string                // Sort order (ASC/DESC)
}

// SearchResult holds search results with metadata
type SearchResult struct {
	Customers  []CustomerSearchResult `json:"customers"`
	Total      int64                  `json:"total"`
	Page       int                    `json:"page"`
	Limit      int                    `json:"limit"`
	TotalPages int                    `json:"total_pages"`
	Query      string                 `json:"query"`
}

// CustomerSearchResult holds individual search result with ranking
type CustomerSearchResult struct {
	models.CustomerResponse
	SearchRank float32 `json:"search_rank"`
}

// AutocompleteResult holds autocomplete suggestions
type AutocompleteResult struct {
	ID          string                `json:"id"`
	DisplayName string                `json:"display_name"`
	Email       string                `json:"email"`
	Type        models.CustomerType   `json:"customer_type"`
	Status      models.CustomerStatus `json:"status"`
}

// Validate validates and normalizes search parameters
func (sp *SearchParams) Validate() error {
	// Validate and set defaults for page
	if sp.Page < 1 {
		sp.Page = 1
	}

	// Validate and set defaults for limit
	if sp.Limit < 1 {
		sp.Limit = 20
	}
	if sp.Limit > 100 {
		sp.Limit = 100
	}

	// Validate sort column
	validSortColumns := map[string]bool{
		"rank":         true,
		"first_name":   true,
		"last_name":    true,
		"company_name": true,
		"email":        true,
		"created_at":   true,
		"updated_at":   true,
	}

	if sp.Sort == "" {
		sp.Sort = "rank"
	} else if !validSortColumns[sp.Sort] {
		return fmt.Errorf("invalid sort column: %s", sp.Sort)
	}

	// Validate and normalize sort order
	sp.Order = strings.ToUpper(sp.Order)
	if sp.Order != "ASC" && sp.Order != "DESC" {
		sp.Order = "DESC"
	}

	// Normalize state to uppercase
	if sp.State != "" {
		sp.State = strings.ToUpper(sp.State)
		if len(sp.State) != 2 {
			return fmt.Errorf("state must be 2-letter code")
		}
	}

	// Trim and validate query
	sp.Query = strings.TrimSpace(sp.Query)
	if sp.Query == "" && sp.Type == "" && sp.Status == "" && sp.City == "" && sp.State == "" {
		return fmt.Errorf("at least one search criterion is required")
	}

	return nil
}

// GetOffset calculates the offset for pagination
func (sp *SearchParams) GetOffset() int {
	return (sp.Page - 1) * sp.Limit
}

// BuildSearchQuery sanitizes and builds a search query for PostgreSQL full-text search
func BuildSearchQuery(query string) string {
	// Remove special characters that could cause issues
	query = strings.TrimSpace(query)

	// Replace multiple spaces with single space
	query = strings.Join(strings.Fields(query), " ")

	// Escape single quotes
	query = strings.ReplaceAll(query, "'", "''")

	return query
}

// HighlightMatches adds highlighting to search matches (for frontend display)
// This is a helper function that can be used to highlight search terms
func HighlightMatches(text, query string, highlightStart, highlightEnd string) string {
	if query == "" || text == "" {
		return text
	}

	// Simple case-insensitive highlighting
	words := strings.Fields(query)
	result := text

	for _, word := range words {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}

		// Case-insensitive replace
		re := strings.NewReplacer(
			word, highlightStart+word+highlightEnd,
			strings.ToLower(word), highlightStart+strings.ToLower(word)+highlightEnd,
			strings.ToUpper(word), highlightStart+strings.ToUpper(word)+highlightEnd,
			strings.Title(word), highlightStart+strings.Title(word)+highlightEnd,
		)
		result = re.Replace(result)
	}

	return result
}

// CalculateTotalPages calculates the total number of pages
func CalculateTotalPages(total int64, limit int) int {
	if limit == 0 {
		return 0
	}
	pages := int(total) / limit
	if int(total)%limit > 0 {
		pages++
	}
	return pages
}

// SuggestSearchTerms suggests alternative search terms (for "did you mean?")
func SuggestSearchTerms(query string) []string {
	// This is a simple implementation
	// In production, you might want to use a more sophisticated spell-checker
	suggestions := []string{}

	// Remove extra spaces
	cleaned := strings.Join(strings.Fields(query), " ")
	if cleaned != query {
		suggestions = append(suggestions, cleaned)
	}

	return suggestions
}

// ParseSearchQuery parses advanced search syntax
// Supports: "exact phrase", term1 AND term2, term1 OR term2
func ParseSearchQuery(query string) (string, error) {
	// This is a basic implementation
	// PostgreSQL's plainto_tsquery handles most of this automatically

	// Remove leading/trailing spaces
	query = strings.TrimSpace(query)

	if query == "" {
		return "", fmt.Errorf("empty search query")
	}

	// For now, just return the sanitized query
	// The PostgreSQL function will handle the tsquery conversion
	return BuildSearchQuery(query), nil
}

// ExtractSearchFilters extracts filter values from search query
// Example: "john type:commercial status:active" -> extracts filters
func ExtractSearchFilters(query string) (cleanQuery string, filters map[string]string) {
	filters = make(map[string]string)
	words := strings.Fields(query)
	cleanWords := []string{}

	for _, word := range words {
		// Check for filter syntax: key:value
		if strings.Contains(word, ":") {
			parts := strings.SplitN(word, ":", 2)
			if len(parts) == 2 {
				key := strings.ToLower(parts[0])
				value := parts[1]

				switch key {
				case "type", "customer_type":
					filters["type"] = value
				case "status":
					filters["status"] = value
				case "city":
					filters["city"] = value
				case "state":
					filters["state"] = value
				default:
					// Not a recognized filter, keep in query
					cleanWords = append(cleanWords, word)
				}
			} else {
				cleanWords = append(cleanWords, word)
			}
		} else {
			cleanWords = append(cleanWords, word)
		}
	}

	cleanQuery = strings.Join(cleanWords, " ")
	return cleanQuery, filters
}

// RankSearchResults applies custom ranking to search results
// This can be used to boost certain types of matches
func RankSearchResults(results []CustomerSearchResult, boostFactors map[string]float32) []CustomerSearchResult {
	for i := range results {
		rank := results[i].SearchRank

		// Apply boost factors
		if boostFactors["active_customers"] > 0 && results[i].Status == models.CustomerStatusActive {
			rank *= boostFactors["active_customers"]
		}

		if boostFactors["commercial"] > 0 && results[i].CustomerType == models.CustomerTypeCommercial {
			rank *= boostFactors["commercial"]
		}

		results[i].SearchRank = rank
	}

	// Results are already sorted by rank from database
	return results
}

// FormatSearchQuery formats a user query for display
func FormatSearchQuery(params SearchParams) string {
	parts := []string{}

	if params.Query != "" {
		parts = append(parts, fmt.Sprintf("Query: \"%s\"", params.Query))
	}

	if params.Type != "" {
		parts = append(parts, fmt.Sprintf("Type: %s", params.Type))
	}

	if params.Status != "" {
		parts = append(parts, fmt.Sprintf("Status: %s", params.Status))
	}

	if params.City != "" {
		parts = append(parts, fmt.Sprintf("City: %s", params.City))
	}

	if params.State != "" {
		parts = append(parts, fmt.Sprintf("State: %s", params.State))
	}

	return strings.Join(parts, ", ")
}
