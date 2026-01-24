package repository

import (
	"time"

	"github.com/google/uuid"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// =============================================================================
// User Repository Interfaces
// =============================================================================

// UserRepositoryInterface defines the interface for user repository (legacy with int IDs)
type UserRepositoryInterface interface {
	GetByEmail(email string) (*models.User, error)
	IncrementFailedLoginCount(userID int) error
	LockAccount(userID int, until time.Time) error
	ResetFailedLoginCount(userID int) error
	Create(email, passwordHash string) (*models.User, error)
}

// UserRepositoryGormInterface extends UserRepositoryInterface with UUID methods
type UserRepositoryGormInterface interface {
	UserRepositoryInterface
	GetByID(id uuid.UUID) (*models.User, error)
	IncrementFailedLoginCountByUUID(userID uuid.UUID) error
	LockAccountByUUID(userID uuid.UUID, until time.Time) error
	ResetFailedLoginCountByUUID(userID uuid.UUID) error
	EmailExists(email string) (bool, error)
	UpdatePassword(userID uuid.UUID, passwordHash string) error
	MarkEmailVerified(userID uuid.UUID) error
	UpdateVerificationSentAt(userID uuid.UUID, sentAt *time.Time) error
	GetUnverifiedUsersForReminder(threshold time.Duration) ([]*models.User, error)
	UpdateProfilePicture(userID uuid.UUID, url *string) error
}

// =============================================================================
// Customer Repository Interface
// =============================================================================

// CustomerRepositoryInterface defines the interface for customer repository
type CustomerRepositoryInterface interface {
	Create(customer *models.Customer) error
	GetByID(id uuid.UUID) (*models.Customer, error)
	GetByEmail(email string) (*models.Customer, error)
	Update(customer *models.Customer) error
	Delete(id uuid.UUID) error
	List(filter *models.CustomerListFilter) ([]models.Customer, int64, error)
	EmailExists(email string) (bool, error)
	PhoneExists(phone string) (bool, error)
	Search(query string) ([]models.Customer, error)
	GetByStatus(status models.CustomerStatus) ([]models.Customer, error)
	GetByType(customerType models.CustomerType) ([]models.Customer, error)
	FullTextSearch(query string, customerType models.CustomerType, status models.CustomerStatus, city, state string, limit, offset int, sortBy, sortOrder string) ([]models.Customer, int64, error)
	Autocomplete(prefix string, limit int) ([]map[string]interface{}, error)
	GetSearchStats() (map[string]interface{}, error)
}

// =============================================================================
// Job Repository Interface
// =============================================================================

// JobRepositoryInterface defines the interface for job repository
type JobRepositoryInterface interface {
	// Basic CRUD operations
	Create(job *models.Job) error
	GetByID(id uuid.UUID) (*models.Job, error)
	GetByJobNumber(jobNumber string) (*models.Job, error)
	Update(job *models.Job) error
	Delete(id uuid.UUID) error

	// List and filter operations
	List(filter *models.JobListFilter) ([]models.Job, int64, error)
	GetByCustomer(customerID uuid.UUID, limit, offset int) ([]models.Job, int64, error)
	GetByStatus(status models.JobStatus) ([]models.Job, error)
	GetByAssignedUser(userID uuid.UUID) ([]models.Job, error)

	// Job-specific queries
	GetScheduledJobs(start, end time.Time) ([]models.Job, error)
	GetOverdueJobs() ([]models.Job, error)
	GetJobsRequiringFollowUp() ([]models.Job, error)

	// Assignment operations
	AddAssignment(assignment *models.JobAssignment) error
	RemoveAssignment(assignmentID uuid.UUID) error
	GetJobAssignments(jobID uuid.UUID) ([]models.JobAssignment, error)

	// Material operations
	AddMaterial(material *models.JobMaterial) error
	UpdateMaterial(material *models.JobMaterial) error
	DeleteMaterial(materialID uuid.UUID) error
	GetJobMaterials(jobID uuid.UUID) ([]models.JobMaterial, error)

	// Note operations
	AddNote(note *models.JobNote) error
	GetJobNotes(jobID uuid.UUID, includeInternal bool) ([]models.JobNote, error)

	// Statistics
	GetJobStats() (map[string]interface{}, error)
	GetTechnicianWorkload(userID uuid.UUID) (map[string]interface{}, error)
}

// =============================================================================
// Quote Repository Interface
// =============================================================================

// QuoteRepositoryInterface defines the interface for quote repository
type QuoteRepositoryInterface interface {
	// Quote CRUD operations
	Create(quote *models.Quote) error
	GetByID(id uuid.UUID) (*models.Quote, error)
	GetByQuoteNumber(quoteNumber string) (*models.Quote, error)
	Update(quote *models.Quote) error
	Delete(id uuid.UUID) error

	// List and filter operations
	List(params *models.QuoteQueryParams) ([]models.Quote, int64, error)
	GetByCustomer(customerID uuid.UUID) ([]models.Quote, error)
	GetByStatus(status models.QuoteStatus) ([]models.Quote, error)

	// Quote item operations
	CreateItem(item *models.QuoteItem) error
	GetItemsByQuote(quoteID uuid.UUID) ([]models.QuoteItem, error)
	UpdateItem(item *models.QuoteItem) error
	DeleteItem(id uuid.UUID) error

	// Status updates
	UpdateStatus(id uuid.UUID, status models.QuoteStatus) error
	MarkAsSent(id uuid.UUID) error
	MarkAsAccepted(id uuid.UUID) error
	MarkAsRejected(id uuid.UUID) error

	// Statistics
	GetQuoteCount() (int64, error)
	GetQuoteCountByStatus(status models.QuoteStatus) (int64, error)
}

// =============================================================================
// Schedule Repository Interface
// =============================================================================

// ScheduleRepositoryInterface defines the interface for schedule repository
type ScheduleRepositoryInterface interface {
	// CRUD operations
	Create(schedule *models.Schedule) error
	GetByID(id uuid.UUID) (*models.Schedule, error)
	Update(schedule *models.Schedule) error
	Delete(id uuid.UUID) error

	// List and filter operations
	List(params *models.ScheduleQueryParams) ([]models.Schedule, int64, error)
	GetByDateRange(startDate, endDate time.Time) ([]models.Schedule, error)
	GetByTechnicianAndDateRange(techID uuid.UUID, startDate, endDate time.Time) ([]models.Schedule, error)
	GetByJobID(jobID uuid.UUID) ([]models.Schedule, error)

	// Conflict detection
	DetectConflicts(schedule *models.Schedule) ([]models.ScheduleConflict, error)
	GetConflicts(scheduleID uuid.UUID) ([]models.ScheduleConflict, error)
	ResolveConflict(conflictID uuid.UUID, resolvedBy uuid.UUID, notes *string) error

	// Recurring schedules
	CreateRecurring(recurring *models.RecurringSchedule) error
	GetRecurringByID(id uuid.UUID) (*models.RecurringSchedule, error)
	UpdateRecurring(recurring *models.RecurringSchedule) error
	DeleteRecurring(id uuid.UUID) error
	GetActiveRecurring() ([]models.RecurringSchedule, error)

	// Statistics
	GetScheduleCount() (int64, error)
	GetConflictCount() (int64, error)
}

// =============================================================================
// Payment Repository Interface
// =============================================================================

// PaymentRepositoryInterface defines the interface for payment repository
type PaymentRepositoryInterface interface {
	CreatePayment(payment *models.Payment) error
	GetPaymentByID(id uuid.UUID) (*models.Payment, error)
	GetPaymentByStripeID(stripePaymentIntentID string) (*models.Payment, error)
	UpdatePayment(payment *models.Payment) error
	GetPaymentsByUserID(userID uuid.UUID, filter *models.PaymentListFilter) ([]models.Payment, error)
	CreateRefund(refund *models.PaymentRefund) error
	GetRefundsByPaymentID(paymentID uuid.UUID) ([]models.PaymentRefund, error)
}
