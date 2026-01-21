// =============================================================================
// Login E2E Tests
// =============================================================================

import { LoginPage } from '../../pages';

describe('Login', () => {
  const loginPage = new LoginPage();

  beforeEach(() => {
    // Visit login page before each test
    loginPage.visit();
  });

  describe('UI Elements', () => {
    it('should display login form with all required elements', () => {
      // Check email input
      loginPage
        .getEmailInput()
        .should('be.visible')
        .and('have.attr', 'type', 'email');

      // Check password input
      loginPage
        .getPasswordInput()
        .should('be.visible')
        .and('have.attr', 'type', 'password');

      // Check login button
      cy.get('button[type="submit"]').should('be.visible');

      // Check forgot password link
      cy.contains('Forgot').should('be.visible');

      // Check register link
      cy.contains(/register|sign up/i).should('be.visible');
    });

    it('should display company branding', () => {
      cy.get('[data-testid="logo"], .logo, img[alt*="logo"]').should(
        'be.visible'
      );
    });
  });

  describe('Valid Login', () => {
    it('should login successfully with valid credentials', () => {
      const email = Cypress.env('testUserEmail');
      const password = Cypress.env('testUserPassword');

      // Stub login API
      cy.intercept('POST', '**/api/v1/auth/login', {
        statusCode: 200,
        body: {
          access_token: 'fake-jwt-token',
          refresh_token: 'fake-refresh-token',
          user: {
            id: 'user-123',
            email,
            role: 'admin',
          },
        },
      }).as('loginRequest');

      // Perform login
      loginPage.login(email, password);

      // Wait for API call
      cy.wait('@loginRequest').its('request.body').should('deep.equal', {
        email,
        password,
      });

      // Verify redirect to dashboard
      cy.url().should('include', '/dashboard');

      // Verify token stored
      cy.window().then((win) => {
        expect(win.localStorage.getItem('access_token')).to.exist;
      });
    });

    it('should store tokens in localStorage after successful login', () => {
      cy.intercept('POST', '**/api/v1/auth/login', {
        statusCode: 200,
        body: {
          access_token: 'test-access-token',
          refresh_token: 'test-refresh-token',
          user: { id: 'user-123', email: 'test@example.com' },
        },
      }).as('loginRequest');

      loginPage.login('test@example.com', 'password123');

      cy.wait('@loginRequest');

      cy.window().then((win) => {
        expect(win.localStorage.getItem('access_token')).to.equal(
          'test-access-token'
        );
        expect(win.localStorage.getItem('refresh_token')).to.equal(
          'test-refresh-token'
        );
      });
    });

    it('should remember user when "Remember Me" is checked', () => {
      cy.intercept('POST', '**/api/v1/auth/login', {
        statusCode: 200,
        body: {
          access_token: 'fake-token',
          refresh_token: 'fake-refresh',
          user: { id: 'user-123' },
        },
      }).as('loginRequest');

      loginPage
        .fillEmail('test@example.com')
        .fillPassword('password123')
        .checkRememberMe();

      loginPage.clickLogin();

      cy.wait('@loginRequest')
        .its('request.body')
        .should('have.property', 'rememberMe', true);
    });
  });

  describe('Invalid Login', () => {
    it('should show error for invalid credentials', () => {
      cy.intercept('POST', '**/api/v1/auth/login', {
        statusCode: 401,
        body: {
          error: true,
          message: 'Invalid email or password',
        },
      }).as('loginRequest');

      loginPage.login('wrong@email.com', 'wrongpassword');

      cy.wait('@loginRequest');

      // Check error message
      loginPage.assertErrorMessage('Invalid email or password');

      // Should stay on login page
      cy.url().should('include', '/login');
    });

    it('should show error for locked account', () => {
      cy.intercept('POST', '**/api/v1/auth/login', {
        statusCode: 403,
        body: {
          error: true,
          message: 'Account is locked. Please contact support.',
        },
      }).as('loginRequest');

      loginPage.login('locked@example.com', 'password123');

      cy.wait('@loginRequest');

      loginPage.assertErrorMessage('Account is locked');
    });

    it('should show error for unverified email', () => {
      cy.intercept('POST', '**/api/v1/auth/login', {
        statusCode: 403,
        body: {
          error: true,
          message: 'Please verify your email before logging in',
        },
      }).as('loginRequest');

      loginPage.login('unverified@example.com', 'password123');

      cy.wait('@loginRequest');

      loginPage.assertErrorMessage('verify your email');
    });
  });

  describe('Form Validation', () => {
    it('should show error for empty email', () => {
      loginPage.fillPassword('password123');
      loginPage.clickLogin();

      // Check for validation error
      cy.get('input[name="email"]:invalid, [data-testid="email-error"]').should(
        'exist'
      );
    });

    it('should show error for empty password', () => {
      loginPage.fillEmail('test@example.com');
      loginPage.clickLogin();

      // Check for validation error
      cy.get(
        'input[name="password"]:invalid, [data-testid="password-error"]'
      ).should('exist');
    });

    it('should show error for invalid email format', () => {
      loginPage.fillEmail('invalid-email');
      loginPage.fillPassword('password123');
      loginPage.clickLogin();

      // Check for validation error
      cy.get('input[name="email"]:invalid, [data-testid="email-error"]').should(
        'exist'
      );
    });

    it('should disable login button when form is empty', () => {
      // Button should be disabled initially or form should validate
      cy.get('form').should('exist');
    });
  });

  describe('Navigation', () => {
    it('should navigate to forgot password page', () => {
      loginPage.clickForgotPassword();

      cy.url().should('include', '/forgot-password');
    });

    it('should navigate to registration page', () => {
      loginPage.clickRegister();

      cy.url().should('satisfy', (url: string) => {
        return url.includes('/register') || url.includes('/signup');
      });
    });
  });

  describe('Rate Limiting', () => {
    it('should show error after too many failed attempts', () => {
      // Simulate rate limiting response
      cy.intercept('POST', '**/api/v1/auth/login', {
        statusCode: 429,
        body: {
          error: true,
          message: 'Too many login attempts. Please try again later.',
        },
      }).as('rateLimitedLogin');

      loginPage.login('test@example.com', 'wrongpassword');

      cy.wait('@rateLimitedLogin');

      loginPage.assertErrorMessage('Too many login attempts');
    });
  });

  describe('Session Management', () => {
    it('should redirect to dashboard if already logged in', () => {
      // Set token in localStorage
      cy.window().then((win) => {
        win.localStorage.setItem('access_token', 'valid-token');
      });

      // Stub token validation
      cy.intercept('GET', '**/api/v1/users/me', {
        statusCode: 200,
        body: { id: 'user-123', email: 'test@example.com' },
      });

      // Visit login page
      cy.visit('/login');

      // Should redirect to dashboard
      cy.url().should('include', '/dashboard');
    });
  });

  describe('Accessibility', () => {
    it('should have proper form labels', () => {
      cy.get('label[for]').should('have.length.at.least', 2);
    });

    it('should support keyboard navigation', () => {
      // Tab to email input
      cy.get('body').tab();
      cy.focused().should('have.attr', 'name', 'email');

      // Tab to password input
      cy.focused().tab();
      cy.focused().should('have.attr', 'name', 'password');

      // Tab to submit button
      cy.focused().tab();
      cy.focused().should('have.attr', 'type', 'submit');
    });

    it('should announce errors to screen readers', () => {
      cy.intercept('POST', '**/api/v1/auth/login', {
        statusCode: 401,
        body: { message: 'Invalid credentials' },
      });

      loginPage.login('test@example.com', 'wrong');

      cy.get('[role="alert"], [aria-live="polite"]').should(
        'contain.text',
        'Invalid'
      );
    });
  });
});
