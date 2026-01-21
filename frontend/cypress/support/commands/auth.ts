// =============================================================================
// Authentication Commands
// =============================================================================

/**
 * Login via UI
 */
Cypress.Commands.add('login', (email?: string, password?: string) => {
  const userEmail = email || Cypress.env('testUserEmail');
  const userPassword = password || Cypress.env('testUserPassword');

  cy.session(
    [userEmail, userPassword],
    () => {
      cy.visit('/login');

      // Fill login form
      cy.get(
        'input[name="email"], input[type="email"], [data-testid="email-input"]'
      )
        .should('be.visible')
        .clear()
        .type(userEmail);

      cy.get(
        'input[name="password"], input[type="password"], [data-testid="password-input"]'
      )
        .should('be.visible')
        .clear()
        .type(userPassword);

      // Submit form
      cy.get('button[type="submit"], [data-testid="login-button"]')
        .should('be.visible')
        .click();

      // Wait for redirect to dashboard
      cy.url().should('include', '/dashboard');

      // Verify auth token is stored
      cy.window()
        .its('localStorage')
        .invoke('getItem', 'access_token')
        .should('exist');
    },
    {
      validate() {
        // Check if session is still valid
        cy.window()
          .its('localStorage')
          .invoke('getItem', 'access_token')
          .should('exist');
      },
      cacheAcrossSpecs: true,
    }
  );
});

/**
 * Login as admin user
 */
Cypress.Commands.add('loginAsAdmin', () => {
  const adminEmail = Cypress.env('adminEmail');
  const adminPassword = Cypress.env('adminPassword');

  cy.login(adminEmail, adminPassword);
});

/**
 * Login as regular user
 */
Cypress.Commands.add('loginAsUser', () => {
  const userEmail = Cypress.env('testUserEmail');
  const userPassword = Cypress.env('testUserPassword');

  cy.login(userEmail, userPassword);
});

/**
 * Login via API (faster, for non-auth tests)
 */
Cypress.Commands.add('apiLogin', (email: string, password: string) => {
  const apiUrl = Cypress.env('apiUrl');

  return cy
    .request({
      method: 'POST',
      url: `${apiUrl}/api/v1/auth/login`,
      body: { email, password },
      failOnStatusCode: false,
    })
    .then((response) => {
      if (response.status === 200) {
        const { access_token, refresh_token, user } = response.body;

        // Store tokens
        cy.window().then((win) => {
          win.localStorage.setItem('access_token', access_token);
          win.localStorage.setItem('refresh_token', refresh_token);
          win.localStorage.setItem('user', JSON.stringify(user));
        });

        return {
          accessToken: access_token,
          refreshToken: refresh_token,
          user,
        };
      }
      throw new Error(
        `Login failed: ${response.body.message || 'Unknown error'}`
      );
    });
});

/**
 * Logout
 */
Cypress.Commands.add('logout', () => {
  // Clear storage
  cy.clearLocalStorage();
  cy.clearCookies();

  // Visit logout endpoint or clear session
  cy.window().then((win) => {
    win.localStorage.removeItem('access_token');
    win.localStorage.removeItem('refresh_token');
    win.localStorage.removeItem('user');
  });

  // Optionally call logout API
  const apiUrl = Cypress.env('apiUrl');
  cy.request({
    method: 'POST',
    url: `${apiUrl}/api/v1/auth/logout`,
    failOnStatusCode: false,
  });
});

/**
 * Register a new user
 */
Cypress.Commands.add(
  'register',
  (userData: {
    email: string;
    password: string;
    firstName: string;
    lastName: string;
  }) => {
    cy.visit('/register');

    // Fill registration form
    cy.get('input[name="email"], [data-testid="email-input"]')
      .should('be.visible')
      .clear()
      .type(userData.email);

    cy.get('input[name="password"], [data-testid="password-input"]')
      .should('be.visible')
      .clear()
      .type(userData.password);

    cy.get(
      'input[name="confirmPassword"], [data-testid="confirm-password-input"]'
    )
      .should('be.visible')
      .clear()
      .type(userData.password);

    cy.get('input[name="firstName"], [data-testid="first-name-input"]')
      .should('be.visible')
      .clear()
      .type(userData.firstName);

    cy.get('input[name="lastName"], [data-testid="last-name-input"]')
      .should('be.visible')
      .clear()
      .type(userData.lastName);

    // Submit form
    cy.get('button[type="submit"], [data-testid="register-button"]')
      .should('be.visible')
      .click();

    // Wait for success (email verification page or dashboard)
    cy.url().should('satisfy', (url: string) => {
      return (
        url.includes('/verify-email') ||
        url.includes('/dashboard') ||
        url.includes('/login')
      );
    });
  }
);

/**
 * Visit page with authentication
 */
Cypress.Commands.add('visitWithAuth', (url: string) => {
  // Ensure logged in first
  cy.login();

  // Then visit the page
  cy.visit(url);
  cy.waitForPageLoad();
});

export {};
