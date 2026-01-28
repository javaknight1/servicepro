package routes

import (
	"log"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/internal/services"
	permissionsSvc "github.com/javaknight1/servicepro/backend/internal/services/permissions"
	stripeService "github.com/javaknight1/servicepro/backend/internal/services/stripe"
	"github.com/javaknight1/servicepro/backend/pkg/auth"
	emailclient "github.com/javaknight1/servicepro/backend/pkg/clients/email"
	storageclient "github.com/javaknight1/servicepro/backend/pkg/clients/storage"
)

// SetupServices initializes all service instances
func SetupServices(
	db *gorm.DB,
	redisClient *redis.Client,
	repos *Repositories,
	cfg *config.Config,
	emailClient emailclient.Client,
	storageClient storageclient.Client,
	jwtManager *auth.JWTManager,
	permChecker *permissionsSvc.PermissionChecker,
) *Services {
	svc := &Services{}

	// Auth service
	svc.Auth = services.NewAuthService(repos.User, jwtManager, &cfg.Auth)

	// Email verification service
	verificationURL := cfg.Server.FrontendURL + "/verify-email"
	svc.EmailVerification = services.NewEmailVerificationService(repos.User, emailClient, redisClient, verificationURL)

	// Registration service
	svc.Registration = services.NewRegistrationService(repos.User, emailClient, svc.EmailVerification)

	// Password reset service
	resetURL := cfg.Server.FrontendURL + "/reset-password"
	svc.PasswordReset = services.NewPasswordResetService(repos.User, emailClient, redisClient, resetURL)

	// Import service
	svc.Import = services.NewImportService(db, repos.Customer, redisClient)

	// Job service
	svc.Job = services.NewJobService(repos.Job, repos.Customer, db)

	// Quote service
	svc.Quote = services.NewQuoteService(repos.Quote)

	// Invoice service
	svc.Invoice = services.NewInvoiceService(db)

	// Revenue report service
	svc.Revenue = services.NewRevenueReportService(db)

	// Customer report service
	svc.Customer = services.NewCustomerReportService(db)

	// PDF service
	svc.PDF = services.NewPDFService("", "", "./exports/pdf")

	// Export service
	svc.Export = services.NewUnifiedExportService(db, svc.PDF, "./exports", "")

	// Payment terms service
	svc.PaymentTerms = services.NewPaymentTermsService(db)

	// Profile picture service (optional - depends on storage client)
	if storageClient != nil {
		svc.ProfilePicture = services.NewProfilePictureService(repos.User, storageClient)
	}

	// Stripe service (optional - depends on configuration)
	if cfg.Stripe.SecretKey != "" {
		svc.Stripe = services.NewStripeService(&cfg.Stripe, repos.Tenant, repos.Membership, repos.Payment)

		// Stripe client and webhook handler
		client, err := stripeService.NewClient(cfg)
		if err != nil {
			log.Printf("[SERVICES] Failed to create Stripe client: %v", err)
		} else {
			svc.StripeClient = client
			svc.StripeEvents = stripeService.NewEventProcessor()
			svc.StripeEvents.RegisterDefaultHandlers()
			svc.StripeWebhook = stripeService.NewWebhookHandler(cfg, svc.StripeEvents)
		}
	}

	// Membership service
	svc.Membership = services.NewMembershipService(repos.Membership, svc.Stripe)

	// Tenant service
	svc.Tenant = services.NewTenantService(repos.Tenant, repos.User)
	svc.Tenant.SetPermissionInvalidator(permChecker)
	svc.Tenant.SetMembershipAssigner(svc.Membership)

	// Invitation service
	svc.Invitation = services.NewInvitationService(db, emailClient, cfg.Server.FrontendURL)

	return svc
}
