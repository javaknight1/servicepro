// =============================================================================
// Navigation Commands
// =============================================================================

/**
 * Navigate to a specific page by name
 */
Cypress.Commands.add('navigateTo', (page: string) => {
  const routes: Record<string, string> = {
    // Auth pages
    login: '/login',
    register: '/register',
    'forgot-password': '/forgot-password',
    'reset-password': '/reset-password',

    // Main pages
    dashboard: '/dashboard',
    home: '/',

    // Customer pages
    customers: '/customers',
    'customer-create': '/customers/new',

    // Job pages
    jobs: '/jobs',
    'job-create': '/jobs/new',
    schedule: '/schedule',
    calendar: '/calendar',

    // Quote pages
    quotes: '/quotes',
    'quote-create': '/quotes/new',

    // Invoice pages
    invoices: '/invoices',
    'invoice-create': '/invoices/new',

    // Payment pages
    payments: '/payments',

    // Settings pages
    settings: '/settings',
    'settings-profile': '/settings/profile',
    'settings-company': '/settings/company',
    'settings-users': '/settings/users',
    'settings-roles': '/settings/roles',
    'settings-templates': '/settings/templates',
    'settings-integrations': '/settings/integrations',

    // Reports
    reports: '/reports',
    'reports-revenue': '/reports/revenue',
  };

  const url = routes[page.toLowerCase()];
  if (!url) {
    throw new Error(
      `Unknown page: ${page}. Available pages: ${Object.keys(routes).join(
        ', '
      )}`
    );
  }

  cy.visit(url);
  cy.waitForPageLoad();
});

/**
 * Navigate to customer detail page
 */
Cypress.Commands.add('goToCustomer', (customerId: string) => {
  cy.visit(`/customers/${customerId}`);
  cy.waitForPageLoad();
});

/**
 * Navigate to job detail page
 */
Cypress.Commands.add('goToJob', (jobId: string) => {
  cy.visit(`/jobs/${jobId}`);
  cy.waitForPageLoad();
});

/**
 * Navigate to quote detail page
 */
Cypress.Commands.add('goToQuote', (quoteId: string) => {
  cy.visit(`/quotes/${quoteId}`);
  cy.waitForPageLoad();
});

/**
 * Navigate to invoice detail page
 */
Cypress.Commands.add('goToInvoice', (invoiceId: string) => {
  cy.visit(`/invoices/${invoiceId}`);
  cy.waitForPageLoad();
});

/**
 * Click navigation menu item
 */
Cypress.Commands.add('clickNavItem', (itemName: string) => {
  cy.get('[data-testid="sidebar"], [data-testid="nav-menu"], nav')
    .contains(itemName)
    .click();
  cy.waitForPageLoad();
});

/**
 * Open user menu/dropdown
 */
Cypress.Commands.add('openUserMenu', () => {
  cy.get(
    '[data-testid="user-menu"], [data-testid="avatar"], [data-testid="profile-dropdown"]'
  )
    .first()
    .click();
});

/**
 * Assert current URL contains path
 */
Cypress.Commands.add('assertUrl', (path: string) => {
  cy.url().should('include', path);
});

/**
 * Assert page title
 */
Cypress.Commands.add('assertPageTitle', (title: string) => {
  cy.get('h1, [data-testid="page-title"]')
    .first()
    .should('contain.text', title);
});

/**
 * Go back in browser history
 */
Cypress.Commands.add('goBack', () => {
  cy.go('back');
  cy.waitForPageLoad();
});

/**
 * Refresh current page
 */
Cypress.Commands.add('refreshPage', () => {
  cy.reload();
  cy.waitForPageLoad();
});

/**
 * Assert breadcrumb contains text
 */
Cypress.Commands.add('assertBreadcrumb', (text: string) => {
  cy.get(
    '[data-testid="breadcrumb"], nav[aria-label="breadcrumb"], .breadcrumb'
  ).should('contain.text', text);
});

/**
 * Click breadcrumb link
 */
Cypress.Commands.add('clickBreadcrumb', (text: string) => {
  cy.get(
    '[data-testid="breadcrumb"], nav[aria-label="breadcrumb"], .breadcrumb'
  )
    .contains('a', text)
    .click();
  cy.waitForPageLoad();
});

// Add type declarations
declare global {
  namespace Cypress {
    interface Chainable {
      navigateTo(page: string): Chainable<void>;
      goToCustomer(customerId: string): Chainable<void>;
      goToJob(jobId: string): Chainable<void>;
      goToQuote(quoteId: string): Chainable<void>;
      goToInvoice(invoiceId: string): Chainable<void>;
      clickNavItem(itemName: string): Chainable<void>;
      openUserMenu(): Chainable<void>;
      assertUrl(path: string): Chainable<void>;
      assertPageTitle(title: string): Chainable<void>;
      goBack(): Chainable<void>;
      refreshPage(): Chainable<void>;
      assertBreadcrumb(text: string): Chainable<void>;
      clickBreadcrumb(text: string): Chainable<void>;
    }
  }
}

export {};
