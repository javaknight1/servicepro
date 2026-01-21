// =============================================================================
// Quote Workflow E2E Tests
// =============================================================================

import { QuotesPage } from '../../pages';

describe('Quote Workflow', () => {
  const quotesPage = new QuotesPage();

  beforeEach(() => {
    cy.login();

    cy.intercept('GET', '**/api/v1/quotes*', { fixture: 'quotes.json' }).as(
      'getQuotes'
    );
    cy.intercept('GET', '**/api/v1/users/me', { fixture: 'user.json' }).as(
      'getProfile'
    );
    cy.intercept('GET', '**/api/v1/customers*', {
      fixture: 'customers.json',
    }).as('getCustomers');
  });

  describe('Quote List', () => {
    beforeEach(() => {
      quotesPage.visit();
      cy.wait('@getQuotes');
    });

    it('should display list of quotes', () => {
      quotesPage.getQuotes().should('have.length', 4);
    });

    it('should display quote status badges', () => {
      cy.contains('QT-2024-0001')
        .parents('tr, [data-testid="quote-row"]')
        .should('contain.text', 'sent');

      cy.contains('QT-2024-0002')
        .parents('tr, [data-testid="quote-row"]')
        .should('contain.text', 'approved');
    });

    it('should filter quotes by status', () => {
      cy.intercept('GET', '**/api/v1/quotes*status=approved*', {
        body: {
          quotes: [
            {
              id: 'quote-002',
              quoteNumber: 'QT-2024-0002',
              status: 'approved',
            },
          ],
          pagination: { total: 1 },
        },
      }).as('filterQuotes');

      quotesPage.filterByStatus('Approved');

      cy.wait('@filterQuotes');
      quotesPage.getQuotes().should('have.length', 1);
    });
  });

  describe('Create Quote', () => {
    beforeEach(() => {
      quotesPage.visit();
      cy.wait('@getQuotes');
    });

    it('should create a new quote', () => {
      const newQuote = {
        customerName: 'John Smith',
        title: 'New Service Quote',
        validUntil: '2024-03-15',
        lineItems: [
          { description: 'Service Item 1', quantity: 2, unitPrice: 100 },
          { description: 'Service Item 2', quantity: 1, unitPrice: 50 },
        ],
        notes: 'Quote notes',
      };

      cy.intercept('POST', '**/api/v1/quotes', {
        statusCode: 201,
        body: {
          id: 'quote-new',
          quoteNumber: 'QT-2024-0005',
          ...newQuote,
          status: 'draft',
          subtotal: 250,
          total: 270.63,
        },
      }).as('createQuote');

      quotesPage.createQuote(newQuote);

      cy.wait('@createQuote')
        .its('request.body.lineItems')
        .should('have.length', 2);

      quotesPage.assertToast('Quote created', 'success');
    });

    it('should calculate totals automatically', () => {
      quotesPage.clickAddQuote();

      quotesPage.selectCustomer('John Smith');

      // Add first line item
      cy.get('[data-testid="item-description"]').first().type('Item 1');
      cy.get('[data-testid="item-quantity"]').first().clear().type('2');
      cy.get('[data-testid="item-price"]').first().clear().type('100.00');

      // Check subtotal updates
      quotesPage.assertTotal('$200.00');

      // Add another line item
      quotesPage.addLineItem({
        description: 'Item 2',
        quantity: 1,
        unitPrice: 50,
      });

      quotesPage.assertTotal('$250.00');
    });

    it('should apply discount', () => {
      quotesPage.clickAddQuote();

      quotesPage.selectCustomer('John Smith');

      // Add line item
      cy.get('[data-testid="item-description"]').first().type('Service');
      cy.get('[data-testid="item-quantity"]').first().clear().type('1');
      cy.get('[data-testid="item-price"]').first().clear().type('100.00');

      // Apply 10% discount
      cy.get('[data-testid="discount-input"]').type('10');
      cy.get('[data-testid="discount-type"]').select('percentage');

      // Total should be 90 + tax
      cy.get('[data-testid="subtotal"]').should('contain.text', '90');
    });
  });

  describe('Quote Actions', () => {
    beforeEach(() => {
      cy.intercept('GET', '**/api/v1/quotes/quote-001', {
        body: {
          id: 'quote-001',
          quoteNumber: 'QT-2024-0001',
          status: 'draft',
          customerName: 'John Smith',
          customerEmail: 'john@example.com',
          total: 5553.23,
          lineItems: [
            {
              description: 'Service',
              quantity: 1,
              unitPrice: 5000,
              total: 5000,
            },
          ],
        },
      }).as('getQuote');
    });

    it('should send quote to customer', () => {
      cy.intercept('POST', '**/api/v1/quotes/quote-001/send', {
        statusCode: 200,
        body: { status: 'sent', sentAt: new Date().toISOString() },
      }).as('sendQuote');

      quotesPage.visitQuote('quote-001');
      cy.wait('@getQuote');

      quotesPage.clickSend();

      cy.get('[data-testid="send-confirmation"]').should('be.visible');
      cy.get('[data-testid="confirm-send"]').click();

      cy.wait('@sendQuote');

      quotesPage.assertToast('sent', 'success');
      quotesPage.assertQuoteStatus('sent');
    });

    it('should preview quote as PDF', () => {
      quotesPage.visitQuote('quote-001');
      cy.wait('@getQuote');

      quotesPage.clickPreview();

      cy.get('[data-testid="preview-modal"]').should('be.visible');
    });

    it('should download quote as PDF', () => {
      cy.intercept('GET', '**/api/v1/quotes/quote-001/pdf', {
        statusCode: 200,
        headers: {
          'Content-Type': 'application/pdf',
          'Content-Disposition': 'attachment; filename="QT-2024-0001.pdf"',
        },
        body: 'PDF content',
      }).as('downloadPdf');

      quotesPage.visitQuote('quote-001');
      cy.wait('@getQuote');

      quotesPage.clickDownloadPdf();

      cy.wait('@downloadPdf');
    });

    it('should duplicate quote', () => {
      cy.intercept('POST', '**/api/v1/quotes/quote-001/duplicate', {
        statusCode: 201,
        body: {
          id: 'quote-new',
          quoteNumber: 'QT-2024-0005',
          status: 'draft',
        },
      }).as('duplicateQuote');

      quotesPage.visitQuote('quote-001');
      cy.wait('@getQuote');

      quotesPage.clickDuplicate();

      cy.wait('@duplicateQuote');

      cy.url().should('include', '/quotes/quote-new');
    });
  });

  describe('Quote Status Workflow', () => {
    it('should approve sent quote', () => {
      cy.intercept('GET', '**/api/v1/quotes/quote-001', {
        body: { id: 'quote-001', status: 'sent', total: 1000 },
      }).as('getQuote');

      cy.intercept('POST', '**/api/v1/quotes/quote-001/approve', {
        statusCode: 200,
        body: { status: 'approved', approvedAt: new Date().toISOString() },
      }).as('approveQuote');

      quotesPage.visitQuote('quote-001');
      cy.wait('@getQuote');

      quotesPage.clickApprove();

      cy.wait('@approveQuote');

      quotesPage.assertQuoteStatus('approved');
    });

    it('should reject quote with reason', () => {
      cy.intercept('GET', '**/api/v1/quotes/quote-001', {
        body: { id: 'quote-001', status: 'sent', total: 1000 },
      }).as('getQuote');

      cy.intercept('POST', '**/api/v1/quotes/quote-001/reject', {
        statusCode: 200,
        body: { status: 'rejected', rejectionReason: 'Too expensive' },
      }).as('rejectQuote');

      quotesPage.visitQuote('quote-001');
      cy.wait('@getQuote');

      quotesPage.clickReject();

      // Enter rejection reason
      cy.get('[data-testid="rejection-reason"]').type('Too expensive');
      cy.get('[data-testid="confirm-reject"]').click();

      cy.wait('@rejectQuote').its('request.body').should('include', {
        reason: 'Too expensive',
      });

      quotesPage.assertQuoteStatus('rejected');
    });
  });

  describe('Convert Quote', () => {
    beforeEach(() => {
      cy.intercept('GET', '**/api/v1/quotes/quote-002', {
        body: {
          id: 'quote-002',
          quoteNumber: 'QT-2024-0002',
          status: 'approved',
          customerId: 'cust-002',
          customerName: 'Jane Doe',
          total: 5358.38,
          lineItems: [
            {
              description: 'AC Unit',
              quantity: 1,
              unitPrice: 3500,
              total: 3500,
            },
            {
              description: 'Installation',
              quantity: 8,
              unitPrice: 125,
              total: 1000,
            },
          ],
        },
      }).as('getQuote');
    });

    it('should convert approved quote to invoice', () => {
      cy.intercept('POST', '**/api/v1/quotes/quote-002/convert-to-invoice', {
        statusCode: 201,
        body: {
          id: 'inv-new',
          invoiceNumber: 'INV-2024-0005',
          quoteId: 'quote-002',
          total: 5358.38,
        },
      }).as('convertToInvoice');

      quotesPage.visitQuote('quote-002');
      cy.wait('@getQuote');

      quotesPage.clickConvertToInvoice();

      cy.wait('@convertToInvoice');

      cy.url().should('include', '/invoices/inv-new');
      quotesPage.assertToast('Invoice created', 'success');
    });

    it('should convert approved quote to job', () => {
      cy.intercept('POST', '**/api/v1/quotes/quote-002/convert-to-job', {
        statusCode: 201,
        body: {
          id: 'job-new',
          title: 'HVAC System Replacement',
          quoteId: 'quote-002',
        },
      }).as('convertToJob');

      quotesPage.visitQuote('quote-002');
      cy.wait('@getQuote');

      quotesPage.clickConvertToJob();

      cy.wait('@convertToJob');

      cy.url().should('include', '/jobs/job-new');
    });
  });

  describe('Quote Templates', () => {
    it('should create quote from template', () => {
      cy.intercept('GET', '**/api/v1/quote-templates', {
        body: {
          templates: [
            {
              id: 'template-1',
              name: 'Standard Lawn Care Package',
              lineItems: [
                { description: 'Weekly Mowing', quantity: 4, unitPrice: 75 },
              ],
            },
          ],
        },
      }).as('getTemplates');

      quotesPage.visit();
      quotesPage.clickAddQuote();

      cy.get('[data-testid="use-template"]').click();
      cy.wait('@getTemplates');

      cy.contains('Standard Lawn Care Package').click();

      // Line items should be pre-filled
      cy.get('[data-testid="item-description"]')
        .first()
        .should('have.value', 'Weekly Mowing');
    });
  });

  describe('Quote Expiration', () => {
    it('should show warning for expiring quotes', () => {
      const tomorrow = new Date();
      tomorrow.setDate(tomorrow.getDate() + 1);

      cy.intercept('GET', '**/api/v1/quotes/quote-001', {
        body: {
          id: 'quote-001',
          status: 'sent',
          validUntil: tomorrow.toISOString().split('T')[0],
        },
      }).as('getQuote');

      quotesPage.visitQuote('quote-001');
      cy.wait('@getQuote');

      cy.get('[data-testid="expiry-warning"]')
        .should('be.visible')
        .and('contain.text', 'expires');
    });

    it('should show expired status for past quotes', () => {
      cy.intercept('GET', '**/api/v1/quotes/quote-001', {
        body: {
          id: 'quote-001',
          status: 'expired',
          validUntil: '2024-01-01',
        },
      }).as('getQuote');

      quotesPage.visitQuote('quote-001');
      cy.wait('@getQuote');

      quotesPage.assertQuoteStatus('expired');
    });
  });
});
