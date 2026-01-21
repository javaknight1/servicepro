// =============================================================================
// Cypress E2E Support File
// =============================================================================
// This file is processed and loaded automatically before test files.

// Import commands
import './commands';
import './commands/auth';
import './commands/api';
import './commands/navigation';
import './commands/forms';

// =============================================================================
// Global Configuration
// =============================================================================

// Prevent TypeScript errors on custom commands
declare global {
  namespace Cypress {
    interface Chainable {
      // Auth commands
      login(email?: string, password?: string): Chainable<void>;
      loginAsAdmin(): Chainable<void>;
      loginAsUser(): Chainable<void>;
      logout(): Chainable<void>;
      register(userData: RegisterData): Chainable<void>;

      // API commands
      apiLogin(email: string, password: string): Chainable<AuthResponse>;
      apiRequest<T>(options: ApiRequestOptions): Chainable<T>;
      seedDatabase(fixtures: string[]): Chainable<void>;
      clearDatabase(): Chainable<void>;

      // Navigation commands
      visitWithAuth(url: string): Chainable<void>;
      waitForPageLoad(): Chainable<void>;
      navigateTo(page: string): Chainable<void>;

      // Form commands
      fillForm(
        formData: Record<string, string | number | boolean>
      ): Chainable<void>;
      submitForm(): Chainable<void>;
      clearForm(): Chainable<void>;
      selectOption(selector: string, value: string): Chainable<void>;
      checkCheckbox(selector: string): Chainable<void>;
      uncheckCheckbox(selector: string): Chainable<void>;

      // Utility commands
      getByTestId(testId: string): Chainable<JQuery<HTMLElement>>;
      getByRole(
        role: string,
        options?: { name?: string | RegExp }
      ): Chainable<JQuery<HTMLElement>>;
      shouldBeVisible(): Chainable<JQuery<HTMLElement>>;
      shouldNotExist(): Chainable<JQuery<HTMLElement>>;
      waitForApi(alias: string): Chainable<void>;
      assertToast(
        message: string,
        type?: 'success' | 'error' | 'warning'
      ): Chainable<void>;

      // Visual testing
      matchImageSnapshot(name?: string): Chainable<void>;
    }
  }
}

// =============================================================================
// Types
// =============================================================================

interface RegisterData {
  email: string;
  password: string;
  firstName: string;
  lastName: string;
}

interface AuthResponse {
  accessToken: string;
  refreshToken: string;
  user: {
    id: string;
    email: string;
    role: string;
  };
}

interface ApiRequestOptions {
  method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
  url: string;
  body?: object;
  headers?: Record<string, string>;
  failOnStatusCode?: boolean;
}

// =============================================================================
// Global Hooks
// =============================================================================

// Before each test
beforeEach(() => {
  // Clear local storage
  cy.clearLocalStorage();

  // Clear cookies
  cy.clearCookies();

  // Intercept common API calls (if network stubbing enabled)
  if (Cypress.env('enableNetworkStubbing')) {
    // Stub health check
    cy.intercept('GET', '**/api/health', {
      statusCode: 200,
      body: { status: 'ok' },
    }).as('healthCheck');
  }

  // Log test start
  cy.task('log', `Starting test: ${Cypress.currentTest.title}`);
});

// After each test
afterEach(() => {
  // Log test completion
  cy.task('log', `Completed test: ${Cypress.currentTest.title}`);
});

// =============================================================================
// Error Handling
// =============================================================================

// Catch uncaught exceptions
Cypress.on('uncaught:exception', (err, runnable) => {
  // Log the error
  console.error('Uncaught exception:', err.message);

  // Don't fail tests on known third-party errors
  if (
    err.message.includes('ResizeObserver') ||
    err.message.includes('Script error')
  ) {
    return false;
  }

  // Let other errors fail the test
  return true;
});

// =============================================================================
// Console Logging
// =============================================================================

// Log console errors in CI
Cypress.on('window:before:load', (win) => {
  cy.spy(win.console, 'error').as('consoleError');
  cy.spy(win.console, 'warn').as('consoleWarn');
});

// =============================================================================
// Network Logging
// =============================================================================

// Log failed requests
Cypress.on('fail', (error, runnable) => {
  // Add additional debugging info
  console.error('Test failed:', error.message);
  throw error;
});

export {};
