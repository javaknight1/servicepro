// =============================================================================
// Login Page Object
// =============================================================================

import { BasePage } from './BasePage';

export class LoginPage extends BasePage {
  private pageSelectors = {
    emailInput:
      'input[name="email"], input[type="email"], [data-testid="email-input"]',
    passwordInput:
      'input[name="password"], input[type="password"], [data-testid="password-input"]',
    loginButton: 'button[type="submit"], [data-testid="login-button"]',
    forgotPasswordLink:
      'a[href*="forgot"], [data-testid="forgot-password-link"]',
    registerLink: 'a[href*="register"], [data-testid="register-link"]',
    rememberMeCheckbox: 'input[name="rememberMe"], [data-testid="remember-me"]',
    errorMessage:
      '[data-testid="error-message"], .error-message, [role="alert"]',
    form: 'form',
  };

  // Visit login page
  visit(): void {
    super.visit('/login');
  }

  // Fill email
  fillEmail(email: string): this {
    cy.get(this.pageSelectors.emailInput)
      .should('be.visible')
      .clear()
      .type(email);
    return this;
  }

  // Fill password
  fillPassword(password: string): this {
    cy.get(this.pageSelectors.passwordInput)
      .should('be.visible')
      .clear()
      .type(password);
    return this;
  }

  // Check remember me
  checkRememberMe(): this {
    cy.get(this.pageSelectors.rememberMeCheckbox).check({ force: true });
    return this;
  }

  // Click login button
  clickLogin(): void {
    cy.get(this.pageSelectors.loginButton).should('be.visible').click();
  }

  // Login with credentials (full flow)
  login(email: string, password: string): void {
    this.fillEmail(email).fillPassword(password).clickLogin();
  }

  // Click forgot password link
  clickForgotPassword(): void {
    cy.get(this.pageSelectors.forgotPasswordLink).click();
  }

  // Click register link
  clickRegister(): void {
    cy.get(this.pageSelectors.registerLink).click();
  }

  // Assert error message
  assertErrorMessage(message: string): void {
    cy.get(this.pageSelectors.errorMessage)
      .should('be.visible')
      .and('contain.text', message);
  }

  // Assert no error message
  assertNoError(): void {
    cy.get(this.pageSelectors.errorMessage).should('not.exist');
  }

  // Assert login button is disabled
  assertLoginButtonDisabled(): void {
    cy.get(this.pageSelectors.loginButton).should('be.disabled');
  }

  // Assert login button is enabled
  assertLoginButtonEnabled(): void {
    cy.get(this.pageSelectors.loginButton).should('not.be.disabled');
  }

  // Assert redirect to dashboard after login
  assertLoginSuccess(): void {
    cy.url().should('include', '/dashboard');
  }

  // Assert validation error on email field
  assertEmailError(message: string): void {
    cy.assertFieldError('email', message);
  }

  // Assert validation error on password field
  assertPasswordError(message: string): void {
    cy.assertFieldError('password', message);
  }

  // Clear form
  clearForm(): void {
    cy.get(this.pageSelectors.emailInput).clear();
    cy.get(this.pageSelectors.passwordInput).clear();
  }

  // Get email input
  getEmailInput(): Cypress.Chainable<JQuery<HTMLElement>> {
    return cy.get(this.pageSelectors.emailInput);
  }

  // Get password input
  getPasswordInput(): Cypress.Chainable<JQuery<HTMLElement>> {
    return cy.get(this.pageSelectors.passwordInput);
  }
}

export default LoginPage;
