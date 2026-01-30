package routes

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/internal/services"
	"github.com/javaknight1/servicepro/backend/internal/services/pdf"
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

	// Document PDF service (for generating quote/invoice/receipt PDFs)
	var documentPDFService *services.DocumentPDFService
	if storageClient != nil {
		pdfConfig := pdf.DefaultConfig()
		pdfConfig.Storage.Type = "local" // Use local storage for temp files, upload to S3 via storage client
		pdfConfig.Storage.LocalPath = "./exports/pdf"
		pdfGenerator, err := pdf.NewGenerator(context.Background(), pdfConfig, nil)
		if err != nil {
			log.Printf("[SERVICES] Failed to create PDF generator: %v", err)
		} else {
			documentPDFService = services.NewDocumentPDFService(pdfGenerator, storageClient)
			log.Println("[SERVICES] Document PDF service initialized")
		}
	} else {
		log.Println("[SERVICES] Document PDF service not available (storage client not configured)")
	}

	// Quote service (with PDF and email for sending quotes)
	svc.Quote = services.NewQuoteServiceWithConfig(repos.Quote, &services.QuoteServiceConfig{
		PDFService:  documentPDFService,
		EmailClient: emailClient,
	})

	// Invoice service (with email client and PDF service for sending invoices)
	svc.Invoice = services.NewInvoiceServiceWithConfig(db, &services.InvoiceServiceConfig{
		EmailClient: emailClient,
		PDFService:  documentPDFService,
		FrontendURL: cfg.Server.FrontendURL,
		BackendURL:  cfg.Server.BackendURL,
	})

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

			// Invoice payment service (for Stripe checkout)
			invoicePaymentSvc, err := services.NewInvoicePaymentService(&services.InvoicePaymentServiceConfig{
				StripeClient:   client,
				InvoiceService: svc.Invoice,
				SuccessURL:     cfg.Server.FrontendURL + "/invoices/payment-success",
				CancelURL:      cfg.Server.FrontendURL + "/invoices/payment-cancelled",
			})
			if err != nil {
				log.Printf("[SERVICES] Failed to create invoice payment service: %v", err)
			} else {
				svc.InvoicePayment = invoicePaymentSvc

				// Register checkout.session.completed handler for invoice payments
				invoicePaymentHandler := services.NewInvoicePaymentHandlerAdapter(svc.Invoice)
				svc.StripeEvents.RegisterHandler(
					stripeService.EventCheckoutSessionCompleted,
					invoicePaymentHandler.CreateCheckoutCompletedHandler(),
				)
			}
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
