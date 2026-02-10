package routes

import (
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/internal/repository"
)

// SetupRepositories initializes all repository instances
func SetupRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		User:       repository.NewUserRepositoryGorm(db),
		Customer:   repository.NewCustomerRepository(db),
		Job:        repository.NewJobRepository(db),
		Quote:      repository.NewQuoteRepository(db),
		Tenant:     repository.NewTenantRepository(db),
		Permission: repository.NewPermissionRepository(db),
		Membership: repository.NewMembershipRepository(db),
		Payment:    repository.NewPaymentRepository(db),
		Schedule:   repository.NewScheduleRepository(db),
	}
}
