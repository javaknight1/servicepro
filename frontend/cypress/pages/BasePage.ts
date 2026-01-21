// =============================================================================
// Base Page Object
// =============================================================================
// All page objects extend this base class for shared functionality

export class BasePage {
  // Common selectors
  protected selectors = {
    // Layout
    header: '[data-testid="header"], header',
    footer: '[data-testid="footer"], footer',
    sidebar: '[data-testid="sidebar"], aside',
    mainContent: '[data-testid="main-content"], main',

    // Navigation
    navMenu: '[data-testid="nav-menu"], nav',
    userMenu: '[data-testid="user-menu"]',
    breadcrumb: '[data-testid="breadcrumb"], nav[aria-label="breadcrumb"]',

    // Common UI elements
    pageTitle: 'h1, [data-testid="page-title"]',
    loadingSpinner: '[data-testid="loading"], .loading, .spinner',
    toast: '[data-testid="toast"], [role="alert"], .toast',
    modal: '[data-testid="modal"], [role="dialog"], .modal',
    modalClose:
      '[data-testid="modal-close"], .modal-close, [aria-label="Close"]',

    // Buttons
    submitButton: 'button[type="submit"], [data-testid="submit-button"]',
    cancelButton: '[data-testid="cancel-button"], button:contains("Cancel")',
    saveButton: '[data-testid="save-button"], button:contains("Save")',
    deleteButton: '[data-testid="delete-button"], button:contains("Delete")',

    // Tables
    table: '[data-testid="data-table"], table',
    tableRow: 'tbody tr',
    tableHeader: 'thead th',
    pagination: '[data-testid="pagination"], .pagination',

    // Search/Filter
    searchInput: '[data-testid="search-input"], input[type="search"]',
    filterButton: '[data-testid="filter-button"]',
    filterDropdown: '[data-testid="filter-dropdown"]',

    // Empty states
    emptyState: '[data-testid="empty-state"], .empty-state',
    noResults: '[data-testid="no-results"]',
  };

  // Visit page
  visit(path: string): void {
    cy.visit(path);
    this.waitForPageLoad();
  }

  // Wait for page to fully load
  waitForPageLoad(): void {
    cy.waitForPageLoad();
  }

  // Wait for loading spinner to disappear
  waitForLoadingComplete(): void {
    cy.get(this.selectors.loadingSpinner, { timeout: 30000 }).should(
      'not.exist'
    );
  }

  // Get page title
  getPageTitle(): Cypress.Chainable<JQuery<HTMLElement>> {
    return cy.get(this.selectors.pageTitle).first();
  }

  // Assert page title
  assertPageTitle(title: string): void {
    this.getPageTitle().should('contain.text', title);
  }

  // Click navigation item
  clickNavItem(itemName: string): void {
    cy.get(this.selectors.navMenu).contains(itemName).click();
    this.waitForPageLoad();
  }

  // Open user menu
  openUserMenu(): void {
    cy.get(this.selectors.userMenu).click();
  }

  // Assert URL contains path
  assertUrl(path: string): void {
    cy.url().should('include', path);
  }

  // Assert toast message
  assertToast(message: string, type?: 'success' | 'error' | 'warning'): void {
    const selector = type
      ? `${this.selectors.toast}.${type}, ${this.selectors.toast}[data-type="${type}"]`
      : this.selectors.toast;

    cy.get(selector).should('be.visible').and('contain.text', message);
  }

  // Close toast
  closeToast(): void {
    cy.get(
      `${this.selectors.toast} button, ${this.selectors.toast} [aria-label="Close"]`
    )
      .first()
      .click();
  }

  // Open modal
  assertModalOpen(): void {
    cy.get(this.selectors.modal).should('be.visible');
  }

  // Close modal
  closeModal(): void {
    cy.get(this.selectors.modalClose).first().click();
    cy.get(this.selectors.modal).should('not.exist');
  }

  // Search in table/list
  search(query: string): void {
    cy.get(this.selectors.searchInput).clear().type(query);
  }

  // Clear search
  clearSearch(): void {
    cy.get(this.selectors.searchInput).clear();
  }

  // Get table rows
  getTableRows(): Cypress.Chainable<JQuery<HTMLElement>> {
    return cy.get(`${this.selectors.table} ${this.selectors.tableRow}`);
  }

  // Assert table row count
  assertTableRowCount(count: number): void {
    this.getTableRows().should('have.length', count);
  }

  // Assert table contains text
  assertTableContains(text: string): void {
    cy.get(this.selectors.table).should('contain.text', text);
  }

  // Click table row
  clickTableRow(index: number): void {
    this.getTableRows().eq(index).click();
  }

  // Get table row by content
  getTableRowByContent(text: string): Cypress.Chainable<JQuery<HTMLElement>> {
    return cy
      .get(`${this.selectors.table} ${this.selectors.tableRow}`)
      .contains(text)
      .parents('tr');
  }

  // Pagination - go to next page
  goToNextPage(): void {
    cy.get(this.selectors.pagination)
      .contains(/next|>/i)
      .click();
    this.waitForLoadingComplete();
  }

  // Pagination - go to previous page
  goToPreviousPage(): void {
    cy.get(this.selectors.pagination)
      .contains(/prev|</i)
      .click();
    this.waitForLoadingComplete();
  }

  // Pagination - go to specific page
  goToPage(pageNumber: number): void {
    cy.get(this.selectors.pagination).contains(pageNumber.toString()).click();
    this.waitForLoadingComplete();
  }

  // Assert empty state is shown
  assertEmptyState(): void {
    cy.get(this.selectors.emptyState).should('be.visible');
  }

  // Assert no results message
  assertNoResults(): void {
    cy.get(`${this.selectors.noResults}, ${this.selectors.emptyState}`).should(
      'be.visible'
    );
  }

  // Submit form
  submitForm(): void {
    cy.get(this.selectors.submitButton).first().click();
  }

  // Cancel action
  cancel(): void {
    cy.get(this.selectors.cancelButton).first().click();
  }

  // Save
  save(): void {
    cy.get(this.selectors.saveButton).first().click();
  }

  // Delete
  clickDelete(): void {
    cy.get(this.selectors.deleteButton).first().click();
  }

  // Confirm delete in modal
  confirmDelete(): void {
    cy.get(this.selectors.modal)
      .should('be.visible')
      .within(() => {
        cy.contains(/confirm|yes|delete/i).click();
      });
  }

  // Take screenshot
  takeScreenshot(name: string): void {
    cy.screenshot(name);
  }
}

export default BasePage;
