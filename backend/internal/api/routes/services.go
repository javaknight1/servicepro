package routes

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/internal/services"
	"github.com/javaknight1/servicepro/backend/internal/services/conflict"
	"github.com/javaknight1/servicepro/backend/internal/services/pdf"
	permissionsSvc "github.com/javaknight1/servicepro/backend/internal/services/permissions"
	"github.com/javaknight1/servicepro/backend/internal/services/quickbooks"
	stripeService "github.com/javaknight1/servicepro/backend/internal/services/stripe"
	"github.com/javaknight1/servicepro/backend/pkg/auth"
	emailclient "github.com/javaknight1/servicepro/backend/pkg/clients/email"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
	qboclient "github.com/javaknight1/servicepro/backend/pkg/clients/quickbooks"
	smsclient "github.com/javaknight1/servicepro/backend/pkg/clients/sms"
	storageclient "github.com/javaknight1/servicepro/backend/pkg/clients/storage"
	"github.com/javaknight1/servicepro/backend/pkg/secretbox"
)

// SetupServices initializes all service instances
func SetupServices(
	db *gorm.DB,
	redisClient *redis.Client,
	repos *Repositories,
	cfg *config.Config,
	emailClient emailclient.Client,
	storageClient storageclient.Client,
	smsClient smsclient.Client,
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
	svc.Import = services.NewImportService(db, repos.Customer)

	// Job service
	svc.Job = services.NewJobService(repos.Job, repos.Customer, db)

	// Customer sites (T186)
	svc.Sites = services.NewSiteService(db, repos.Sites)

	// Customer purchase orders with amounts (T187)
	svc.PurchaseOrders = services.NewPurchaseOrderService(db, repos.PurchaseOrders)

	// Approver OKs for extra work over a customer's not-to-exceed limit (T187)
	svc.JobApprovals = services.NewJobApprovalService(db, repos.JobApprovals)

	// Service agreements (T167)
	svc.Agreements = services.NewAgreementService(db, repos.Agreements)

	// Accepted quote -> job (T028)
	svc.QuoteConversion = services.NewQuoteConversionService(db)

	// Document PDF service (for generating quote/invoice/receipt PDFs)
	// Without a storage client (production until R2, T160) PDFs are generated
	// on demand and attached to emails, just not cached or linked (T256).
	var documentPDFService *services.DocumentPDFService
	pdfConfig := pdf.DefaultConfig()
	pdfConfig.Storage.Type = "local" // Use local storage for temp files, upload to S3 via storage client
	pdfConfig.Storage.LocalPath = "./exports/pdf"
	pdfGenerator, err := pdf.NewGenerator(context.Background(), pdfConfig, nil)
	if err != nil {
		log.Printf("[SERVICES] Failed to create PDF generator: %v", err)
	} else {
		documentPDFService = services.NewDocumentPDFService(pdfGenerator, storageClient)
		if storageClient == nil {
			log.Println("[SERVICES] Document PDF service initialized without storage: PDFs generated on demand, not cached")
		} else {
			log.Println("[SERVICES] Document PDF service initialized")
		}
	}

	// Quote service (with PDF and email for sending quotes)
	svc.Quote = services.NewQuoteServiceWithConfig(repos.Quote, &services.QuoteServiceConfig{
		PDFService:  documentPDFService,
		EmailClient: emailClient,
		DB:          db,
	})

	// Stripe Connect: each tenant's own Stripe account (T169). Without a
	// Stripe key the service exists but refuses to connect.
	var connectAPI stripeService.ConnectAPI
	if cfg.Stripe.SecretKey != "" {
		connectAPI = stripeService.NewConnectClient(cfg.Stripe.SecretKey)
	}
	svc.PaymentsConnect = services.NewPaymentsConnectService(services.PaymentsConnectServiceConfig{
		DB:              db,
		Repo:            repos.ConnectedAccounts,
		API:             connectAPI,
		FrontendURL:     cfg.Server.FrontendURL,
		PaymentsEnabled: cfg.Stripe.InvoicePaymentsEnabled,
	})

	// Invoice service (with email client and PDF service for sending invoices)
	svc.Invoice = services.NewInvoiceServiceWithConfig(db, &services.InvoiceServiceConfig{
		EmailClient: emailClient,
		PDFService:  documentPDFService,
		FrontendURL: cfg.Server.FrontendURL,
		BackendURL:  cfg.Server.BackendURL,
		// Pay links only when payments are on and Stripe is configured (T261)...
		OnlinePayments: cfg.Stripe.InvoicePaymentsEnabled && cfg.Stripe.SecretKey != "",
	})
	// ...and the invoice's tenant can take payments on its own account (T169).
	svc.Invoice.SetPayLinkPolicy(svc.PaymentsConnect)

	// QuickBooks Online sync (T192): off (status "not configured") unless
	// the Intuit app keys are set. Sent invoices and recorded payments are
	// queued for the tenant's QuickBooks when it is connected.
	svc.QuickBooks = setupQuickBooks(cfg, db, permChecker)
	if svc.QuickBooks.Configured() {
		svc.Invoice.SetAccountingSync(svc.QuickBooks)
	}

	// Revenue report service
	svc.Revenue = services.NewRevenueReportService(db)

	// A/R Aging report service
	svc.ARAging = services.NewARAgingService(db)

	// Export service
	// Export files go to object storage so any instance can serve the
	// download (T128); local ./exports only without a storage client (dev).
	var exportFiles services.ExportFileStore
	if storageClient != nil {
		exportFiles = services.NewObjectExportStore(storageClient)
	} else {
		log.Println("[SERVICES] Storage client not configured: exports are stored on local disk (development only)")
		exportFiles = services.NewLocalExportStore("./exports")
	}
	svc.Export = services.NewUnifiedExportService(db, pdfGenerator, exportFiles, "")

	// Payment terms service
	svc.PaymentTerms = services.NewPaymentTermsService(db)
	svc.TaxRates = services.NewTaxRateService(db)

	// Conflict detection service
	svc.Conflict = conflict.NewConflictDetector(repos.Schedule, repos.Job)

	// Profile picture service (optional - depends on storage client)
	if storageClient != nil {
		svc.ProfilePicture = services.NewProfilePictureService(repos.User, repos.Tenant, storageClient)
		// Job photos and files (T174)
		svc.JobAttachments = services.NewJobAttachmentService(repos.JobAttachments, storageClient)
		// Customer sign-off and the emailed service report (T176)
		svc.JobSignoffs = services.NewJobSignoffService(repos.JobSignoffs, storageClient, pdfGenerator, emailClient)
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
			// Platform billing (tenant subscriptions) shares this endpoint (T122).
			services.RegisterBillingWebhookHandlers(svc.StripeEvents, svc.Stripe)
			svc.StripeWebhook = stripeService.NewWebhookHandler(cfg, svc.StripeEvents)
			// Persistent dedupe across restarts/instances (F-10).
			eventStore := stripeService.NewGormEventStore(db)
			svc.StripeWebhook.SetEventStore(eventStore)

			// Connect (T169): events from connected accounts arrive on their
			// own endpoint and secret, through the same processor and dedupe
			// store; v2 account events arrive as thin events.
			if cfg.Stripe.ConnectWebhookSecret != "" {
				svc.StripeConnectWebhook = stripeService.NewWebhookHandlerWithSecret(cfg.Stripe.ConnectWebhookSecret, svc.StripeEvents)
				svc.StripeConnectWebhook.SetEventStore(eventStore)
			}
			if cfg.Stripe.V2WebhookSecret != "" {
				connectSvc := svc.PaymentsConnect
				svc.StripeV2Webhook = stripeService.NewThinEventHandler(cfg.Stripe.V2WebhookSecret,
					func(ctx context.Context, ev *stripeService.ThinEvent) error {
						if !stripeService.IsAccountEvent(ev) {
							return nil
						}
						return connectSvc.SyncAccount(ctx, ev.RelatedObject.ID)
					})
				svc.StripeV2Webhook.SetEventStore(eventStore)
			}
			services.RegisterConnectWebhookHandlers(svc.StripeEvents, svc.PaymentsConnect)

			// Invoice payment service (for Stripe checkout)
			invoicePaymentSvc, err := services.NewInvoicePaymentService(&services.InvoicePaymentServiceConfig{
				StripeClient:   client,
				InvoiceService: svc.Invoice,
				SuccessURL:     cfg.Server.FrontendURL + "/invoices/payment-success",
				CancelURL:      cfg.Server.FrontendURL + "/invoices/payment-cancelled",
				Accounts:       svc.PaymentsConnect,
				Plans:          repos.Membership,
				Enabled:        cfg.Stripe.InvoicePaymentsEnabled,
			})
			if err != nil {
				log.Printf("[SERVICES] Failed to create invoice payment service: %v", err)
			} else {
				svc.InvoicePayment = invoicePaymentSvc

				// Register checkout.session.completed handler for invoice payments
				invoicePaymentHandler := services.NewInvoicePaymentHandlerAdapter(svc.Invoice)
				invoicePaymentHandler.SetConnectedAccounts(svc.PaymentsConnect)
				svc.StripeEvents.RegisterHandler(
					stripeService.EventCheckoutSessionCompleted,
					invoicePaymentHandler.CreateCheckoutCompletedHandler(),
				)
				// ACH: the session completes unpaid; the payment lands later.
				svc.StripeEvents.RegisterHandler(
					stripeService.EventCheckoutSessionAsyncPaymentSucceeded,
					invoicePaymentHandler.CreateCheckoutCompletedHandler(),
				)
				// ACH that fails clears the invoice's payment pending (T170).
				svc.StripeEvents.RegisterHandler(
					stripeService.EventCheckoutSessionAsyncPaymentFailed,
					invoicePaymentHandler.CreateCheckoutAsyncFailedHandler(),
				)
				// Refunds lower the invoice's amount_paid (T125)
				svc.StripeEvents.RegisterHandler(
					stripeService.EventChargeRefunded,
					invoicePaymentHandler.CreateChargeRefundedHandler(),
				)
			}
		}
	}

	// Membership service
	svc.Membership = services.NewMembershipService(repos.Membership, svc.Stripe)
	// Plan Checkout + founding offer on the platform account (T210, D15).
	svc.Membership.SetPlanCheckout(services.NewPlanCheckoutServiceFromStripe(
		repos.Membership, svc.Stripe, cfg.Stripe.SecretKey, cfg.Stripe.InvoicePaymentsEnabled, cfg.Server.FrontendURL))

	// Tenant service
	svc.Tenant = services.NewTenantService(repos.Tenant, repos.User)
	svc.Tenant.SetPermissionInvalidator(permChecker)

	// Invitation service
	svc.Invitation = services.NewInvitationService(db, emailClient, cfg.Server.FrontendURL)

	// Platform invite codes (REGISTRATION_MODE=invite_only)
	svc.InviteCodes = services.NewInviteCodeService(db)

	// First Neon Auth sign-in → ServicePro user, same invite-only gate as /auth/register (T158)
	svc.NeonProvisioning = services.NewNeonProvisioningService(db, &services.RegistrationGate{
		Mode:        cfg.Auth.RegistrationMode,
		InviteCodes: svc.InviteCodes,
		Invitations: svc.Invitation,
	}, svc.Invitation)

	// Payment reminder service
	svc.PaymentReminder = services.NewPaymentReminderService(db, &services.PaymentReminderServiceConfig{
		EmailClient: emailClient,
		SMSClient:   smsClient, // nil when no SMS provider is configured: reminders fall back to email
		FrontendURL: cfg.Server.FrontendURL,
		BackendURL:  cfg.Server.BackendURL,
	})

	return svc
}

// setupQuickBooks builds the QuickBooks service (T192). Without Intuit keys
// (or QUICKBOOKS_FAKE in development), or without a valid
// INTEGRATIONS_ENCRYPTION_KEY outside development, it is the unconfigured
// service: the integration shows "Coming soon" and the server still starts.
func setupQuickBooks(cfg *config.Config, db *gorm.DB, perms quickbooks.PermissionChecker) *quickbooks.Service {
	ctx := context.Background()
	off := quickbooks.NewService(quickbooks.Config{DB: db})
	isDev := cfg.Server.Env == "development"
	if !cfg.QuickBooks.Enabled(cfg.Server.Env) {
		return off
	}
	box, err := secretbox.FromBase64(cfg.IntegrationsEncryptionKey)
	if err != nil {
		if !isDev {
			logging.Warn(ctx, "[QUICKBOOKS] INTEGRATIONS_ENCRYPTION_KEY missing or invalid: QuickBooks stays off", map[string]any{"source": "quickbooks"})
			return off
		}
		// Development only: a per-process key (stored tokens stop opening
		// after a restart; reconnect).
		if box, err = secretbox.Random(); err != nil {
			return off
		}
		logging.Warn(ctx, "[QUICKBOOKS] No INTEGRATIONS_ENCRYPTION_KEY; using a random development key", map[string]any{"source": "quickbooks"})
	}
	var client qboclient.Client
	env := cfg.QuickBooks.Environment
	fake := cfg.QuickBooks.Fake && isDev && cfg.QuickBooks.ClientID == ""
	if fake {
		client = qboclient.NewFake(cfg.QuickBooks.RedirectURI)
		env = "sandbox"
	} else {
		client = qboclient.NewHTTPClient(qboclient.Config{
			ClientID: cfg.QuickBooks.ClientID, ClientSecret: cfg.QuickBooks.ClientSecret,
			RedirectURI: cfg.QuickBooks.RedirectURI, Environment: env,
		})
	}
	logging.Info(ctx, "[QUICKBOOKS] QuickBooks sync enabled", map[string]any{"source": "quickbooks", "environment": env, "fake": fake})
	return quickbooks.NewService(quickbooks.Config{DB: db, Client: client, Box: box, Environment: env, Perms: perms})
}
