// =============================================================================
// Invoices Page Object
// =============================================================================

import { BasePage } from './BasePage';

export interface InvoiceLineItem {
  description: string;
  quantity: number;
  unitPrice: number;
}

export interface InvoiceData {
  customerId?: string;
  customerName?: string;
  dueDate?: string;
  issueDate?: string;
  lineItems: InvoiceLineItem[];
  notes?: string;
  paymentTerms?: string;
  discount?: number;
  discountType?: 'percentage' | 'fixed';
  taxRate?: number;
}

export class InvoicesPage extends BasePage {
  private pageSelectors = {
    // List view
    invoiceList: '[data-testid="invoice-list"], .invoice-list',
    invoiceCard: '[data-testid="invoice-card"], .invoice-card',
    invoiceRow: '[data-testid="invoice-row"], tbody tr',

    // Actions
    addInvoiceButton:
      '[data-testid="add-invoice"], button:contains("Add Invoice"), button:contains("New Invoice")',

    // Filters
    statusFilter: '[data-testid="status-filter"]',
    dateFilter: '[data-testid="date-filter"]',
    customerFilter: '[data-testid="customer-filter"]',
    overdueFilter: '[data-testid="overdue-filter"]',

    // Form fields
    customerSelect: '[name="customerId"], [data-testid="customer-select"]',
    customerSearch: '[data-testid="customer-search"]',
    issueDateInput: '[name="issueDate"], [data-testid="issue-date"]',
    dueDateInput: '[name="dueDate"], [data-testid="due-date"]',
    notesInput: '[name="notes"], [data-testid="notes-input"]',
    paymentTermsSelect: '[name="paymentTerms"], [data-testid="payment-terms"]',
    discountInput: '[name="discount"], [data-testid="discount-input"]',
    discountTypeSelect: '[name="discountType"], [data-testid="discount-type"]',
    taxRateInput: '[name="taxRate"], [data-testid="tax-rate"]',

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
    discount: '[data-testid="discount-amount"], .discount',
    total: '[data-testid="total"], .total',
    balanceDue: '[data-testid="balance-due"], .balance-due',

    // Detail view
    invoiceDetail: '[data-testid="invoice-detail"]',
    invoiceNumber: '[data-testid="invoice-number"], .invoice-number',
    invoiceStatus: '[data-testid="invoice-status"], .invoice-status',
    invoiceCustomer: '[data-testid="invoice-customer"], .invoice-customer',
    invoiceAmount: '[data-testid="invoice-amount"], .invoice-amount',
    invoiceDueDate: '[data-testid="invoice-due-date"], .invoice-due-date',

    // Payment info
    paymentHistory: '[data-testid="payment-history"], .payment-history',
    paymentRow: '[data-testid="payment-row"], .payment-item',

    // Actions in detail view
    editButton: '[data-testid="edit-invoice"], button:contains("Edit")',
    deleteButton: '[data-testid="delete-invoice"], button:contains("Delete")',
    sendButton: '[data-testid="send-invoice"], button:contains("Send")',
    recordPaymentButton:
      '[data-testid="record-payment"], button:contains("Record Payment")',
    markPaidButton:
      '[data-testid="mark-paid"], button:contains("Mark as Paid")',
    voidButton: '[data-testid="void-invoice"], button:contains("Void")',
    duplicateButton:
      '[data-testid="duplicate-invoice"], button:contains("Duplicate")',
    downloadPdfButton:
      '[data-testid="download-pdf"], button:contains("Download PDF")',
    printButton: '[data-testid="print-invoice"], button:contains("Print")',
    sendReminderButton:
      '[data-testid="send-reminder"], button:contains("Send Reminder")',

    // Payment modal
    paymentModal: '[data-testid="payment-modal"]',
    paymentAmountInput: '[name="amount"], [data-testid="payment-amount"]',
    paymentMethodSelect:
      '[name="paymentMethod"], [data-testid="payment-method"]',
    paymentDateInput: '[name="paymentDate"], [data-testid="payment-date"]',
    paymentReferenceInput:
      '[name="reference"], [data-testid="payment-reference"]',
    paymentNotesInput: '[name="paymentNotes"], [data-testid="payment-notes"]',
  };

  // Visit invoices list
  visit(): void {
    super.visit('/invoices');
  }

  // Visit invoice detail page
  visitInvoice(invoiceId: string): void {
    super.visit(`/invoices/${invoiceId}`);
  }

  // Click add invoice button
  clickAddInvoice(): void {
    cy.get(this.pageSelectors.addInvoiceButton).click();
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

  // Fill invoice form
  fillInvoiceForm(data: InvoiceData): void {
    if (data.customerName) {
      this.selectCustomer(data.customerName);
    }

    if (data.issueDate) {
      cy.get(this.pageSelectors.issueDateInput).clear().type(data.issueDate);
    }

    if (data.dueDate) {
      cy.get(this.pageSelectors.dueDateInput).clear().type(data.dueDate);
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

    if (data.taxRate) {
      cy.get(this.pageSelectors.taxRateInput)
        .clear()
        .type(data.taxRate.toString());
    }

    if (data.notes) {
      cy.get(this.pageSelectors.notesInput).clear().type(data.notes);
    }

    if (data.paymentTerms) {
      cy.get(this.pageSelectors.paymentTermsSelect).select(data.paymentTerms);
    }
  }

  // Add line item
  addLineItem(item: InvoiceLineItem): void {
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

  // Create invoice (full flow)
  createInvoice(data: InvoiceData): void {
    this.clickAddInvoice();
    this.fillInvoiceForm(data);
    this.submitForm();
  }

  // Get invoices
  getInvoices(): Cypress.Chainable<JQuery<HTMLElement>> {
    return cy.get(
      `${this.pageSelectors.invoiceCard}, ${this.pageSelectors.invoiceRow}`
    );
  }

  // Click invoice to view details
  clickInvoice(identifier: string): void {
    cy.get(
      `${this.pageSelectors.invoiceCard}, ${this.pageSelectors.invoiceRow}`
    )
      .contains(identifier)
      .click();
  }

  // Filter by status
  filterByStatus(status: string): void {
    cy.get(this.pageSelectors.statusFilter).click();
    cy.get('[role="listbox"], .dropdown-menu').contains(status).click();
  }

  // Filter overdue invoices
  filterOverdue(): void {
    cy.get(this.pageSelectors.overdueFilter).click();
  }

  // Click edit button
  clickEdit(): void {
    cy.get(this.pageSelectors.editButton).click();
  }

  // Click delete button
  clickDelete(): void {
    cy.get(this.pageSelectors.deleteButton).click();
  }

  // Send invoice to customer
  clickSend(): void {
    cy.get(this.pageSelectors.sendButton).click();
  }

  // Record payment
  clickRecordPayment(): void {
    cy.get(this.pageSelectors.recordPaymentButton).click();
  }

  // Fill payment form
  fillPaymentForm(
    amount: number,
    method: string,
    date?: string,
    reference?: string
  ): void {
    cy.get(this.pageSelectors.paymentAmountInput)
      .clear()
      .type(amount.toFixed(2));
    cy.get(this.pageSelectors.paymentMethodSelect).select(method);

    if (date) {
      cy.get(this.pageSelectors.paymentDateInput).clear().type(date);
    }

    if (reference) {
      cy.get(this.pageSelectors.paymentReferenceInput).clear().type(reference);
    }
  }

  // Record payment (full flow)
  recordPayment(
    amount: number,
    method: string,
    date?: string,
    reference?: string
  ): void {
    this.clickRecordPayment();
    this.fillPaymentForm(amount, method, date, reference);
    this.submitForm();
  }

  // Mark as paid
  clickMarkPaid(): void {
    cy.get(this.pageSelectors.markPaidButton).click();
  }

  // Void invoice
  clickVoid(): void {
    cy.get(this.pageSelectors.voidButton).click();
  }

  // Duplicate invoice
  clickDuplicate(): void {
    cy.get(this.pageSelectors.duplicateButton).click();
  }

  // Download PDF
  clickDownloadPdf(): void {
    cy.get(this.pageSelectors.downloadPdfButton).click();
  }

  // Print invoice
  clickPrint(): void {
    cy.get(this.pageSelectors.printButton).click();
  }

  // Send reminder
  clickSendReminder(): void {
    cy.get(this.pageSelectors.sendReminderButton).click();
  }

  // Assert invoice status
  assertInvoiceStatus(status: string): void {
    cy.get(this.pageSelectors.invoiceStatus).should('contain.text', status);
  }

  // Assert invoice total
  assertTotal(amount: string): void {
    cy.get(this.pageSelectors.total).should('contain.text', amount);
  }

  // Assert balance due
  assertBalanceDue(amount: string): void {
    cy.get(this.pageSelectors.balanceDue).should('contain.text', amount);
  }

  // Get payment history
  getPayments(): Cypress.Chainable<JQuery<HTMLElement>> {
    return cy
      .get(this.pageSelectors.paymentHistory)
      .find(this.pageSelectors.paymentRow);
  }

  // Assert payment count
  assertPaymentCount(count: number): void {
    this.getPayments().should('have.length', count);
  }

  // Assert invoice in list
  assertInvoiceInList(identifier: string): void {
    cy.get(`${this.pageSelectors.invoiceList}, ${this.selectors.table}`).should(
      'contain.text',
      identifier
    );
  }
}

export default InvoicesPage;
