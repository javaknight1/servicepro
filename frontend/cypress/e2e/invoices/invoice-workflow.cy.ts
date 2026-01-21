// =============================================================================
// Invoice Workflow E2E Tests
// =============================================================================

import { InvoicesPage } from '../../pages';

describe('Invoice Workflow', () => {
  const invoicesPage = new InvoicesPage();

  beforeEach(() => {
    cy.login();

    cy.intercept('GET', '**/api/v1/invoices*', { fixture: 'invoices.json' }).as(
      'getInvoices'
    );
    cy.intercept('GET', '**/api/v1/users/me', { fixture: 'user.json' }).as(
      'getProfile'
    );
    cy.intercept('GET', '**/api/v1/customers*', {
      fixture: 'customers.json',
    }).as('getCustomers');
  });

  describe('Invoice List', () => {
    beforeEach(() => {
      invoicesPage.visit();
      cy.wait('@getInvoices');
    });

    it('should display list of invoices', () => {
      invoicesPage.getInvoices().should('have.length', 4);
    });

    it('should show invoice status badges', () => {
      cy.contains('INV-2024-0001')
        .parents('tr, [data-testid="invoice-row"]')
        .should('contain.text', 'sent');

      cy.contains('INV-2024-0002')
        .parents('tr, [data-testid="invoice-row"]')
        .should('contain.text', 'paid');

      cy.contains('INV-2024-0004')
        .parents('tr, [data-testid="invoice-row"]')
        .should('contain.text', 'overdue');
    });

    it('should filter overdue invoices', () => {
      cy.intercept('GET', '**/api/v1/invoices*status=overdue*', {
        body: {
          invoices: [
            {
              id: 'inv-004',
              invoiceNumber: 'INV-2024-0004',
              status: 'overdue',
            },
          ],
          pagination: { total: 1 },
        },
      }).as('filterOverdue');

      invoicesPage.filterByStatus('Overdue');

      cy.wait('@filterOverdue');
      invoicesPage.getInvoices().should('have.length', 1);
    });

    it('should search invoices by number', () => {
      cy.intercept('GET', '**/api/v1/invoices*q=INV-2024-0002*', {
        body: {
          invoices: [
            { id: 'inv-002', invoiceNumber: 'INV-2024-0002', status: 'paid' },
          ],
          pagination: { total: 1 },
        },
      }).as('searchInvoices');

      invoicesPage.search('INV-2024-0002');

      cy.wait('@searchInvoices');
      invoicesPage.assertInvoiceInList('INV-2024-0002');
    });
  });

  describe('Create Invoice', () => {
    beforeEach(() => {
      invoicesPage.visit();
      cy.wait('@getInvoices');
    });

    it('should create a new invoice', () => {
      const today = new Date().toISOString().split('T')[0];
      const dueDate = new Date(Date.now() + 30 * 24 * 60 * 60 * 1000)
        .toISOString()
        .split('T')[0];

      cy.intercept('POST', '**/api/v1/invoices', {
        statusCode: 201,
        body: {
          id: 'inv-new',
          invoiceNumber: 'INV-2024-0005',
          status: 'draft',
          total: 216.5,
        },
      }).as('createInvoice');

      invoicesPage.createInvoice({
        customerName: 'John Smith',
        issueDate: today,
        dueDate: dueDate,
        lineItems: [
          { description: 'Service Call', quantity: 1, unitPrice: 100 },
          { description: 'Parts', quantity: 2, unitPrice: 50 },
        ],
        notes: 'Thank you for your business!',
      });

      cy.wait('@createInvoice');

      invoicesPage.assertToast('Invoice created', 'success');
    });

    it('should calculate totals with tax', () => {
      invoicesPage.clickAddInvoice();

      invoicesPage.selectCustomer('John Smith');

      // Add line items
      cy.get('[data-testid="item-description"]').first().type('Service');
      cy.get('[data-testid="item-quantity"]').first().clear().type('1');
      cy.get('[data-testid="item-price"]').first().clear().type('100.00');

      // Check subtotal
      cy.get('[data-testid="subtotal"]').should('contain.text', '100.00');

      // Tax should be applied (8.25%)
      cy.get('[data-testid="tax"]').should('contain.text', '8.25');

      // Total should include tax
      cy.get('[data-testid="total"]').should('contain.text', '108.25');
    });
  });

  describe('Invoice Actions', () => {
    beforeEach(() => {
      cy.intercept('GET', '**/api/v1/invoices/inv-001', {
        body: {
          id: 'inv-001',
          invoiceNumber: 'INV-2024-0001',
          status: 'draft',
          customerName: 'John Smith',
          customerEmail: 'john@example.com',
          total: 216.5,
          balanceDue: 216.5,
          lineItems: [
            {
              description: 'Pool Cleaning',
              quantity: 1,
              unitPrice: 150,
              total: 150,
            },
          ],
        },
      }).as('getInvoice');
    });

    it('should send invoice to customer', () => {
      cy.intercept('POST', '**/api/v1/invoices/inv-001/send', {
        statusCode: 200,
        body: { status: 'sent', sentAt: new Date().toISOString() },
      }).as('sendInvoice');

      invoicesPage.visitInvoice('inv-001');
      cy.wait('@getInvoice');

      invoicesPage.clickSend();

      cy.get('[data-testid="send-confirmation"]').should('be.visible');
      cy.contains('john@example.com').should('be.visible');
      cy.get('[data-testid="confirm-send"]').click();

      cy.wait('@sendInvoice');

      invoicesPage.assertToast('Invoice sent', 'success');
      invoicesPage.assertInvoiceStatus('sent');
    });

    it('should download invoice as PDF', () => {
      cy.intercept('GET', '**/api/v1/invoices/inv-001/pdf', {
        statusCode: 200,
        headers: {
          'Content-Type': 'application/pdf',
        },
        body: 'PDF content',
      }).as('downloadPdf');

      invoicesPage.visitInvoice('inv-001');
      cy.wait('@getInvoice');

      invoicesPage.clickDownloadPdf();

      cy.wait('@downloadPdf');
    });

    it('should void invoice', () => {
      cy.intercept('POST', '**/api/v1/invoices/inv-001/void', {
        statusCode: 200,
        body: { status: 'void' },
      }).as('voidInvoice');

      invoicesPage.visitInvoice('inv-001');
      cy.wait('@getInvoice');

      invoicesPage.clickVoid();
      invoicesPage.confirmDelete();

      cy.wait('@voidInvoice');

      invoicesPage.assertInvoiceStatus('void');
    });

    it('should duplicate invoice', () => {
      cy.intercept('POST', '**/api/v1/invoices/inv-001/duplicate', {
        statusCode: 201,
        body: {
          id: 'inv-new',
          invoiceNumber: 'INV-2024-0005',
          status: 'draft',
        },
      }).as('duplicateInvoice');

      invoicesPage.visitInvoice('inv-001');
      cy.wait('@getInvoice');

      invoicesPage.clickDuplicate();

      cy.wait('@duplicateInvoice');

      cy.url().should('include', '/invoices/inv-new');
    });
  });

  describe('Record Payment', () => {
    beforeEach(() => {
      cy.intercept('GET', '**/api/v1/invoices/inv-001', {
        body: {
          id: 'inv-001',
          invoiceNumber: 'INV-2024-0001',
          status: 'sent',
          total: 216.5,
          amountPaid: 0,
          balanceDue: 216.5,
        },
      }).as('getInvoice');
    });

    it('should record full payment', () => {
      cy.intercept('POST', '**/api/v1/invoices/inv-001/payments', {
        statusCode: 201,
        body: {
          id: 'payment-001',
          amount: 216.5,
          method: 'credit_card',
          invoiceStatus: 'paid',
        },
      }).as('recordPayment');

      invoicesPage.visitInvoice('inv-001');
      cy.wait('@getInvoice');

      invoicesPage.recordPayment(216.5, 'credit_card', '2024-02-01', 'REF-001');

      cy.wait('@recordPayment').its('request.body').should('include', {
        amount: 216.5,
        method: 'credit_card',
        reference: 'REF-001',
      });

      invoicesPage.assertToast('Payment recorded', 'success');
      invoicesPage.assertInvoiceStatus('paid');
    });

    it('should record partial payment', () => {
      cy.intercept('POST', '**/api/v1/invoices/inv-001/payments', {
        statusCode: 201,
        body: {
          id: 'payment-001',
          amount: 100,
          invoiceStatus: 'partial',
        },
      }).as('recordPayment');

      cy.intercept('GET', '**/api/v1/invoices/inv-001', {
        body: {
          id: 'inv-001',
          status: 'partial',
          total: 216.5,
          amountPaid: 100,
          balanceDue: 116.5,
        },
      }).as('getUpdatedInvoice');

      invoicesPage.visitInvoice('inv-001');
      cy.wait('@getInvoice');

      invoicesPage.recordPayment(100, 'cash');

      cy.wait('@recordPayment');

      invoicesPage.assertInvoiceStatus('partial');
      invoicesPage.assertBalanceDue('116.50');
    });

    it('should mark invoice as paid', () => {
      cy.intercept('POST', '**/api/v1/invoices/inv-001/mark-paid', {
        statusCode: 200,
        body: { status: 'paid', paidAt: new Date().toISOString() },
      }).as('markPaid');

      invoicesPage.visitInvoice('inv-001');
      cy.wait('@getInvoice');

      invoicesPage.clickMarkPaid();

      cy.wait('@markPaid');

      invoicesPage.assertInvoiceStatus('paid');
    });

    it('should display payment history', () => {
      cy.intercept('GET', '**/api/v1/invoices/inv-001', {
        body: {
          id: 'inv-001',
          status: 'partial',
          total: 300,
          amountPaid: 200,
          balanceDue: 100,
          payments: [
            { id: 'pay-001', amount: 100, method: 'cash', date: '2024-01-15' },
            {
              id: 'pay-002',
              amount: 100,
              method: 'credit_card',
              date: '2024-01-20',
            },
          ],
        },
      }).as('getInvoice');

      invoicesPage.visitInvoice('inv-001');
      cy.wait('@getInvoice');

      invoicesPage.assertPaymentCount(2);
    });
  });

  describe('Invoice Reminders', () => {
    it('should send payment reminder', () => {
      cy.intercept('GET', '**/api/v1/invoices/inv-004', {
        body: {
          id: 'inv-004',
          invoiceNumber: 'INV-2024-0004',
          status: 'overdue',
          customerEmail: 'bob@example.com',
          total: 514.19,
          balanceDue: 514.19,
        },
      }).as('getInvoice');

      cy.intercept('POST', '**/api/v1/invoices/inv-004/send-reminder', {
        statusCode: 200,
        body: { reminderSent: true, sentAt: new Date().toISOString() },
      }).as('sendReminder');

      invoicesPage.visitInvoice('inv-004');
      cy.wait('@getInvoice');

      invoicesPage.clickSendReminder();

      cy.wait('@sendReminder');

      invoicesPage.assertToast('Reminder sent', 'success');
    });
  });

  describe('Overdue Invoices', () => {
    it('should highlight overdue invoices in list', () => {
      invoicesPage.visit();
      cy.wait('@getInvoices');

      cy.contains('INV-2024-0004')
        .parents('tr, [data-testid="invoice-row"]')
        .should('have.class', 'overdue')
        .or('have.css', 'background-color');
    });

    it('should show overdue badge on invoice detail', () => {
      cy.intercept('GET', '**/api/v1/invoices/inv-004', {
        body: {
          id: 'inv-004',
          status: 'overdue',
          dueDate: '2024-01-15',
          total: 514.19,
        },
      }).as('getInvoice');

      invoicesPage.visitInvoice('inv-004');
      cy.wait('@getInvoice');

      cy.get('[data-testid="overdue-badge"], .overdue-indicator').should(
        'be.visible'
      );
    });
  });

  describe('Invoice from Job', () => {
    it('should pre-populate from job details', () => {
      cy.intercept('GET', '**/api/v1/jobs/job-002', {
        body: {
          id: 'job-002',
          title: 'Pool Cleaning',
          customerId: 'cust-001',
          customerName: 'John Smith',
          price: 150,
          completedAt: '2024-01-25',
        },
      }).as('getJob');

      cy.visit('/invoices/new?jobId=job-002');
      cy.wait('@getJob');

      // Customer should be pre-selected
      cy.get('[data-testid="customer-select"]').should(
        'contain.text',
        'John Smith'
      );

      // Line item should be pre-filled
      cy.get('[data-testid="item-description"]')
        .first()
        .should('have.value', 'Pool Cleaning');

      cy.get('[data-testid="item-price"]')
        .first()
        .should('have.value', '150.00');
    });
  });

  describe('Recurring Invoices', () => {
    it('should create recurring invoice', () => {
      cy.intercept('POST', '**/api/v1/invoices', {
        statusCode: 201,
        body: {
          id: 'inv-recurring',
          isRecurring: true,
          recurrencePattern: 'monthly',
        },
      }).as('createRecurringInvoice');

      invoicesPage.visit();
      invoicesPage.clickAddInvoice();

      invoicesPage.selectCustomer('John Smith');

      // Add line item
      cy.get('[data-testid="item-description"]')
        .first()
        .type('Monthly Service');
      cy.get('[data-testid="item-quantity"]').first().clear().type('1');
      cy.get('[data-testid="item-price"]').first().clear().type('100.00');

      // Enable recurring
      cy.get('[data-testid="recurring-toggle"]').click();
      cy.get('[data-testid="recurrence-pattern"]').select('monthly');

      invoicesPage.submitForm();

      cy.wait('@createRecurringInvoice').its('request.body').should('include', {
        isRecurring: true,
        recurrencePattern: 'monthly',
      });
    });
  });

  describe('Print Invoice', () => {
    it('should open print dialog', () => {
      cy.intercept('GET', '**/api/v1/invoices/inv-001', {
        body: { id: 'inv-001', invoiceNumber: 'INV-2024-0001' },
      }).as('getInvoice');

      cy.window().then((win) => {
        cy.stub(win, 'print').as('printDialog');
      });

      invoicesPage.visitInvoice('inv-001');
      cy.wait('@getInvoice');

      invoicesPage.clickPrint();

      cy.get('@printDialog').should('have.been.called');
    });
  });
});
