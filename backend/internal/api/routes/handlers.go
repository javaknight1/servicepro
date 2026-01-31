package routes

import (
	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/internal/api/handlers"
)

// SetupHandlers initializes all handler instances
func SetupHandlers(
	cfg *config.Config,
	repos *Repositories,
	svc *Services,
	mw *Middleware,
) *Handlers {
	h := &Handlers{}

	// Auth handler (with cookie manager for httpOnly token storage)
	h.Auth = handlers.NewAuthHandler(svc.Auth, mw.RateLimiter, mw.CookieManager)

	// Registration handler
	h.Registration = handlers.NewRegistrationHandler(svc.Registration, svc.Invitation)

	// Email verification handler
	h.EmailVerification = handlers.NewEmailVerificationHandler(svc.EmailVerification)

	// Password reset handler
	h.PasswordReset = handlers.NewPasswordResetHandler(svc.PasswordReset)

	// User handler
	h.User = handlers.NewUserHandler(repos.User)

	// Customer handler
	h.Customer = handlers.NewCustomerHandler(repos.Customer)

	// Customer report handler
	h.CustomerReport = handlers.NewCustomerReportHandler(svc.Customer)

	// Job handler
	h.Job = handlers.NewJobHandler(svc.Job)

	// Quote handler
	h.Quote = handlers.NewQuoteHandler(svc.Quote)

	// Invoice handler
	h.Invoice = handlers.NewInvoiceHandler(svc.Invoice)

	// Public invoice handler (for payment links - optional, depends on InvoicePayment service)
	if svc.InvoicePayment != nil {
		h.PublicInvoice = handlers.NewPublicInvoiceHandler(svc.InvoicePayment, svc.Invoice)
	}

	// Tenant handler
	h.Tenant = handlers.NewTenantHandler(svc.Tenant)

	// Invitation handler
	h.Invitation = handlers.NewInvitationHandler(svc.Invitation)

	// Role handler
	h.Role = handlers.NewRoleHandler(repos.Permission)

	// Revenue handler
	h.Revenue = handlers.NewRevenueHandler(svc.Revenue)

	// Import/Export handler
	h.ImportExport = handlers.NewImportExportHandler(svc.Import)

	// Export handler
	h.Export = handlers.NewExportHandler(svc.Export)

	// Profile picture handler (optional - depends on service availability)
	if svc.ProfilePicture != nil {
		h.ProfilePicture = handlers.NewProfilePictureHandler(svc.ProfilePicture)
	}

	// Payment terms handler
	h.PaymentTerms = handlers.NewPaymentTermsHandler(svc.PaymentTerms)

	// Stripe handlers (optional - depends on Stripe configuration)
	if svc.StripeClient != nil && svc.StripeWebhook != nil {
		h.Stripe = handlers.NewStripeHandler(svc.StripeClient, svc.StripeWebhook)
	}

	// Billing handler (optional - depends on Stripe service)
	if svc.Stripe != nil {
		h.Billing = handlers.NewBillingHandler(svc.Stripe)
		h.Webhook = handlers.NewWebhookHandler(svc.Stripe, cfg.Stripe.WebhookSecret, repos.Tenant)
	}

	// Membership handler
	h.Membership = handlers.NewMembershipHandler(svc.Membership)

	return h
}
