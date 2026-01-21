package permissions

// Resources defines all resource names in the system
const (
	ResourceCustomers    = "customers"
	ResourceJobs         = "jobs"
	ResourceQuotes       = "quotes"
	ResourceInvoices     = "invoices"
	ResourcePayments     = "payments"
	ResourcePaymentTerms = "payment_terms"
	ResourceReports      = "reports"
	ResourceExports      = "exports"
	ResourceTenants      = "tenants"
	ResourceUsers        = "users"
	ResourceRoles        = "roles"
	ResourceSettings     = "settings"
)

// Actions defines all action types in the system
const (
	ActionCreate  = "create"
	ActionRead    = "read"
	ActionUpdate  = "update"
	ActionDelete  = "delete"
	ActionList    = "list"
	ActionAssign  = "assign"
	ActionApprove = "approve"
	ActionExecute = "execute"
	ActionView    = "view"
	ActionExport  = "export"
	ActionAdmin   = "admin"
	ActionManage  = "manage"
)

// Customer permissions
const (
	CustomersCreate = "customers.create"
	CustomersRead   = "customers.read"
	CustomersUpdate = "customers.update"
	CustomersDelete = "customers.delete"
	CustomersList   = "customers.list"
)

// Job permissions
const (
	JobsCreate = "jobs.create"
	JobsRead   = "jobs.read"
	JobsUpdate = "jobs.update"
	JobsDelete = "jobs.delete"
	JobsList   = "jobs.list"
	JobsAssign = "jobs.assign"
)

// Quote permissions
const (
	QuotesCreate  = "quotes.create"
	QuotesRead    = "quotes.read"
	QuotesUpdate  = "quotes.update"
	QuotesDelete  = "quotes.delete"
	QuotesList    = "quotes.list"
	QuotesApprove = "quotes.approve"
)

// Invoice permissions
const (
	InvoicesCreate  = "invoices.create"
	InvoicesRead    = "invoices.read"
	InvoicesUpdate  = "invoices.update"
	InvoicesDelete  = "invoices.delete"
	InvoicesList    = "invoices.list"
	InvoicesExecute = "invoices.execute" // For sending invoices
)

// Payment permissions
const (
	PaymentsCreate = "payments.create"
	PaymentsRead   = "payments.read"
	PaymentsList   = "payments.list"
	PaymentsRefund = "payments.refund"
)

// Payment Terms permissions
const (
	PaymentTermsCreate = "payment_terms.create"
	PaymentTermsRead   = "payment_terms.read"
	PaymentTermsUpdate = "payment_terms.update"
	PaymentTermsDelete = "payment_terms.delete"
	PaymentTermsList   = "payment_terms.list"
)

// Report permissions
const (
	ReportsView   = "reports.view"
	ReportsExport = "reports.export"
	ReportsAdmin  = "reports.admin"
)

// Export permissions
const (
	ExportsCreate = "exports.create"
	ExportsRead   = "exports.read"
	ExportsList   = "exports.list"
)

// Tenant permissions
const (
	TenantsCreate = "tenants.create"
	TenantsRead   = "tenants.read"
	TenantsUpdate = "tenants.update"
	TenantsDelete = "tenants.delete"
	TenantsList   = "tenants.list"
	TenantsManage = "tenants.manage"
)

// User permissions
const (
	UsersCreate = "users.create"
	UsersRead   = "users.read"
	UsersUpdate = "users.update"
	UsersDelete = "users.delete"
	UsersList   = "users.list"
)

// Role permissions
const (
	RolesCreate = "roles.create"
	RolesRead   = "roles.read"
	RolesUpdate = "roles.update"
	RolesDelete = "roles.delete"
	RolesAssign = "roles.assign"
)

// Settings permissions
const (
	SettingsRead   = "settings.read"
	SettingsUpdate = "settings.update"
)

// PermissionDefinition represents a permission with its metadata
type PermissionDefinition struct {
	Name        string
	Description string
	Resource    string
	Action      string
}

// AllPermissions returns all permission definitions for database seeding
func AllPermissions() []PermissionDefinition {
	return []PermissionDefinition{
		// Customers
		{CustomersCreate, "Create new customers", ResourceCustomers, ActionCreate},
		{CustomersRead, "View customer information", ResourceCustomers, ActionRead},
		{CustomersUpdate, "Update customer information", ResourceCustomers, ActionUpdate},
		{CustomersDelete, "Delete customers", ResourceCustomers, ActionDelete},
		{CustomersList, "List all customers", ResourceCustomers, ActionList},

		// Jobs
		{JobsCreate, "Create new jobs", ResourceJobs, ActionCreate},
		{JobsRead, "View job information", ResourceJobs, ActionRead},
		{JobsUpdate, "Update job information", ResourceJobs, ActionUpdate},
		{JobsDelete, "Delete jobs", ResourceJobs, ActionDelete},
		{JobsList, "List all jobs", ResourceJobs, ActionList},
		{JobsAssign, "Assign technicians to jobs", ResourceJobs, ActionAssign},

		// Quotes
		{QuotesCreate, "Create new quotes", ResourceQuotes, ActionCreate},
		{QuotesRead, "View quote information", ResourceQuotes, ActionRead},
		{QuotesUpdate, "Update quote information", ResourceQuotes, ActionUpdate},
		{QuotesDelete, "Delete quotes", ResourceQuotes, ActionDelete},
		{QuotesList, "List all quotes", ResourceQuotes, ActionList},
		{QuotesApprove, "Approve or reject quotes", ResourceQuotes, ActionApprove},

		// Invoices
		{InvoicesCreate, "Create new invoices", ResourceInvoices, ActionCreate},
		{InvoicesRead, "View invoice information", ResourceInvoices, ActionRead},
		{InvoicesUpdate, "Update invoice information", ResourceInvoices, ActionUpdate},
		{InvoicesDelete, "Delete invoices", ResourceInvoices, ActionDelete},
		{InvoicesList, "List all invoices", ResourceInvoices, ActionList},
		{InvoicesExecute, "Send invoices to customers", ResourceInvoices, ActionExecute},

		// Payments
		{PaymentsCreate, "Record new payments", ResourcePayments, ActionCreate},
		{PaymentsRead, "View payment information", ResourcePayments, ActionRead},
		{PaymentsList, "List all payments", ResourcePayments, ActionList},
		{PaymentsRefund, "Process payment refunds", ResourcePayments, ActionAdmin},

		// Payment Terms
		{PaymentTermsCreate, "Create payment terms", ResourcePaymentTerms, ActionCreate},
		{PaymentTermsRead, "View payment terms", ResourcePaymentTerms, ActionRead},
		{PaymentTermsUpdate, "Update payment terms", ResourcePaymentTerms, ActionUpdate},
		{PaymentTermsDelete, "Delete payment terms", ResourcePaymentTerms, ActionDelete},
		{PaymentTermsList, "List payment terms", ResourcePaymentTerms, ActionList},

		// Reports
		{ReportsView, "View reports", ResourceReports, ActionView},
		{ReportsExport, "Export reports", ResourceReports, ActionExport},
		{ReportsAdmin, "Administer report settings", ResourceReports, ActionAdmin},

		// Exports
		{ExportsCreate, "Create data exports", ResourceExports, ActionCreate},
		{ExportsRead, "View export status", ResourceExports, ActionRead},
		{ExportsList, "List all exports", ResourceExports, ActionList},

		// Tenants
		{TenantsCreate, "Create new organizations", ResourceTenants, ActionCreate},
		{TenantsRead, "View organization information", ResourceTenants, ActionRead},
		{TenantsUpdate, "Update organization settings", ResourceTenants, ActionUpdate},
		{TenantsDelete, "Delete organizations", ResourceTenants, ActionDelete},
		{TenantsList, "List all organizations", ResourceTenants, ActionList},
		{TenantsManage, "Manage organization members", ResourceTenants, ActionManage},

		// Users
		{UsersCreate, "Create new users", ResourceUsers, ActionCreate},
		{UsersRead, "View user information", ResourceUsers, ActionRead},
		{UsersUpdate, "Update user information", ResourceUsers, ActionUpdate},
		{UsersDelete, "Delete users", ResourceUsers, ActionDelete},
		{UsersList, "List all users", ResourceUsers, ActionList},

		// Roles
		{RolesCreate, "Create new roles", ResourceRoles, ActionCreate},
		{RolesRead, "View role information", ResourceRoles, ActionRead},
		{RolesUpdate, "Update role information", ResourceRoles, ActionUpdate},
		{RolesDelete, "Delete roles", ResourceRoles, ActionDelete},
		{RolesAssign, "Assign roles to users", ResourceRoles, ActionAssign},

		// Settings
		{SettingsRead, "View system settings", ResourceSettings, ActionRead},
		{SettingsUpdate, "Update system settings", ResourceSettings, ActionUpdate},
	}
}

// RolePermissions defines the default permission sets for each role
type RolePermissions struct {
	RoleName    string
	Permissions []string
}

// DefaultRolePermissions returns the default permissions for each role
func DefaultRolePermissions() []RolePermissions {
	return []RolePermissions{
		{
			RoleName: "super_admin",
			Permissions: []string{
				// All permissions
				CustomersCreate, CustomersRead, CustomersUpdate, CustomersDelete, CustomersList,
				JobsCreate, JobsRead, JobsUpdate, JobsDelete, JobsList, JobsAssign,
				QuotesCreate, QuotesRead, QuotesUpdate, QuotesDelete, QuotesList, QuotesApprove,
				InvoicesCreate, InvoicesRead, InvoicesUpdate, InvoicesDelete, InvoicesList, InvoicesExecute,
				PaymentsCreate, PaymentsRead, PaymentsList, PaymentsRefund,
				PaymentTermsCreate, PaymentTermsRead, PaymentTermsUpdate, PaymentTermsDelete, PaymentTermsList,
				ReportsView, ReportsExport, ReportsAdmin,
				ExportsCreate, ExportsRead, ExportsList,
				TenantsCreate, TenantsRead, TenantsUpdate, TenantsDelete, TenantsList, TenantsManage,
				UsersCreate, UsersRead, UsersUpdate, UsersDelete, UsersList,
				RolesCreate, RolesRead, RolesUpdate, RolesDelete, RolesAssign,
				SettingsRead, SettingsUpdate,
			},
		},
		{
			RoleName: "admin",
			Permissions: []string{
				// All except tenant deletion and some role management
				CustomersCreate, CustomersRead, CustomersUpdate, CustomersDelete, CustomersList,
				JobsCreate, JobsRead, JobsUpdate, JobsDelete, JobsList, JobsAssign,
				QuotesCreate, QuotesRead, QuotesUpdate, QuotesDelete, QuotesList, QuotesApprove,
				InvoicesCreate, InvoicesRead, InvoicesUpdate, InvoicesDelete, InvoicesList, InvoicesExecute,
				PaymentsCreate, PaymentsRead, PaymentsList, PaymentsRefund,
				PaymentTermsCreate, PaymentTermsRead, PaymentTermsUpdate, PaymentTermsDelete, PaymentTermsList,
				ReportsView, ReportsExport, ReportsAdmin,
				ExportsCreate, ExportsRead, ExportsList,
				TenantsCreate, TenantsRead, TenantsUpdate, TenantsList, TenantsManage,
				UsersCreate, UsersRead, UsersUpdate, UsersDelete, UsersList,
				RolesRead, RolesAssign,
				SettingsRead, SettingsUpdate,
			},
		},
		{
			RoleName: "manager",
			Permissions: []string{
				// CRUD on business resources, reports view
				CustomersCreate, CustomersRead, CustomersUpdate, CustomersList,
				JobsCreate, JobsRead, JobsUpdate, JobsList, JobsAssign,
				QuotesCreate, QuotesRead, QuotesUpdate, QuotesList, QuotesApprove,
				InvoicesCreate, InvoicesRead, InvoicesUpdate, InvoicesList, InvoicesExecute,
				PaymentsCreate, PaymentsRead, PaymentsList,
				PaymentTermsRead, PaymentTermsList,
				ReportsView, ReportsExport,
				ExportsCreate, ExportsRead, ExportsList,
				TenantsRead,
				UsersRead, UsersList,
				RolesRead,
				SettingsRead,
			},
		},
		{
			RoleName: "user",
			Permissions: []string{
				// Standard user access
				CustomersRead, CustomersList,
				JobsRead, JobsList,
				QuotesRead, QuotesList,
				InvoicesRead, InvoicesList,
				PaymentsRead, PaymentsList,
				PaymentTermsRead, PaymentTermsList,
				ReportsView,
				ExportsRead,
				TenantsRead,
				UsersRead,
				RolesRead,
				SettingsRead,
			},
		},
		{
			RoleName: "guest",
			Permissions: []string{
				// Minimal read-only access
				CustomersRead,
				JobsRead,
				QuotesRead,
				InvoicesRead,
				PaymentTermsRead,
				TenantsRead,
				SettingsRead,
			},
		},
	}
}

// PermissionsByResource returns all permissions grouped by resource
func PermissionsByResource() map[string][]string {
	return map[string][]string{
		ResourceCustomers:    {CustomersCreate, CustomersRead, CustomersUpdate, CustomersDelete, CustomersList},
		ResourceJobs:         {JobsCreate, JobsRead, JobsUpdate, JobsDelete, JobsList, JobsAssign},
		ResourceQuotes:       {QuotesCreate, QuotesRead, QuotesUpdate, QuotesDelete, QuotesList, QuotesApprove},
		ResourceInvoices:     {InvoicesCreate, InvoicesRead, InvoicesUpdate, InvoicesDelete, InvoicesList, InvoicesExecute},
		ResourcePayments:     {PaymentsCreate, PaymentsRead, PaymentsList, PaymentsRefund},
		ResourcePaymentTerms: {PaymentTermsCreate, PaymentTermsRead, PaymentTermsUpdate, PaymentTermsDelete, PaymentTermsList},
		ResourceReports:      {ReportsView, ReportsExport, ReportsAdmin},
		ResourceExports:      {ExportsCreate, ExportsRead, ExportsList},
		ResourceTenants:      {TenantsCreate, TenantsRead, TenantsUpdate, TenantsDelete, TenantsList, TenantsManage},
		ResourceUsers:        {UsersCreate, UsersRead, UsersUpdate, UsersDelete, UsersList},
		ResourceRoles:        {RolesCreate, RolesRead, RolesUpdate, RolesDelete, RolesAssign},
		ResourceSettings:     {SettingsRead, SettingsUpdate},
	}
}
