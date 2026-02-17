package routeconfigs

import (
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/internal/api/handlers"
	"github.com/javaknight1/servicepro/backend/internal/api/middleware"
	"github.com/javaknight1/servicepro/backend/internal/repository"
	"github.com/javaknight1/servicepro/backend/internal/services"
	"github.com/javaknight1/servicepro/backend/internal/services/conflict"
	permissionsSvc "github.com/javaknight1/servicepro/backend/internal/services/permissions"
	stripeService "github.com/javaknight1/servicepro/backend/internal/services/stripe"
	"github.com/javaknight1/servicepro/backend/pkg/auth"
	analyticsclient "github.com/javaknight1/servicepro/backend/pkg/clients/analytics"
	emailclient "github.com/javaknight1/servicepro/backend/pkg/clients/email"
	smsclient "github.com/javaknight1/servicepro/backend/pkg/clients/sms"
	storageclient "github.com/javaknight1/servicepro/backend/pkg/clients/storage"
)

// RouteConfig holds all dependencies needed for route setup
type RouteConfig struct {
	Config     *config.Config
	Clients    *Clients
	Repos      *Repositories
	Middleware *Middleware
	Services   *Services
	Handlers   *Handlers
}

// Clients holds all external client instances
type Clients struct {
	DB        *gorm.DB
	Redis     *redis.Client
	Email     emailclient.Client
	Storage   storageclient.Client
	SMS       smsclient.Client
	Analytics analyticsclient.Client
}

// Repositories holds all repository instances
type Repositories struct {
	User       repository.UserRepositoryGormInterface
	Customer   repository.CustomerRepositoryInterface
	Job        repository.JobRepositoryInterface
	Quote      repository.QuoteRepositoryInterface
	Tenant     *repository.TenantRepository
	Permission *repository.PermissionRepository
	Membership *repository.MembershipRepository
	Payment    *repository.PaymentRepository
	Schedule   repository.ScheduleRepositoryInterface
}

// Services holds all service instances
type Services struct {
	Auth              *services.AuthService
	Registration      services.RegistrationServiceInterface
	EmailVerification *services.EmailVerificationService
	PasswordReset     *services.PasswordResetService
	Customer          services.CustomerReportService
	Job               *services.JobService
	Quote             *services.QuoteService
	Invoice           *services.InvoiceService
	InvoicePayment    *services.InvoicePaymentService
	Tenant            *services.TenantService
	Invitation        *services.InvitationService
	Membership        *services.MembershipService
	Stripe            *services.StripeService
	ProfilePicture    *services.ProfilePictureService
	Revenue           services.RevenueReportService
	ARAging           services.ARAgingService
	Export            services.UnifiedExportService
	PDF               *services.PDFService
	Import            *services.ImportService
	PaymentTerms      *services.PaymentTermsService
	StripeClient      *stripeService.Client
	StripeEvents      *stripeService.EventProcessor
	StripeWebhook     *stripeService.WebhookHandler
	Conflict          *conflict.ConflictDetector
	PaymentReminder   *services.PaymentReminderService
}

// Middleware holds all middleware instances
type Middleware struct {
	JWTManager     *auth.JWTManager
	CookieManager  *auth.CookieManager
	PermChecker    *permissionsSvc.PermissionChecker
	PermMiddleware *middleware.PermissionMiddleware
	TenantMW       *middleware.TenantMiddleware
	RateLimiter    *middleware.RateLimiter
	APIRateLimiter *middleware.APIRateLimiter
}

// Handlers holds all handler instances
type Handlers struct {
	Auth              *handlers.AuthHandler
	Registration      *handlers.RegistrationHandler
	EmailVerification *handlers.EmailVerificationHandler
	PasswordReset     *handlers.PasswordResetHandler
	User              *handlers.UserHandler
	Customer          *handlers.CustomerHandler
	CustomerReport    *handlers.CustomerReportHandler
	Job               *handlers.JobHandler
	Quote             *handlers.QuoteHandler
	Invoice           *handlers.InvoiceHandler
	PublicInvoice     *handlers.PublicInvoiceHandler
	Tenant            *handlers.TenantHandler
	Invitation        *handlers.InvitationHandler
	Role              *handlers.RoleHandler
	Revenue           *handlers.RevenueHandler
	ARAging           *handlers.ARAgingHandler
	Stripe            *handlers.StripeHandler
	Billing           *handlers.BillingHandler
	Webhook           *handlers.WebhookHandler
	Membership        *handlers.MembershipHandler
	Export            *handlers.ExportHandler
	ImportExport      *handlers.ImportExportHandler
	ProfilePicture    *handlers.ProfilePictureHandler
	PaymentTerms      *handlers.PaymentTermsHandler
	Conflict          *handlers.ConflictHandler
	PaymentReminder   *handlers.PaymentReminderHandler
}
