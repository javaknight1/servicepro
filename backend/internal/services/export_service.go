package services

import (
	"context"
	"fmt"

	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/repository"
)

// ExportService handles customer export operations
type ExportService struct {
	customerRepo repository.CustomerRepositoryInterface
}

// NewExportService creates a new export service
func NewExportService(customerRepo repository.CustomerRepositoryInterface) *ExportService {
	return &ExportService{
		customerRepo: customerRepo,
	}
}

// GetCustomersForExport retrieves customers based on export filters
func (s *ExportService) GetCustomersForExport(
	ctx context.Context,
	customerType models.CustomerType,
	status models.CustomerStatus,
	city, state string,
) ([]models.Customer, error) {
	filter := &models.CustomerListFilter{
		CustomerType: customerType,
		Status:       status,
		City:         city,
		State:        state,
		Page:         1,
		PageSize:     10000, // Large page size for export
	}

	customers, _, err := s.customerRepo.List(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve customers: %w", err)
	}

	return customers, nil
}

// GetAllCustomers retrieves all customers for export
func (s *ExportService) GetAllCustomers(ctx context.Context) ([]models.Customer, error) {
	filter := &models.CustomerListFilter{
		Page:     1,
		PageSize: 10000, // Large page size for export
	}

	customers, _, err := s.customerRepo.List(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve customers: %w", err)
	}

	return customers, nil
}
