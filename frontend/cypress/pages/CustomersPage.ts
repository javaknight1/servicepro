// =============================================================================
// Customers Page Object
// =============================================================================

import { BasePage } from './BasePage';

export interface CustomerData {
  firstName: string;
  lastName: string;
  email: string;
  phone?: string;
  companyName?: string;
  address?: {
    street?: string;
    city?: string;
    state?: string;
    zip?: string;
  };
  notes?: string;
}

export class CustomersPage extends BasePage {
  private pageSelectors = {
    // List view
    customerList: '[data-testid="customer-list"], .customer-list',
    customerCard: '[data-testid="customer-card"], .customer-card',
    customerRow: '[data-testid="customer-row"], tbody tr',

    // Actions
    addCustomerButton:
      '[data-testid="add-customer"], button:contains("Add Customer"), button:contains("New Customer")',
    importButton: '[data-testid="import-button"], button:contains("Import")',
    exportButton: '[data-testid="export-button"], button:contains("Export")',

    // Filters
    statusFilter: '[data-testid="status-filter"]',
    sortDropdown: '[data-testid="sort-dropdown"]',

    // Form fields
    firstNameInput: '[name="firstName"], [data-testid="firstName-input"]',
    lastNameInput: '[name="lastName"], [data-testid="lastName-input"]',
    emailInput: '[name="email"], [data-testid="email-input"]',
    phoneInput: '[name="phone"], [data-testid="phone-input"]',
    companyNameInput: '[name="companyName"], [data-testid="companyName-input"]',
    streetInput:
      '[name="street"], [name="address.street"], [data-testid="street-input"]',
    cityInput:
      '[name="city"], [name="address.city"], [data-testid="city-input"]',
    stateInput:
      '[name="state"], [name="address.state"], [data-testid="state-input"]',
    zipInput: '[name="zip"], [name="address.zip"], [data-testid="zip-input"]',
    notesInput: '[name="notes"], [data-testid="notes-input"]',

    // Detail view
    customerDetail: '[data-testid="customer-detail"]',
    customerName: '[data-testid="customer-name"], .customer-name',
    customerEmail: '[data-testid="customer-email"], .customer-email',
    customerPhone: '[data-testid="customer-phone"], .customer-phone',

    // Related data
    customerJobs: '[data-testid="customer-jobs"], .customer-jobs',
    customerInvoices: '[data-testid="customer-invoices"], .customer-invoices',
    customerQuotes: '[data-testid="customer-quotes"], .customer-quotes',

    // Actions in detail view
    editButton: '[data-testid="edit-customer"], button:contains("Edit")',
    deleteButton: '[data-testid="delete-customer"], button:contains("Delete")',
    createJobButton:
      '[data-testid="create-job"], button:contains("Create Job")',
    createQuoteButton:
      '[data-testid="create-quote"], button:contains("Create Quote")',
    createInvoiceButton:
      '[data-testid="create-invoice"], button:contains("Create Invoice")',
  };

  // Visit customers list
  visit(): void {
    super.visit('/customers');
  }

  // Visit customer detail page
  visitCustomer(customerId: string): void {
    super.visit(`/customers/${customerId}`);
  }

  // Click add customer button
  clickAddCustomer(): void {
    cy.get(this.pageSelectors.addCustomerButton).click();
  }

  // Fill customer form
  fillCustomerForm(data: CustomerData): void {
    cy.get(this.pageSelectors.firstNameInput).clear().type(data.firstName);
    cy.get(this.pageSelectors.lastNameInput).clear().type(data.lastName);
    cy.get(this.pageSelectors.emailInput).clear().type(data.email);

    if (data.phone) {
      cy.get(this.pageSelectors.phoneInput).clear().type(data.phone);
    }

    if (data.companyName) {
      cy.get(this.pageSelectors.companyNameInput)
        .clear()
        .type(data.companyName);
    }

    if (data.address) {
      if (data.address.street) {
        cy.get(this.pageSelectors.streetInput)
          .clear()
          .type(data.address.street);
      }
      if (data.address.city) {
        cy.get(this.pageSelectors.cityInput).clear().type(data.address.city);
      }
      if (data.address.state) {
        cy.get(this.pageSelectors.stateInput).clear().type(data.address.state);
      }
      if (data.address.zip) {
        cy.get(this.pageSelectors.zipInput).clear().type(data.address.zip);
      }
    }

    if (data.notes) {
      cy.get(this.pageSelectors.notesInput).clear().type(data.notes);
    }
  }

  // Create customer (full flow)
  createCustomer(data: CustomerData): void {
    this.clickAddCustomer();
    this.fillCustomerForm(data);
    this.submitForm();
  }

  // Search for customer
  searchCustomer(query: string): void {
    this.search(query);
  }

  // Get customer cards/rows
  getCustomers(): Cypress.Chainable<JQuery<HTMLElement>> {
    return cy.get(
      `${this.pageSelectors.customerCard}, ${this.pageSelectors.customerRow}`
    );
  }

  // Assert customer count
  assertCustomerCount(count: number): void {
    this.getCustomers().should('have.length', count);
  }

  // Click customer to view details
  clickCustomer(name: string): void {
    cy.get(
      `${this.pageSelectors.customerCard}, ${this.pageSelectors.customerRow}`
    )
      .contains(name)
      .click();
  }

  // Filter by status
  filterByStatus(status: string): void {
    cy.get(this.pageSelectors.statusFilter).click();
    cy.get('[role="listbox"], .dropdown-menu').contains(status).click();
  }

  // Sort customers
  sortBy(field: string): void {
    cy.get(this.pageSelectors.sortDropdown).click();
    cy.get('[role="listbox"], .dropdown-menu').contains(field).click();
  }

  // Click edit button (detail view)
  clickEdit(): void {
    cy.get(this.pageSelectors.editButton).click();
  }

  // Click delete button (detail view)
  clickDelete(): void {
    cy.get(this.pageSelectors.deleteButton).click();
  }

  // Create job for customer
  clickCreateJob(): void {
    cy.get(this.pageSelectors.createJobButton).click();
  }

  // Create quote for customer
  clickCreateQuote(): void {
    cy.get(this.pageSelectors.createQuoteButton).click();
  }

  // Create invoice for customer
  clickCreateInvoice(): void {
    cy.get(this.pageSelectors.createInvoiceButton).click();
  }

  // Get customer's jobs
  getCustomerJobs(): Cypress.Chainable<JQuery<HTMLElement>> {
    return cy.get(this.pageSelectors.customerJobs).find('li, tr, .job-item');
  }

  // Get customer's invoices
  getCustomerInvoices(): Cypress.Chainable<JQuery<HTMLElement>> {
    return cy
      .get(this.pageSelectors.customerInvoices)
      .find('li, tr, .invoice-item');
  }

  // Assert customer detail visible
  assertCustomerDetailVisible(): void {
    cy.get(this.pageSelectors.customerDetail).should('be.visible');
  }

  // Assert customer name in detail view
  assertCustomerName(name: string): void {
    cy.get(this.pageSelectors.customerName).should('contain.text', name);
  }

  // Assert customer email in detail view
  assertCustomerEmail(email: string): void {
    cy.get(this.pageSelectors.customerEmail).should('contain.text', email);
  }

  // Import customers
  clickImport(): void {
    cy.get(this.pageSelectors.importButton).click();
  }

  // Export customers
  clickExport(): void {
    cy.get(this.pageSelectors.exportButton).click();
  }

  // Assert customer in list
  assertCustomerInList(name: string): void {
    cy.get(
      `${this.pageSelectors.customerList}, ${this.selectors.table}`
    ).should('contain.text', name);
  }

  // Assert customer not in list
  assertCustomerNotInList(name: string): void {
    cy.get(
      `${this.pageSelectors.customerList}, ${this.selectors.table}`
    ).should('not.contain.text', name);
  }
}

export default CustomersPage;
