// =============================================================================
// Register Page Object
// =============================================================================

import { BasePage } from './BasePage';

export interface RegistrationData {
  email: string;
  password: string;
  confirmPassword?: string;
  firstName: string;
  lastName: string;
  companyName?: string;
  phone?: string;
}

export class RegisterPage extends BasePage {
  private pageSelectors = {
    firstNameInput: 'input[name="firstName"], [data-testid="first-name-input"]',
    lastNameInput: 'input[name="lastName"], [data-testid="last-name-input"]',
    emailInput:
      'input[name="email"], input[type="email"], [data-testid="email-input"]',
    passwordInput: 'input[name="password"], [data-testid="password-input"]',
    confirmPasswordInput:
      'input[name="confirmPassword"], [data-testid="confirm-password-input"]',
    companyNameInput:
      'input[name="companyName"], [data-testid="company-name-input"]',
    phoneInput:
      'input[name="phone"], input[type="tel"], [data-testid="phone-input"]',
    termsCheckbox: 'input[name="terms"], [data-testid="terms-checkbox"]',
    registerButton: 'button[type="submit"], [data-testid="register-button"]',
    loginLink: 'a[href*="login"], [data-testid="login-link"]',
    errorMessage:
      '[data-testid="error-message"], .error-message, [role="alert"]',
    successMessage: '[data-testid="success-message"], .success-message',
  };

  // Visit register page
  visit(): void {
    super.visit('/register');
  }

  // Fill first name
  fillFirstName(firstName: string): this {
    cy.get(this.pageSelectors.firstNameInput).clear().type(firstName);
    return this;
  }

  // Fill last name
  fillLastName(lastName: string): this {
    cy.get(this.pageSelectors.lastNameInput).clear().type(lastName);
    return this;
  }

  // Fill email
  fillEmail(email: string): this {
    cy.get(this.pageSelectors.emailInput).clear().type(email);
    return this;
  }

  // Fill password
  fillPassword(password: string): this {
    cy.get(this.pageSelectors.passwordInput).clear().type(password);
    return this;
  }

  // Fill confirm password
  fillConfirmPassword(password: string): this {
    cy.get(this.pageSelectors.confirmPasswordInput).clear().type(password);
    return this;
  }

  // Fill company name (optional)
  fillCompanyName(companyName: string): this {
    cy.get(this.pageSelectors.companyNameInput).clear().type(companyName);
    return this;
  }

  // Fill phone (optional)
  fillPhone(phone: string): this {
    cy.get(this.pageSelectors.phoneInput).clear().type(phone);
    return this;
  }

  // Accept terms
  acceptTerms(): this {
    cy.get(this.pageSelectors.termsCheckbox).check({ force: true });
    return this;
  }

  // Click register button
  clickRegister(): void {
    cy.get(this.pageSelectors.registerButton).click();
  }

  // Register with full data
  register(data: RegistrationData): void {
    this.fillFirstName(data.firstName)
      .fillLastName(data.lastName)
      .fillEmail(data.email)
      .fillPassword(data.password)
      .fillConfirmPassword(data.confirmPassword || data.password);

    if (data.companyName) {
      this.fillCompanyName(data.companyName);
    }

    if (data.phone) {
      this.fillPhone(data.phone);
    }

    this.acceptTerms().clickRegister();
  }

  // Click login link
  clickLoginLink(): void {
    cy.get(this.pageSelectors.loginLink).click();
  }

  // Assert error message
  assertErrorMessage(message: string): void {
    cy.get(this.pageSelectors.errorMessage)
      .should('be.visible')
      .and('contain.text', message);
  }

  // Assert success message
  assertSuccessMessage(message: string): void {
    cy.get(this.pageSelectors.successMessage)
      .should('be.visible')
      .and('contain.text', message);
  }

  // Assert registration success (redirected to verify email or login)
  assertRegistrationSuccess(): void {
    cy.url().should('satisfy', (url: string) => {
      return (
        url.includes('/verify-email') ||
        url.includes('/login') ||
        url.includes('/dashboard')
      );
    });
  }

  // Assert field error
  assertFieldValidationError(fieldName: string, message: string): void {
    cy.assertFieldError(fieldName, message);
  }

  // Assert password mismatch error
  assertPasswordMismatchError(): void {
    cy.get(this.pageSelectors.confirmPasswordInput)
      .parents('.form-group, .field')
      .find('.error, .error-message')
      .should('contain.text', 'match');
  }
}

export default RegisterPage;
