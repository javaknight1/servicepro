package services

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	apperrors "github.com/javaknight1/servicepro/backend/internal/errors"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/internal/tenancy"
)

// Site domain errors (T186).
var (
	// ErrSiteNotFound: no such site under that customer in the caller's tenant.
	ErrSiteNotFound = apperrors.NotFound("site_not_found", "Site not found")
	// ErrSiteCustomerMismatch: a job (or agreement) named a site that isn't
	// one of its customer's active sites. Other tenants' sites look the same.
	ErrSiteCustomerMismatch = apperrors.BadRequest("site_customer_mismatch", "The site doesn't belong to this customer")
)

// SiteServiceInterface is the customer-sites use case the handler calls.
// The tenant always comes from ctx (set by the tenant middleware).
type SiteServiceInterface interface {
	List(ctx context.Context, customerID uuid.UUID, includeInactive bool) ([]models.Site, error)
	Get(ctx context.Context, customerID, siteID uuid.UUID) (*models.Site, error)
	Create(ctx context.Context, customerID uuid.UUID, req *models.CreateSiteRequest) (*models.Site, error)
	Update(ctx context.Context, customerID, siteID uuid.UUID, req *models.UpdateSiteRequest) (*models.Site, error)
	Delete(ctx context.Context, customerID, siteID uuid.UUID) error
}

// SiteService manages customer sites.
type SiteService struct {
	db    *gorm.DB
	sites repository.SiteRepositoryInterface
}

// NewSiteService creates a site service.
func NewSiteService(db *gorm.DB, sites repository.SiteRepositoryInterface) *SiteService {
	return &SiteService{db: db, sites: sites}
}

// requireTenantCustomer returns the tenant in ctx, or ErrCustomerNotFound
// unless customerID is one of its (non-deleted) customers.
func requireTenantCustomer(ctx context.Context, db *gorm.DB, customerID uuid.UUID) (uuid.UUID, error) {
	tenantID, err := tenancy.MustTenantID(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	var n int64
	if err := db.WithContext(ctx).Model(&models.Customer{}).
		Where("tenant_id = ? AND id = ? AND deleted_at IS NULL", tenantID, customerID).
		Count(&n).Error; err != nil {
		return uuid.Nil, err
	}
	if n == 0 {
		return uuid.Nil, ErrCustomerNotFound
	}
	return tenantID, nil
}

func siteError(err error) error {
	if errors.Is(err, repository.ErrSiteNotFound) {
		return ErrSiteNotFound
	}
	return err
}

// List returns the customer's sites (active only unless includeInactive).
func (s *SiteService) List(ctx context.Context, customerID uuid.UUID, includeInactive bool) ([]models.Site, error) {
	tenantID, err := requireTenantCustomer(ctx, s.db, customerID)
	if err != nil {
		return nil, err
	}
	return s.sites.ListByCustomer(ctx, tenantID, customerID, includeInactive)
}

// Get returns one of the customer's sites.
func (s *SiteService) Get(ctx context.Context, customerID, siteID uuid.UUID) (*models.Site, error) {
	tenantID, err := requireTenantCustomer(ctx, s.db, customerID)
	if err != nil {
		return nil, err
	}
	site, err := s.sites.GetForCustomer(ctx, tenantID, customerID, siteID)
	return site, siteError(err)
}

// Create adds a site to the customer.
func (s *SiteService) Create(ctx context.Context, customerID uuid.UUID, req *models.CreateSiteRequest) (*models.Site, error) {
	tenantID, err := requireTenantCustomer(ctx, s.db, customerID)
	if err != nil {
		return nil, err
	}
	site := &models.Site{
		CustomerID:    customerID,
		Name:          strings.TrimSpace(req.Name),
		AddressStreet: req.AddressStreet,
		AddressCity:   req.AddressCity,
		AddressState:  strings.ToUpper(req.AddressState),
		AddressZip:    req.AddressZip,
		AccessNotes:   req.AccessNotes,
		ContactName:   req.ContactName,
		ContactPhone:  req.ContactPhone,
		ContactEmail:  req.ContactEmail,
		IsActive:      req.IsActive == nil || *req.IsActive,
	}
	if site.Name == "" {
		return nil, apperrors.Validation("Site name is required", map[string]string{"name": "is required"})
	}
	if err := s.sites.Create(ctx, tenantID, site); err != nil {
		return nil, err
	}
	return s.sites.GetForCustomer(ctx, tenantID, customerID, site.ID)
}

// Update changes the fields present in req.
func (s *SiteService) Update(ctx context.Context, customerID, siteID uuid.UUID, req *models.UpdateSiteRequest) (*models.Site, error) {
	tenantID, err := requireTenantCustomer(ctx, s.db, customerID)
	if err != nil {
		return nil, err
	}
	site, err := s.sites.GetForCustomer(ctx, tenantID, customerID, siteID)
	if err != nil {
		return nil, siteError(err)
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, apperrors.Validation("Site name is required", map[string]string{"name": "is required"})
		}
		site.Name = name
	}
	setString(&site.AddressStreet, req.AddressStreet)
	setString(&site.AddressCity, req.AddressCity)
	if req.AddressState != nil {
		site.AddressState = strings.ToUpper(*req.AddressState)
	}
	setString(&site.AddressZip, req.AddressZip)
	if req.AccessNotes != nil {
		site.AccessNotes = req.AccessNotes
	}
	if req.ContactName != nil {
		site.ContactName = req.ContactName
	}
	if req.ContactPhone != nil {
		site.ContactPhone = req.ContactPhone
	}
	if req.ContactEmail != nil {
		site.ContactEmail = req.ContactEmail
	}
	if req.IsActive != nil {
		site.IsActive = *req.IsActive
	}
	if err := s.sites.Update(ctx, tenantID, site); err != nil {
		return nil, siteError(err)
	}
	site, err = s.sites.GetForCustomer(ctx, tenantID, customerID, siteID)
	return site, siteError(err)
}

// Delete removes one of the customer's sites.
func (s *SiteService) Delete(ctx context.Context, customerID, siteID uuid.UUID) error {
	tenantID, err := requireTenantCustomer(ctx, s.db, customerID)
	if err != nil {
		return err
	}
	return siteError(s.sites.Delete(ctx, tenantID, customerID, siteID))
}

func setString(dst *string, v *string) {
	if v != nil {
		*dst = *v
	}
}

// resolveJobSite returns the site a job may use: an active site of
// customerID in the caller's tenant, or ErrSiteCustomerMismatch.
func resolveJobSite(ctx context.Context, db *gorm.DB, customerID, siteID uuid.UUID) (*models.Site, error) {
	tenantID, err := tenancy.MustTenantID(ctx)
	if err != nil {
		return nil, err
	}
	var site models.Site
	err = db.WithContext(ctx).
		Where("tenant_id = ? AND customer_id = ? AND id = ? AND is_active = TRUE", tenantID, customerID, siteID).
		First(&site).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrSiteCustomerMismatch
	}
	if err != nil {
		return nil, err
	}
	return &site, nil
}
