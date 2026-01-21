package repository

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
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

// Create creates a new quote
func (r *QuoteRepository) Create(quote *models.Quote) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
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
		// Update quote
		if err := tx.Save(quote).Error; err != nil {
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

// List retrieves quotes with pagination and filters
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
		Order("created_at DESC").
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
	return r.db.Model(&models.Quote{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// MarkAsSent marks a quote as sent
func (r *QuoteRepository) MarkAsSent(id uuid.UUID) error {
	quote, err := r.GetByID(id)
	if err != nil {
		return err
	}

	if !quote.CanSend() {
		return fmt.Errorf("quote cannot be sent in current status: %s", quote.Status)
	}

	return r.UpdateStatus(id, models.QuoteStatusSent)
}

// MarkAsAccepted marks a quote as accepted
func (r *QuoteRepository) MarkAsAccepted(id uuid.UUID) error {
	quote, err := r.GetByID(id)
	if err != nil {
		return err
	}

	if !quote.CanAccept() {
		return fmt.Errorf("quote cannot be accepted in current status: %s", quote.Status)
	}

	return r.UpdateStatus(id, models.QuoteStatusAccepted)
}

// MarkAsRejected marks a quote as rejected
func (r *QuoteRepository) MarkAsRejected(id uuid.UUID) error {
	quote, err := r.GetByID(id)
	if err != nil {
		return err
	}

	if !quote.CanReject() {
		return fmt.Errorf("quote cannot be rejected in current status: %s", quote.Status)
	}

	return r.UpdateStatus(id, models.QuoteStatusDeclined)
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
