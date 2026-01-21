// =============================================================================
// Registration E2E Tests
// =============================================================================

import { RegisterPage } from '../../pages';

describe('Registration', () => {
  const registerPage = new RegisterPage();

  beforeEach(() => {
    registerPage.visit();
  });

  describe('UI Elements', () => {
    it('should display registration form with all required fields', () => {
      cy.get('input[name="firstName"]').should('be.visible');
      cy.get('input[name="lastName"]').should('be.visible');
      cy.get('input[name="email"]').should('be.visible');
      cy.get('input[name="password"]').should('be.visible');
      cy.get('input[name="confirmPassword"]').should('be.visible');
      cy.get('button[type="submit"]').should('be.visible');
    });

    it('should have a link to login page', () => {
      cy.contains(/already have an account|login|sign in/i).should(
        'be.visible'
      );
    });
  });

  describe('Successful Registration', () => {
    it('should register a new user successfully', () => {
      const uniqueEmail = `test-${Date.now()}@example.com`;

      cy.intercept('POST', '**/api/v1/auth/register', {
        statusCode: 201,
        body: {
          success: true,
          message:
            'Registration successful. Please check your email to verify your account.',
          user: {
            id: 'new-user-123',
            email: uniqueEmail,
          },
        },
      }).as('registerRequest');

      registerPage.register({
        firstName: 'Test',
        lastName: 'User',
        email: uniqueEmail,
        password: 'SecurePassword123!',
      });

      cy.wait('@registerRequest').its('request.body').should('include', {
        firstName: 'Test',
        lastName: 'User',
        email: uniqueEmail,
      });

      // Should redirect to verify email or login page
      registerPage.assertRegistrationSuccess();
    });

    it('should show success message after registration', () => {
      cy.intercept('POST', '**/api/v1/auth/register', {
        statusCode: 201,
        body: {
          success: true,
          message: 'Please check your email',
        },
      }).as('registerRequest');

      registerPage.register({
        firstName: 'Test',
        lastName: 'User',
        email: `test-${Date.now()}@example.com`,
        password: 'SecurePassword123!',
      });

      cy.wait('@registerRequest');

      // Check for success indication
      cy.contains(/success|check your email|verify/i).should('be.visible');
    });
  });

  describe('Registration Validation', () => {
    it('should show error for empty required fields', () => {
      cy.get('button[type="submit"]').click();

      // Check for validation errors
      cy.get('input:invalid').should('have.length.at.least', 1);
    });

    it('should show error for invalid email format', () => {
      registerPage.fillFirstName('Test');
      registerPage.fillLastName('User');
      registerPage.fillEmail('invalid-email');
      registerPage.fillPassword('Password123!');
      registerPage.fillConfirmPassword('Password123!');
      registerPage.clickRegister();

      cy.get('input[name="email"]:invalid, [data-testid="email-error"]').should(
        'exist'
      );
    });

    it('should show error for weak password', () => {
      cy.intercept('POST', '**/api/v1/auth/register', {
        statusCode: 400,
        body: {
          error: true,
          message:
            'Password must be at least 8 characters with uppercase, lowercase, and number',
        },
      }).as('registerRequest');

      registerPage.register({
        firstName: 'Test',
        lastName: 'User',
        email: 'test@example.com',
        password: 'weak',
      });

      cy.wait('@registerRequest');

      registerPage.assertErrorMessage('Password must be');
    });

    it('should show error when passwords do not match', () => {
      registerPage.fillFirstName('Test');
      registerPage.fillLastName('User');
      registerPage.fillEmail('test@example.com');
      registerPage.fillPassword('Password123!');
      registerPage.fillConfirmPassword('DifferentPassword123!');
      registerPage.clickRegister();

      // Check for password mismatch error
      cy.get(
        '[data-testid="confirm-password-error"], [data-testid="password-error"]'
      ).should('contain.text', 'match');
    });

    it('should show error for existing email', () => {
      cy.intercept('POST', '**/api/v1/auth/register', {
        statusCode: 409,
        body: {
          error: true,
          message: 'An account with this email already exists',
        },
      }).as('registerRequest');

      registerPage.register({
        firstName: 'Test',
        lastName: 'User',
        email: 'existing@example.com',
        password: 'Password123!',
      });

      cy.wait('@registerRequest');

      registerPage.assertErrorMessage('already exists');
    });

    it('should validate first name is not empty', () => {
      registerPage.fillLastName('User');
      registerPage.fillEmail('test@example.com');
      registerPage.fillPassword('Password123!');
      registerPage.fillConfirmPassword('Password123!');
      registerPage.clickRegister();

      cy.get(
        'input[name="firstName"]:invalid, [data-testid="firstName-error"]'
      ).should('exist');
    });

    it('should validate last name is not empty', () => {
      registerPage.fillFirstName('Test');
      registerPage.fillEmail('test@example.com');
      registerPage.fillPassword('Password123!');
      registerPage.fillConfirmPassword('Password123!');
      registerPage.clickRegister();

      cy.get(
        'input[name="lastName"]:invalid, [data-testid="lastName-error"]'
      ).should('exist');
    });
  });

  describe('Password Strength', () => {
    it('should show password strength indicator', () => {
      registerPage.fillPassword('weak');

      cy.get('[data-testid="password-strength"], .password-strength').should(
        'exist'
      );
    });

    it('should update strength indicator as password changes', () => {
      registerPage.fillPassword('a');
      cy.get('[data-testid="password-strength"]').should(
        'contain.text',
        /weak/i
      );

      cy.get('input[name="password"]').clear().type('Stronger123!');
      cy.get('[data-testid="password-strength"]').should(
        'contain.text',
        /strong/i
      );
    });
  });

  describe('Terms and Conditions', () => {
    it('should require accepting terms before registration', () => {
      registerPage.fillFirstName('Test');
      registerPage.fillLastName('User');
      registerPage.fillEmail('test@example.com');
      registerPage.fillPassword('Password123!');
      registerPage.fillConfirmPassword('Password123!');

      // Don't accept terms
      registerPage.clickRegister();

      // Should show error or prevent submission
      cy.get('[data-testid="terms-error"], input[name="terms"]:invalid').should(
        'exist'
      );
    });

    it('should link to terms and conditions page', () => {
      cy.contains(/terms|conditions/i).should('have.attr', 'href');
    });

    it('should link to privacy policy', () => {
      cy.contains(/privacy/i).should('have.attr', 'href');
    });
  });

  describe('Navigation', () => {
    it('should navigate to login page', () => {
      registerPage.clickLoginLink();

      cy.url().should('include', '/login');
    });
  });

  describe('Company Registration', () => {
    it('should allow optional company name', () => {
      cy.intercept('POST', '**/api/v1/auth/register', {
        statusCode: 201,
        body: { success: true },
      }).as('registerRequest');

      const companyName = 'Test Company LLC';

      registerPage.fillFirstName('Test');
      registerPage.fillLastName('User');
      registerPage.fillEmail(`test-${Date.now()}@example.com`);
      registerPage.fillPassword('Password123!');
      registerPage.fillConfirmPassword('Password123!');
      registerPage.fillCompanyName(companyName);
      registerPage.acceptTerms();
      registerPage.clickRegister();

      cy.wait('@registerRequest').its('request.body').should('include', {
        companyName,
      });
    });
  });

  describe('Accessibility', () => {
    it('should have proper form labels', () => {
      cy.get('label').should('have.length.at.least', 5);
    });

    it('should have proper ARIA attributes on error messages', () => {
      cy.get('button[type="submit"]').click();

      cy.get('[aria-invalid="true"], [aria-describedby]').should('exist');
    });

    it('should support keyboard navigation', () => {
      cy.get('input[name="firstName"]').focus();
      cy.focused().should('have.attr', 'name', 'firstName');

      cy.focused().tab();
      cy.focused().should('have.attr', 'name', 'lastName');
    });
  });

  describe('Rate Limiting', () => {
    it('should handle rate limiting gracefully', () => {
      cy.intercept('POST', '**/api/v1/auth/register', {
        statusCode: 429,
        body: {
          error: true,
          message: 'Too many registration attempts. Please try again later.',
        },
      }).as('rateLimitedRequest');

      registerPage.register({
        firstName: 'Test',
        lastName: 'User',
        email: 'test@example.com',
        password: 'Password123!',
      });

      cy.wait('@rateLimitedRequest');

      registerPage.assertErrorMessage('Too many');
    });
  });
});
