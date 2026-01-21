// =============================================================================
// Quotes Page Object
// =============================================================================

import { BasePage } from './BasePage';

export interface QuoteLineItem {
  description: string;
  quantity: number;
  unitPrice: number;
}

export interface QuoteData {
  customerId?: string;
  customerName?: string;
  title?: string;
  validUntil?: string;
  lineItems: QuoteLineItem[];
  notes?: string;
  terms?: string;
  discount?: number;
  discountType?: 'percentage' | 'fixed';
}

export class QuotesPage extends BasePage {
  private pageSelectors = {
    // List view
    quoteList: '[data-testid="quote-list"], .quote-list',
    quoteCard: '[data-testid="quote-card"], .quote-card',
    quoteRow: '[data-testid="quote-row"], tbody tr',

    // Actions
    addQuoteButton:
      '[data-testid="add-quote"], button:contains("Add Quote"), button:contains("New Quote")',

    // Filters
    statusFilter: '[data-testid="status-filter"]',
    dateFilter: '[data-testid="date-filter"]',
    customerFilter: '[data-testid="customer-filter"]',

    // Form fields
    titleInput: '[name="title"], [data-testid="title-input"]',
    customerSelect: '[name="customerId"], [data-testid="customer-select"]',
    customerSearch: '[data-testid="customer-search"]',
    validUntilInput: '[name="validUntil"], [data-testid="valid-until"]',
    notesInput: '[name="notes"], [data-testid="notes-input"]',
    termsInput: '[name="terms"], [data-testid="terms-input"]',
    discountInput: '[name="discount"], [data-testid="discount-input"]',
    discountTypeSelect: '[name="discountType"], [data-testid="discount-type"]',

    // Line items
    lineItemsContainer: '[data-testid="line-items"], .line-items',
    addLineItemButton:
      '[data-testid="add-line-item"], button:contains("Add Line"), button:contains("Add Item")',
    lineItemRow: '[data-testid="line-item"], .line-item',
    lineItemDescription:
      '[name*="description"], [data-testid="item-description"]',
    lineItemQuantity: '[name*="quantity"], [data-testid="item-quantity"]',
    lineItemPrice:
      '[name*="price"], [name*="unitPrice"], [data-testid="item-price"]',
    removeLineItemButton: '[data-testid="remove-line-item"], .remove-item',

    // Totals
    subtotal: '[data-testid="subtotal"], .subtotal',
    tax: '[data-testid="tax"], .tax',
    total: '[data-testid="total"], .total',

    // Detail view
    quoteDetail: '[data-testid="quote-detail"]',
    quoteNumber: '[data-testid="quote-number"], .quote-number',
    quoteStatus: '[data-testid="quote-status"], .quote-status',
    quoteCustomer: '[data-testid="quote-customer"], .quote-customer',
    quoteAmount: '[data-testid="quote-amount"], .quote-amount',

    // Actions in detail view
    editButton: '[data-testid="edit-quote"], button:contains("Edit")',
    deleteButton: '[data-testid="delete-quote"], button:contains("Delete")',
    sendButton: '[data-testid="send-quote"], button:contains("Send")',
    approveButton: '[data-testid="approve-quote"], button:contains("Approve")',
    rejectButton: '[data-testid="reject-quote"], button:contains("Reject")',
    convertToInvoiceButton:
      '[data-testid="convert-to-invoice"], button:contains("Convert to Invoice")',
    convertToJobButton:
      '[data-testid="convert-to-job"], button:contains("Convert to Job")',
    duplicateButton:
      '[data-testid="duplicate-quote"], button:contains("Duplicate")',
    downloadPdfButton:
      '[data-testid="download-pdf"], button:contains("Download PDF")',

    // Preview
    previewButton: '[data-testid="preview-quote"], button:contains("Preview")',
    previewModal: '[data-testid="preview-modal"]',
  };

  // Visit quotes list
  visit(): void {
    super.visit('/quotes');
  }

  // Visit quote detail page
  visitQuote(quoteId: string): void {
    super.visit(`/quotes/${quoteId}`);
  }

  // Click add quote button
  clickAddQuote(): void {
    cy.get(this.pageSelectors.addQuoteButton).click();
  }

  // Select customer
  selectCustomer(customerName: string): void {
    cy.get(
      this.pageSelectors.customerSearch || this.pageSelectors.customerSelect
    )
      .click()
      .type(customerName);
    cy.get('[role="listbox"], .autocomplete-results')
      .contains(customerName)
      .click();
  }

  // Fill quote form
  fillQuoteForm(data: QuoteData): void {
    if (data.title) {
      cy.get(this.pageSelectors.titleInput).clear().type(data.title);
    }

    if (data.customerName) {
      this.selectCustomer(data.customerName);
    }

    if (data.validUntil) {
      cy.get(this.pageSelectors.validUntilInput).clear().type(data.validUntil);
    }

    // Add line items
    data.lineItems.forEach((item, index) => {
      if (index > 0) {
        cy.get(this.pageSelectors.addLineItemButton).click();
      }

      cy.get(this.pageSelectors.lineItemRow)
        .eq(index)
        .within(() => {
          cy.get(this.pageSelectors.lineItemDescription)
            .clear()
            .type(item.description);
          cy.get(this.pageSelectors.lineItemQuantity)
            .clear()
            .type(item.quantity.toString());
          cy.get(this.pageSelectors.lineItemPrice)
            .clear()
            .type(item.unitPrice.toFixed(2));
        });
    });

    if (data.discount) {
      cy.get(this.pageSelectors.discountInput)
        .clear()
        .type(data.discount.toString());
      if (data.discountType) {
        cy.get(this.pageSelectors.discountTypeSelect).select(data.discountType);
      }
    }

    if (data.notes) {
      cy.get(this.pageSelectors.notesInput).clear().type(data.notes);
    }

    if (data.terms) {
      cy.get(this.pageSelectors.termsInput).clear().type(data.terms);
    }
  }

  // Add line item
  addLineItem(item: QuoteLineItem): void {
    cy.get(this.pageSelectors.addLineItemButton).click();
    cy.get(this.pageSelectors.lineItemRow)
      .last()
      .within(() => {
        cy.get(this.pageSelectors.lineItemDescription)
          .clear()
          .type(item.description);
        cy.get(this.pageSelectors.lineItemQuantity)
          .clear()
          .type(item.quantity.toString());
        cy.get(this.pageSelectors.lineItemPrice)
          .clear()
          .type(item.unitPrice.toFixed(2));
      });
  }

  // Remove line item
  removeLineItem(index: number): void {
    cy.get(this.pageSelectors.lineItemRow)
      .eq(index)
      .find(this.pageSelectors.removeLineItemButton)
      .click();
  }

  // Create quote (full flow)
  createQuote(data: QuoteData): void {
    this.clickAddQuote();
    this.fillQuoteForm(data);
    this.submitForm();
  }

  // Get quotes
  getQuotes(): Cypress.Chainable<JQuery<HTMLElement>> {
    return cy.get(
      `${this.pageSelectors.quoteCard}, ${this.pageSelectors.quoteRow}`
    );
  }

  // Click quote to view details
  clickQuote(identifier: string): void {
    cy.get(`${this.pageSelectors.quoteCard}, ${this.pageSelectors.quoteRow}`)
      .contains(identifier)
      .click();
  }

  // Filter by status
  filterByStatus(status: string): void {
    cy.get(this.pageSelectors.statusFilter).click();
    cy.get('[role="listbox"], .dropdown-menu').contains(status).click();
  }

  // Click edit button
  clickEdit(): void {
    cy.get(this.pageSelectors.editButton).click();
  }

  // Click delete button
  clickDelete(): void {
    cy.get(this.pageSelectors.deleteButton).click();
  }

  // Send quote to customer
  clickSend(): void {
    cy.get(this.pageSelectors.sendButton).click();
  }

  // Approve quote
  clickApprove(): void {
    cy.get(this.pageSelectors.approveButton).click();
  }

  // Reject quote
  clickReject(): void {
    cy.get(this.pageSelectors.rejectButton).click();
  }

  // Convert to invoice
  clickConvertToInvoice(): void {
    cy.get(this.pageSelectors.convertToInvoiceButton).click();
  }

  // Convert to job
  clickConvertToJob(): void {
    cy.get(this.pageSelectors.convertToJobButton).click();
  }

  // Duplicate quote
  clickDuplicate(): void {
    cy.get(this.pageSelectors.duplicateButton).click();
  }

  // Download PDF
  clickDownloadPdf(): void {
    cy.get(this.pageSelectors.downloadPdfButton).click();
  }

  // Preview quote
  clickPreview(): void {
    cy.get(this.pageSelectors.previewButton).click();
  }

  // Assert quote status
  assertQuoteStatus(status: string): void {
    cy.get(this.pageSelectors.quoteStatus).should('contain.text', status);
  }

  // Assert quote total
  assertTotal(amount: string): void {
    cy.get(this.pageSelectors.total).should('contain.text', amount);
  }

  // Get line items count
  getLineItemsCount(): Cypress.Chainable<number> {
    return cy.get(this.pageSelectors.lineItemRow).its('length');
  }

  // Assert quote in list
  assertQuoteInList(identifier: string): void {
    cy.get(`${this.pageSelectors.quoteList}, ${this.selectors.table}`).should(
      'contain.text',
      identifier
    );
  }
}

export default QuotesPage;
