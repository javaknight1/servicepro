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
	"github.com/javaknight1/servicepro/backend/internal/services/quickbooks"
	stripeService "github.com/javaknight1/servicepro/backend/internal/services/stripe"
	"github.com/javaknight1/servicepro/backend/pkg/auth"
	"github.com/javaknight1/servicepro/backend/pkg/auth/neonauth"
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
	User     repository.UserRepositoryGormInterface
	Customer repository.CustomerRepositoryInterface
	Job      repository.JobRepositoryInterface
	Quote    repository.QuoteRepositoryInterface
	Tenant   *repository.TenantRepository
	// EmailSuppressions lists bounced/complained addresses (T159).
	EmailSuppressions *repository.EmailSuppressionRepository
	Permission        *repository.PermissionRepository
	Membership        *repository.MembershipRepository
	Payment           *repository.PaymentRepository
	Schedule          repository.ScheduleRepositoryInterface
	// ConnectedAccounts stores each tenant's Stripe Connect account (T169).
	ConnectedAccounts *repository.ConnectedAccountRepository
	// Sites are customers' service locations (T186).
	Sites repository.SiteRepositoryInterface
	// PurchaseOrders are customers' POs with amounts (T187).
	PurchaseOrders repository.PurchaseOrderRepositoryInterface
	// JobApprovals are approvers' OKs for extra work (T187).
	JobApprovals repository.JobApprovalRepositoryInterface
	// Agreements are recurring service agreements (T167).
	Agreements repository.AgreementRepositoryInterface
	// JobAttachments are job photos and files (T174).
	JobAttachments repository.JobAttachmentRepositoryInterface
	// JobSignoffs are customers' sign-offs and service reports (T176).
	JobSignoffs repository.JobSignoffRepositoryInterface
}

// Services holds all service instances
type Services struct {
	Auth              *services.AuthService
	Registration      services.RegistrationServiceInterface
	EmailVerification *services.EmailVerificationService
	PasswordReset     *services.PasswordResetService
	Job               *services.JobService
	Quote             *services.QuoteService
	Invoice           *services.InvoiceService
	InvoicePayment    *services.InvoicePaymentService
	Tenant            *services.TenantService
	Invitation        *services.InvitationService
	InviteCodes       *services.InviteCodeService
	NeonProvisioning  *services.NeonProvisioningService
	Membership        *services.MembershipService
	Stripe            *services.StripeService
	ProfilePicture    *services.ProfilePictureService
	// JobAttachments stores job photos and files (T174); nil without storage.
	JobAttachments *services.JobAttachmentService
	// JobSignoffs stores sign-offs and emails service reports (T176); nil
	// without storage.
	JobSignoffs   *services.JobSignoffService
	Revenue       services.RevenueReportService
	ARAging       services.ARAgingService
	Export        services.UnifiedExportService
	Import        *services.ImportService
	PaymentTerms  *services.PaymentTermsService
	TaxRates      *services.TaxRateService
	StripeClient  *stripeService.Client
	StripeEvents  *stripeService.EventProcessor
	StripeWebhook *stripeService.WebhookHandler
	// StripeConnectWebhook receives events from connected accounts and
	// StripeV2Webhook thin v2 account events (T169); nil when unconfigured.
	StripeConnectWebhook *stripeService.WebhookHandler
	StripeV2Webhook      *stripeService.ThinEventHandler
	// PaymentsConnect connects tenants' own Stripe accounts (T169).
	PaymentsConnect *services.PaymentsConnectService
	Conflict        *conflict.ConflictDetector
	PaymentReminder *services.PaymentReminderService
	Sites           *services.SiteService
	PurchaseOrders  *services.PurchaseOrderService
	JobApprovals    *services.JobApprovalService
	Agreements      *services.AgreementService
	// QuoteConversion turns accepted quotes into jobs (T028).
	QuoteConversion *services.QuoteConversionService
	// QuickBooks is the QuickBooks Online sync (T192); always set, answers
	// "not configured" without Intuit keys.
	QuickBooks *quickbooks.Service
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
	// NeonAuth verifies Neon Auth JWTs; nil when NEON_AUTH_BASE_URL is unset.
	NeonAuth *neonauth.Verifier
}

// Handlers holds all handler instances
type Handlers struct {
	Auth              *handlers.AuthHandler
	Registration      *handlers.RegistrationHandler
	EmailVerification *handlers.EmailVerificationHandler
	PasswordReset     *handlers.PasswordResetHandler
	User              *handlers.UserHandler
	Customer          *handlers.CustomerHandler
	Job               *handlers.JobHandler
	Quote             *handlers.QuoteHandler
	Invoice           *handlers.InvoiceHandler
	PublicInvoice     *handlers.PublicInvoiceHandler
	Tenant            *handlers.TenantHandler
	Invitation        *handlers.InvitationHandler
	InviteCode        *handlers.InviteCodeHandler
	Role              *handlers.RoleHandler
	Revenue           *handlers.RevenueHandler
	ARAging           *handlers.ARAgingHandler
	Stripe            *handlers.StripeHandler
	PaymentsConnect   *handlers.PaymentsConnectHandler
	ResendWebhook     *handlers.ResendWebhookHandler
	Billing           *handlers.BillingHandler
	Membership        *handlers.MembershipHandler
	Export            *handlers.ExportHandler
	ImportExport      *handlers.ImportExportHandler
	Imports           *handlers.ImportsHandler // mapped imports (T193)
	ProfilePicture    *handlers.ProfilePictureHandler
	// JobAttachment is nil when no object storage is configured (T174).
	JobAttachment *handlers.JobAttachmentHandler
	// JobSignoff is nil when no object storage is configured (T176).
	JobSignoff      *handlers.JobSignoffHandler
	PaymentTerms    *handlers.PaymentTermsHandler
	TaxRate         *handlers.TaxRateHandler
	Conflict        *handlers.ConflictHandler
	PaymentReminder *handlers.PaymentReminderHandler
	NeonAuth        *handlers.NeonAuthHandler
	Site            *handlers.SiteHandler
	PurchaseOrder   *handlers.PurchaseOrderHandler
	JobApproval     *handlers.JobApprovalHandler
	Agreement       *handlers.AgreementHandler
	QuoteConversion *handlers.QuoteConversionHandler
	// QuickBooks serves Settings → Integrations → QuickBooks (T192).
	QuickBooks *handlers.QuickBooksHandler
}
