// =============================================================================
// Payments Page Object
// =============================================================================

import { BasePage } from './BasePage';

export interface PaymentData {
  invoiceId?: string;
  amount: number;
  method: 'credit_card' | 'bank_transfer' | 'cash' | 'check' | 'other';
  date?: string;
  reference?: string;
  notes?: string;
}

export class PaymentsPage extends BasePage {
  private pageSelectors = {
    // List view
    paymentList: '[data-testid="payment-list"], .payment-list',
    paymentCard: '[data-testid="payment-card"], .payment-card',
    paymentRow: '[data-testid="payment-row"], tbody tr',

    // Actions
    addPaymentButton:
      '[data-testid="add-payment"], button:contains("Add Payment"), button:contains("Record Payment")',

    // Filters
    methodFilter: '[data-testid="method-filter"]',
    dateFilter: '[data-testid="date-filter"]',
    customerFilter: '[data-testid="customer-filter"]',
    dateRangeStart: '[data-testid="date-start"]',
    dateRangeEnd: '[data-testid="date-end"]',

    // Form fields
    invoiceSelect: '[name="invoiceId"], [data-testid="invoice-select"]',
    invoiceSearch: '[data-testid="invoice-search"]',
    amountInput: '[name="amount"], [data-testid="amount-input"]',
    methodSelect: '[name="method"], [data-testid="method-select"]',
    dateInput: '[name="date"], [data-testid="date-input"]',
    referenceInput: '[name="reference"], [data-testid="reference-input"]',
    notesInput: '[name="notes"], [data-testid="notes-input"]',

    // Detail view
    paymentDetail: '[data-testid="payment-detail"]',
    paymentId: '[data-testid="payment-id"], .payment-id',
    paymentAmount: '[data-testid="payment-amount"], .payment-amount',
    paymentMethod: '[data-testid="payment-method"], .payment-method',
    paymentDate: '[data-testid="payment-date"], .payment-date',
    paymentStatus: '[data-testid="payment-status"], .payment-status',
    relatedInvoice: '[data-testid="related-invoice"], .related-invoice',

    // Actions in detail view
    editButton: '[data-testid="edit-payment"], button:contains("Edit")',
    deleteButton: '[data-testid="delete-payment"], button:contains("Delete")',
    refundButton: '[data-testid="refund-payment"], button:contains("Refund")',
    voidButton: '[data-testid="void-payment"], button:contains("Void")',
    viewInvoiceButton:
      '[data-testid="view-invoice"], button:contains("View Invoice")',
    downloadReceiptButton:
      '[data-testid="download-receipt"], button:contains("Download Receipt")',

    // Refund modal
    refundModal: '[data-testid="refund-modal"]',
    refundAmountInput: '[name="refundAmount"], [data-testid="refund-amount"]',
    refundReasonInput: '[name="refundReason"], [data-testid="refund-reason"]',
    confirmRefundButton: '[data-testid="confirm-refund"]',

    // Summary/totals
    totalPayments: '[data-testid="total-payments"], .total-payments',
    totalRefunds: '[data-testid="total-refunds"], .total-refunds',
    netPayments: '[data-testid="net-payments"], .net-payments',

    // Stripe integration
    stripePaymentForm: '[data-testid="stripe-payment-form"]',
    cardElement: '[data-testid="card-element"], .StripeElement',
    stripeErrors: '[data-testid="stripe-errors"], .stripe-error',
    payWithCardButton: '[data-testid="pay-with-card"]',
  };

  // Visit payments list
  visit(): void {
    super.visit('/payments');
  }

  // Visit payment detail page
  visitPayment(paymentId: string): void {
    super.visit(`/payments/${paymentId}`);
  }

  // Click add payment button
  clickAddPayment(): void {
    cy.get(this.pageSelectors.addPaymentButton).click();
  }

  // Select invoice
  selectInvoice(invoiceNumber: string): void {
    cy.get(this.pageSelectors.invoiceSearch || this.pageSelectors.invoiceSelect)
      .click()
      .type(invoiceNumber);
    cy.get('[role="listbox"], .autocomplete-results')
      .contains(invoiceNumber)
      .click();
  }

  // Fill payment form
  fillPaymentForm(data: PaymentData): void {
    cy.get(this.pageSelectors.amountInput).clear().type(data.amount.toFixed(2));
    cy.get(this.pageSelectors.methodSelect).select(data.method);

    if (data.date) {
      cy.get(this.pageSelectors.dateInput).clear().type(data.date);
    }

    if (data.reference) {
      cy.get(this.pageSelectors.referenceInput).clear().type(data.reference);
    }

    if (data.notes) {
      cy.get(this.pageSelectors.notesInput).clear().type(data.notes);
    }
  }

  // Record payment (full flow)
  recordPayment(data: PaymentData, invoiceNumber?: string): void {
    this.clickAddPayment();

    if (invoiceNumber) {
      this.selectInvoice(invoiceNumber);
    }

    this.fillPaymentForm(data);
    this.submitForm();
  }

  // Get payments
  getPayments(): Cypress.Chainable<JQuery<HTMLElement>> {
    return cy.get(
      `${this.pageSelectors.paymentCard}, ${this.pageSelectors.paymentRow}`
    );
  }

  // Assert payment count
  assertPaymentCount(count: number): void {
    this.getPayments().should('have.length', count);
  }

  // Click payment to view details
  clickPayment(identifier: string): void {
    cy.get(
      `${this.pageSelectors.paymentCard}, ${this.pageSelectors.paymentRow}`
    )
      .contains(identifier)
      .click();
  }

  // Filter by payment method
  filterByMethod(method: string): void {
    cy.get(this.pageSelectors.methodFilter).click();
    cy.get('[role="listbox"], .dropdown-menu').contains(method).click();
  }

  // Filter by date range
  filterByDateRange(startDate: string, endDate: string): void {
    cy.get(this.pageSelectors.dateRangeStart).clear().type(startDate);
    cy.get(this.pageSelectors.dateRangeEnd).clear().type(endDate);
  }

  // Click edit button
  clickEdit(): void {
    cy.get(this.pageSelectors.editButton).click();
  }

  // Click delete button
  clickDelete(): void {
    cy.get(this.pageSelectors.deleteButton).click();
  }

  // Click refund button
  clickRefund(): void {
    cy.get(this.pageSelectors.refundButton).click();
  }

  // Fill refund form
  fillRefundForm(amount: number, reason?: string): void {
    cy.get(this.pageSelectors.refundAmountInput)
      .clear()
      .type(amount.toFixed(2));
    if (reason) {
      cy.get(this.pageSelectors.refundReasonInput).clear().type(reason);
    }
  }

  // Process refund (full flow)
  processRefund(amount: number, reason?: string): void {
    this.clickRefund();
    this.fillRefundForm(amount, reason);
    cy.get(this.pageSelectors.confirmRefundButton).click();
  }

  // Void payment
  clickVoid(): void {
    cy.get(this.pageSelectors.voidButton).click();
  }

  // View related invoice
  clickViewInvoice(): void {
    cy.get(this.pageSelectors.viewInvoiceButton).click();
  }

  // Download receipt
  clickDownloadReceipt(): void {
    cy.get(this.pageSelectors.downloadReceiptButton).click();
  }

  // Assert payment amount
  assertPaymentAmount(amount: string): void {
    cy.get(this.pageSelectors.paymentAmount).should('contain.text', amount);
  }

  // Assert payment method
  assertPaymentMethod(method: string): void {
    cy.get(this.pageSelectors.paymentMethod).should('contain.text', method);
  }

  // Assert payment status
  assertPaymentStatus(status: string): void {
    cy.get(this.pageSelectors.paymentStatus).should('contain.text', status);
  }

  // Get total payments
  getTotalPayments(): Cypress.Chainable<JQuery<HTMLElement>> {
    return cy.get(this.pageSelectors.totalPayments);
  }

  // Assert total payments
  assertTotalPayments(amount: string): void {
    this.getTotalPayments().should('contain.text', amount);
  }

  // Stripe payment form interactions
  getStripeCardElement(): Cypress.Chainable<JQuery<HTMLElement>> {
    return cy.get(this.pageSelectors.cardElement);
  }

  // Fill Stripe card (for testing with Stripe test cards)
  fillStripeCard(cardNumber: string, expiry: string, cvc: string): void {
    // Stripe uses iframes, need special handling
    cy.get(this.pageSelectors.cardElement).within(() => {
      cy.get('iframe').then(($iframe) => {
        const body = $iframe.contents().find('body');
        cy.wrap(body).find('input[name="cardnumber"]').type(cardNumber);
        cy.wrap(body).find('input[name="exp-date"]').type(expiry);
        cy.wrap(body).find('input[name="cvc"]').type(cvc);
      });
    });
  }

  // Click pay with card button
  clickPayWithCard(): void {
    cy.get(this.pageSelectors.payWithCardButton).click();
  }

  // Assert Stripe error
  assertStripeError(message: string): void {
    cy.get(this.pageSelectors.stripeErrors)
      .should('be.visible')
      .and('contain.text', message);
  }

  // Assert payment in list
  assertPaymentInList(identifier: string): void {
    cy.get(`${this.pageSelectors.paymentList}, ${this.selectors.table}`).should(
      'contain.text',
      identifier
    );
  }
}

export default PaymentsPage;
