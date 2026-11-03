package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// ErrSiteNotFound: no such site in the caller's tenant (or not under the
// given customer). Other tenants' sites look exactly like missing ones.
var ErrSiteNotFound = errors.New("site not found")

// SiteRepositoryInterface is data access for customer sites (T186). Every
// method takes ctx (which carries the tenant for the tenancy plugin and the
// RLS backstop) and the tenant ID, and filters tenant_id explicitly too.
type SiteRepositoryInterface interface {
	Create(ctx context.Context, tenantID uuid.UUID, site *models.Site) error
	// Get returns a site of the tenant by ID, whatever its customer.
	Get(ctx context.Context, tenantID, id uuid.UUID) (*models.Site, error)
	// GetForCustomer returns a site only if it belongs to customerID.
	GetForCustomer(ctx context.Context, tenantID, customerID, id uuid.UUID) (*models.Site, error)
	ListByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, includeInactive bool) ([]models.Site, error)
	// Update saves every editable column of site (matched by tenant,
	// customer and ID).
	Update(ctx context.Context, tenantID uuid.UUID, site *models.Site) error
	Delete(ctx context.Context, tenantID, customerID, id uuid.UUID) error
}

// SiteRepository implements SiteRepositoryInterface with GORM.
type SiteRepository struct {
	db *gorm.DB
}

// NewSiteRepository creates a site repository.
func NewSiteRepository(db *gorm.DB) *SiteRepository {
	return &SiteRepository{db: db}
}

// Create inserts site into tenantID (the tenant always comes from the
// caller, never from the site value).
func (r *SiteRepository) Create(ctx context.Context, tenantID uuid.UUID, site *models.Site) error {
	site.TenantID = tenantID
	if site.ID == uuid.Nil {
		site.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(site).Error
}

func (r *SiteRepository) first(ctx context.Context, q *gorm.DB) (*models.Site, error) {
	var site models.Site
	if err := q.First(&site).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSiteNotFound
		}
		return nil, err
	}
	return &site, nil
}

// Get returns one of the tenant's sites.
func (r *SiteRepository) Get(ctx context.Context, tenantID, id uuid.UUID) (*models.Site, error) {
	return r.first(ctx, r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, id))
}

// GetForCustomer returns one of the customer's sites.
func (r *SiteRepository) GetForCustomer(ctx context.Context, tenantID, customerID, id uuid.UUID) (*models.Site, error) {
	return r.first(ctx, r.db.WithContext(ctx).
		Where("tenant_id = ? AND customer_id = ? AND id = ?", tenantID, customerID, id))
}

// ListByCustomer returns the customer's sites by name.
func (r *SiteRepository) ListByCustomer(ctx context.Context, tenantID, customerID uuid.UUID, includeInactive bool) ([]models.Site, error) {
	q := r.db.WithContext(ctx).Where("tenant_id = ? AND customer_id = ?", tenantID, customerID)
	if !includeInactive {
		q = q.Where("is_active = TRUE")
	}
	sites := []models.Site{}
	if err := q.Order("name ASC, created_at ASC").Find(&sites).Error; err != nil {
		return nil, err
	}
	return sites, nil
}

// siteColumns are the columns Update writes. Select makes GORM write false
// and nil values too.
var siteColumns = []string{
	"name", "address_street", "address_city", "address_state", "address_zip",
	"access_notes", "contact_name", "contact_phone", "contact_email", "is_active", "updated_at",
}

// Update saves the site's editable columns.
func (r *SiteRepository) Update(ctx context.Context, tenantID uuid.UUID, site *models.Site) error {
	res := r.db.WithContext(ctx).Model(&models.Site{}).
		Where("tenant_id = ? AND customer_id = ? AND id = ?", tenantID, site.CustomerID, site.ID).
		Select(siteColumns).Updates(site)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrSiteNotFound
	}
	return nil
}

// Delete soft-deletes one of the customer's sites. Jobs keep pointing at it
// (their history stays readable); new jobs can't pick it.
func (r *SiteRepository) Delete(ctx context.Context, tenantID, customerID, id uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("tenant_id = ? AND customer_id = ? AND id = ?", tenantID, customerID, id).
		Delete(&models.Site{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrSiteNotFound
	}
	return nil
}
