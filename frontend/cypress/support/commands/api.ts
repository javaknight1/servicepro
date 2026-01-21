// =============================================================================
// API Commands
// =============================================================================

interface ApiRequestOptions {
  method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE';
  url: string;
  body?: object;
  headers?: Record<string, string>;
  failOnStatusCode?: boolean;
}

/**
 * Make authenticated API request
 */
Cypress.Commands.add('apiRequest', <T>(options: ApiRequestOptions) => {
  const apiUrl = Cypress.env('apiUrl');

  return cy.window().then((win) => {
    const token = win.localStorage.getItem('access_token');

    return cy.request<T>({
      method: options.method,
      url: `${apiUrl}${options.url}`,
      body: options.body,
      headers: {
        Authorization: token ? `Bearer ${token}` : '',
        'Content-Type': 'application/json',
        ...options.headers,
      },
      failOnStatusCode: options.failOnStatusCode ?? true,
    });
  });
});

/**
 * Seed database with test data
 */
Cypress.Commands.add('seedDatabase', (fixtures: string[]) => {
  const apiUrl = Cypress.env('apiUrl');

  // Load fixtures and send to seeding endpoint
  fixtures.forEach((fixtureName) => {
    cy.fixture(fixtureName).then((data) => {
      cy.request({
        method: 'POST',
        url: `${apiUrl}/api/v1/test/seed`,
        body: { fixture: fixtureName, data },
        failOnStatusCode: false,
      });
    });
  });

  // Alternative: Use task if API endpoint not available
  cy.task('seedTestData', { fixtures });
});

/**
 * Clear test data from database
 */
Cypress.Commands.add('clearDatabase', () => {
  const apiUrl = Cypress.env('apiUrl');

  // Call cleanup endpoint
  cy.request({
    method: 'POST',
    url: `${apiUrl}/api/v1/test/clear`,
    failOnStatusCode: false,
  });

  // Alternative: Use task
  cy.task('clearTestData');
});

// =============================================================================
// API Stubbing Helpers
// =============================================================================

/**
 * Stub API endpoint with fixture data
 */
Cypress.Commands.add(
  'stubApi',
  (method: string, urlPattern: string, fixture: string, alias: string) => {
    cy.fixture(fixture).then((data) => {
      cy.intercept(method, urlPattern, {
        statusCode: 200,
        body: data,
      }).as(alias);
    });
  }
);

/**
 * Stub API endpoint with inline response
 */
Cypress.Commands.add(
  'stubApiResponse',
  (
    method: string,
    urlPattern: string,
    response: object,
    alias: string,
    statusCode = 200
  ) => {
    cy.intercept(method, urlPattern, {
      statusCode,
      body: response,
    }).as(alias);
  }
);

/**
 * Stub API error response
 */
Cypress.Commands.add(
  'stubApiError',
  (
    method: string,
    urlPattern: string,
    statusCode: number,
    errorMessage: string,
    alias: string
  ) => {
    cy.intercept(method, urlPattern, {
      statusCode,
      body: {
        error: true,
        message: errorMessage,
      },
    }).as(alias);
  }
);

/**
 * Intercept and spy on API calls without modifying response
 */
Cypress.Commands.add(
  'spyOnApi',
  (method: string, urlPattern: string, alias: string) => {
    cy.intercept(method, urlPattern).as(alias);
  }
);

// =============================================================================
// Common API Stubs
// =============================================================================

/**
 * Set up common API stubs for testing
 */
Cypress.Commands.add('setupApiStubs', () => {
  // Health check
  cy.intercept('GET', '**/api/health', {
    statusCode: 200,
    body: { status: 'ok' },
  }).as('healthCheck');

  // User profile
  cy.intercept('GET', '**/api/v1/users/me', { fixture: 'user.json' }).as(
    'getProfile'
  );

  // Common lists
  cy.intercept('GET', '**/api/v1/customers*', { fixture: 'customers.json' }).as(
    'getCustomers'
  );
  cy.intercept('GET', '**/api/v1/jobs*', { fixture: 'jobs.json' }).as(
    'getJobs'
  );
  cy.intercept('GET', '**/api/v1/invoices*', { fixture: 'invoices.json' }).as(
    'getInvoices'
  );
});

// =============================================================================
// Network Delay Simulation
// =============================================================================

/**
 * Simulate slow network for API calls
 */
Cypress.Commands.add('simulateSlowNetwork', (delayMs = 2000) => {
  cy.intercept('**/api/**', (req) => {
    req.on('response', (res) => {
      res.setDelay(delayMs);
    });
  }).as('slowNetwork');
});

/**
 * Simulate network failure
 */
Cypress.Commands.add(
  'simulateNetworkFailure',
  (urlPattern: string, alias: string) => {
    cy.intercept(urlPattern, { forceNetworkError: true }).as(alias);
  }
);

// Add type declarations
declare global {
  namespace Cypress {
    interface Chainable {
      stubApi(
        method: string,
        urlPattern: string,
        fixture: string,
        alias: string
      ): Chainable<void>;
      stubApiResponse(
        method: string,
        urlPattern: string,
        response: object,
        alias: string,
        statusCode?: number
      ): Chainable<void>;
      stubApiError(
        method: string,
        urlPattern: string,
        statusCode: number,
        errorMessage: string,
        alias: string
      ): Chainable<void>;
      spyOnApi(
        method: string,
        urlPattern: string,
        alias: string
      ): Chainable<void>;
      setupApiStubs(): Chainable<void>;
      simulateSlowNetwork(delayMs?: number): Chainable<void>;
      simulateNetworkFailure(
        urlPattern: string,
        alias: string
      ): Chainable<void>;
    }
  }
}

export {};
