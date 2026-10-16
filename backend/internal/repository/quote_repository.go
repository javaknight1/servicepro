package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/utils"
)

// QuoteRepository implements QuoteRepositoryInterface
// Interface definition is in interface.go
type QuoteRepository struct {
	db *gorm.DB
}

// NewQuoteRepository creates a new quote repository
func NewQuoteRepository(db *gorm.DB) *QuoteRepository {
	return &QuoteRepository{db: db}
}

// WithContext returns a copy of the repository bound to ctx (tenant scoping
// via internal/tenancy reads the tenant from it).
func (r *QuoteRepository) WithContext(ctx context.Context) QuoteRepositoryInterface {
	return &QuoteRepository{db: r.db.WithContext(ctx)}
}

// Create creates a new quote
func (r *QuoteRepository) Create(quote *models.Quote) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// The customer must belong to the caller's tenant (scoped lookup).
		if err := requireCustomerInTenant(tx, quote.CustomerID); err != nil {
			return err
		}

		// Set QuoteID on items before creation (GORM will auto-create associations)
		for i := range quote.Items {
			quote.Items[i].QuoteID = quote.ID
		}

		// Create quote with items (GORM handles association creation automatically)
		if err := tx.Create(quote).Error; err != nil {
			return fmt.Errorf("failed to create quote: %w", err)
		}

		return nil
	})
}

// GetByID retrieves a quote by ID with all relationships
func (r *QuoteRepository) GetByID(id uuid.UUID) (*models.Quote, error) {
	var quote models.Quote
	err := r.db.Preload("Customer").
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC")
		}).
		Preload("CreatedByUser").
		Preload("UpdatedByUser").
		First(&quote, "id = ?", id).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("quote not found")
		}
		return nil, err
	}

	return &quote, nil
}

// GetByQuoteNumber retrieves a quote by quote number
func (r *QuoteRepository) GetByQuoteNumber(quoteNumber string) (*models.Quote, error) {
	var quote models.Quote
	err := r.db.Preload("Customer").
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC")
		}).
		First(&quote, "quote_number = ?", quoteNumber).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("quote not found")
		}
		return nil, err
	}

	return &quote, nil
}

// Update updates a quote
func (r *QuoteRepository) Update(quote *models.Quote) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// A changed customer must belong to the caller's tenant too.
		if err := requireCustomerInTenant(tx, quote.CustomerID); err != nil {
			return err
		}

		// Omit associations: saving with the preloaded Customer /
		// CreatedByUser / UpdatedByUser made GORM reset customer_id and
		// updated_by from the stale structs, silently reverting a customer
		// change (audit 7.H6). Items are saved explicitly below.
		if err := tx.Omit(clause.Associations).Save(quote).Error; err != nil {
			return fmt.Errorf("failed to update quote: %w", err)
		}

		// Update items if provided
		if len(quote.Items) > 0 {
			// Get existing items
			var existingItems []models.QuoteItem
			if err := tx.Where("quote_id = ?", quote.ID).Find(&existingItems).Error; err != nil {
				return fmt.Errorf("failed to get existing items: %w", err)
			}

			// Create a map of existing item IDs
			existingIDs := make(map[uuid.UUID]bool)
			for _, item := range existingItems {
				existingIDs[item.ID] = true
			}

			// Update or create items
			newIDs := make(map[uuid.UUID]bool)
			for i := range quote.Items {
				quote.Items[i].QuoteID = quote.ID

				// Only items already on THIS quote may be updated by ID; an
				// unknown ID (e.g. another quote's item) is created fresh.
				if quote.Items[i].ID != uuid.Nil && !existingIDs[quote.Items[i].ID] {
					quote.Items[i].ID = uuid.Nil
				}
				if quote.Items[i].ID != uuid.Nil {
					// Update existing item
					if err := tx.Save(&quote.Items[i]).Error; err != nil {
						return fmt.Errorf("failed to update quote item: %w", err)
					}
					newIDs[quote.Items[i].ID] = true
				} else {
					// Create new item
					if err := tx.Create(&quote.Items[i]).Error; err != nil {
						return fmt.Errorf("failed to create quote item: %w", err)
					}
					newIDs[quote.Items[i].ID] = true
				}
			}

			// Delete items that are no longer in the list
			for id := range existingIDs {
				if !newIDs[id] {
					if err := tx.Delete(&models.QuoteItem{}, "id = ?", id).Error; err != nil {
						return fmt.Errorf("failed to delete quote item: %w", err)
					}
				}
			}
		}

		return nil
	})
}

// Delete soft deletes a quote
func (r *QuoteRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Quote{}, "id = ?", id).Error
}

// likeEscaper escapes LIKE wildcards in user search input (used with ESCAPE '\').
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// List retrieves quotes with pagination, filters, search (Q) and sorting.
func (r *QuoteRepository) List(params *models.QuoteQueryParams) ([]models.Quote, int64, error) {
	var quotes []models.Quote
	var total int64

	query := r.db.Model(&models.Quote{})

	// Apply filters
	if params.CustomerID != nil {
		query = query.Where("customer_id = ?", params.CustomerID)
	}
	if params.Status != nil {
		query = query.Where("status = ?", params.Status)
	}
	if q := strings.TrimSpace(params.Q); q != "" {
		pattern := "%" + likeEscaper.Replace(strings.ToLower(q)) + "%"
		// The customer subquery is raw SQL, so the tenancy plugin doesn't
		// scope it: tie it to the quote's own (plugin-scoped) tenant_id.
		query = query.Where(
			`LOWER(quotes.quote_number) LIKE ? ESCAPE '\' OR quotes.customer_id IN (
				SELECT c.id FROM customers c
				WHERE c.tenant_id = quotes.tenant_id AND c.deleted_at IS NULL AND (
					LOWER(c.first_name) LIKE ? ESCAPE '\' OR
					LOWER(c.last_name) LIKE ? ESCAPE '\' OR
					LOWER(c.company_name) LIKE ? ESCAPE '\' OR
					LOWER(c.first_name || ' ' || c.last_name) LIKE ? ESCAPE '\'))`,
			pattern, pattern, pattern, pattern, pattern,
		)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	page := params.Page
	if page < 1 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize < 1 {
		pageSize = 50
	}

	offset := (page - 1) * pageSize

	// Fetch quotes
	err := query.Preload("Customer").
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC")
		}).
		Order("quotes." + utils.SafeOrderClause(params.SortBy, params.SortOrder, utils.QuoteAllowedSortColumns, "created_at")).
		Order("quotes.id").
		Limit(pageSize).
		Offset(offset).
		Find(&quotes).Error

	return quotes, total, err
}

// GetByCustomer retrieves all quotes for a customer
func (r *QuoteRepository) GetByCustomer(customerID uuid.UUID) ([]models.Quote, error) {
	var quotes []models.Quote
	err := r.db.Where("customer_id = ?", customerID).
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC")
		}).
		Order("created_at DESC").
		Find(&quotes).Error

	return quotes, err
}

// GetByStatus retrieves all quotes with a specific status
func (r *QuoteRepository) GetByStatus(status models.QuoteStatus) ([]models.Quote, error) {
	var quotes []models.Quote
	err := r.db.Where("status = ?", status).
		Preload("Customer").
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC")
		}).
		Order("created_at DESC").
		Find(&quotes).Error

	return quotes, err
}

// CreateItem creates a new quote item
func (r *QuoteRepository) CreateItem(item *models.QuoteItem) error {
	return r.db.Create(item).Error
}

// GetItemsByQuote retrieves all items for a quote
func (r *QuoteRepository) GetItemsByQuote(quoteID uuid.UUID) ([]models.QuoteItem, error) {
	var items []models.QuoteItem
	err := r.db.Where("quote_id = ?", quoteID).
		Order("sort_order ASC").
		Find(&items).Error

	return items, err
}

// UpdateItem updates a quote item
func (r *QuoteRepository) UpdateItem(item *models.QuoteItem) error {
	return r.db.Save(item).Error
}

// DeleteItem soft deletes a quote item
func (r *QuoteRepository) DeleteItem(id uuid.UUID) error {
	return r.db.Delete(&models.QuoteItem{}, "id = ?", id).Error
}

// UpdateStatus updates the status of a quote
func (r *QuoteRepository) UpdateStatus(id uuid.UUID, status models.QuoteStatus) error {
	return r.db.Session(&gorm.Session{SkipHooks: true}).Model(&models.Quote{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// ErrQuoteStatusConflict is returned when a quote is no longer in a status
// that allows the requested change (e.g. it was accepted or declined by
// someone else in the meantime, or has expired).
var ErrQuoteStatusConflict = errors.New("quote cannot be changed in its current status")

// transition changes a quote's status only if it is still in one of from
// (and, when requireValid, not past valid_until). The check and the write
// are one conditional UPDATE, so two people accepting/declining at once
// can't both succeed (audit 7.M6): the loser affects 0 rows.
func (r *QuoteRepository) transition(id uuid.UUID, from []models.QuoteStatus, to models.QuoteStatus, requireValid bool) error {
	// SkipHooks: Quote.BeforeUpdate validates the whole (here empty) model and
	// would reject every status-only update.
	q := r.db.Session(&gorm.Session{SkipHooks: true}).Model(&models.Quote{}).Where("id = ? AND status IN ?", id, from)
	if requireValid {
		q = q.Where("valid_until > ?", time.Now())
	}
	res := q.Update("status", to)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		if _, err := r.GetByID(id); err != nil {
			return err // not found (or not this tenant's)
		}
		return ErrQuoteStatusConflict
	}
	return nil
}

// MarkAsSent marks a draft quote as sent
func (r *QuoteRepository) MarkAsSent(id uuid.UUID) error {
	return r.transition(id, []models.QuoteStatus{models.QuoteStatusDraft}, models.QuoteStatusSent, true)
}

// MarkAsAccepted marks a sent (or viewed), unexpired quote as accepted
func (r *QuoteRepository) MarkAsAccepted(id uuid.UUID) error {
	return r.transition(id, []models.QuoteStatus{models.QuoteStatusSent, models.QuoteStatusViewed}, models.QuoteStatusAccepted, true)
}

// MarkAsRejected marks a sent (or viewed) quote as declined
func (r *QuoteRepository) MarkAsRejected(id uuid.UUID) error {
	return r.transition(id, []models.QuoteStatus{models.QuoteStatusSent, models.QuoteStatusViewed}, models.QuoteStatusDeclined, false)
}

// MarkAsRevised takes a sent, viewed, expired or declined quote back to
// draft so it can be edited and sent again.
func (r *QuoteRepository) MarkAsRevised(id uuid.UUID) error {
	return r.transition(id, []models.QuoteStatus{
		models.QuoteStatusSent, models.QuoteStatusViewed, models.QuoteStatusExpired, models.QuoteStatusDeclined,
	}, models.QuoteStatusDraft, false)
}

// GetQuoteCount returns the total number of quotes
func (r *QuoteRepository) GetQuoteCount() (int64, error) {
	var count int64
	err := r.db.Model(&models.Quote{}).Count(&count).Error
	return count, err
}

// GetQuoteCountByStatus returns the number of quotes by status
func (r *QuoteRepository) GetQuoteCountByStatus(status models.QuoteStatus) (int64, error) {
	var count int64
	err := r.db.Model(&models.Quote{}).
		Where("status = ?", status).
		Count(&count).Error
	return count, err
}

// requireCustomerInTenant returns ErrCustomerNotFound unless the customer is
// visible to the caller (tenant scoping makes other tenants' customers
// invisible), so documents can't reference another tenant's customer.
func requireCustomerInTenant(db *gorm.DB, customerID uuid.UUID) error {
	var count int64
	if err := db.Model(&models.Customer{}).Where("id = ? AND deleted_at IS NULL", customerID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrCustomerNotFound
	}
	return nil
}
