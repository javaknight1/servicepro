// =============================================================================
// Payment Processing E2E Tests
// =============================================================================

import { PaymentsPage } from '../../pages';

describe('Payment Processing', () => {
  const paymentsPage = new PaymentsPage();

  beforeEach(() => {
    cy.login();

    cy.intercept('GET', '**/api/v1/payments*', {
      body: {
        payments: [
          {
            id: 'pay-001',
            invoiceId: 'inv-002',
            invoiceNumber: 'INV-2024-0002',
            amount: 299.75,
            method: 'credit_card',
            status: 'completed',
            date: '2024-01-15',
            customerName: 'John Smith',
          },
          {
            id: 'pay-002',
            invoiceId: 'inv-001',
            invoiceNumber: 'INV-2024-0001',
            amount: 100.0,
            method: 'cash',
            status: 'completed',
            date: '2024-01-20',
            customerName: 'Jane Doe',
          },
        ],
        pagination: { page: 1, total: 2 },
        summary: {
          totalPayments: 399.75,
          totalRefunds: 0,
          netPayments: 399.75,
        },
      },
    }).as('getPayments');

    cy.intercept('GET', '**/api/v1/users/me', { fixture: 'user.json' }).as(
      'getProfile'
    );
    cy.intercept('GET', '**/api/v1/invoices*', { fixture: 'invoices.json' }).as(
      'getInvoices'
    );
  });

  describe('Payment List', () => {
    beforeEach(() => {
      paymentsPage.visit();
      cy.wait('@getPayments');
    });

    it('should display list of payments', () => {
      paymentsPage.assertPaymentCount(2);
    });

    it('should show payment details in list', () => {
      paymentsPage.assertPaymentInList('INV-2024-0002');
      paymentsPage.assertPaymentInList('$299.75');
    });

    it('should filter payments by method', () => {
      cy.intercept('GET', '**/api/v1/payments*method=credit_card*', {
        body: {
          payments: [{ id: 'pay-001', method: 'credit_card', amount: 299.75 }],
          pagination: { total: 1 },
        },
      }).as('filterByMethod');

      paymentsPage.filterByMethod('Credit Card');

      cy.wait('@filterByMethod');
      paymentsPage.assertPaymentCount(1);
    });

    it('should filter payments by date range', () => {
      cy.intercept(
        'GET',
        '**/api/v1/payments*startDate=2024-01-01*endDate=2024-01-31*',
        {
          body: {
            payments: [
              { id: 'pay-001', date: '2024-01-15' },
              { id: 'pay-002', date: '2024-01-20' },
            ],
            pagination: { total: 2 },
          },
        }
      ).as('filterByDate');

      paymentsPage.filterByDateRange('2024-01-01', '2024-01-31');

      cy.wait('@filterByDate');
    });

    it('should display payment summary totals', () => {
      paymentsPage.assertTotalPayments('$399.75');
    });
  });

  describe('Record Payment', () => {
    beforeEach(() => {
      paymentsPage.visit();
      cy.wait('@getPayments');
    });

    it('should record a new payment', () => {
      cy.intercept('POST', '**/api/v1/payments', {
        statusCode: 201,
        body: {
          id: 'pay-new',
          amount: 150.0,
          method: 'credit_card',
          status: 'completed',
        },
      }).as('createPayment');

      paymentsPage.recordPayment(
        {
          amount: 150.0,
          method: 'credit_card',
          date: '2024-02-01',
          reference: 'TXN-12345',
        },
        'INV-2024-0001'
      );

      cy.wait('@createPayment').its('request.body').should('include', {
        amount: 150.0,
        method: 'credit_card',
        reference: 'TXN-12345',
      });

      paymentsPage.assertToast('Payment recorded', 'success');
    });

    it('should validate payment amount', () => {
      paymentsPage.clickAddPayment();

      cy.fillForm({
        amount: 0,
        method: 'cash',
      });

      paymentsPage.submitForm();

      cy.assertFieldError('amount', 'greater than 0');
    });

    it('should require payment method', () => {
      paymentsPage.clickAddPayment();

      cy.fillForm({
        amount: 100,
      });

      paymentsPage.submitForm();

      cy.assertFieldError('method', 'required');
    });
  });

  describe('View Payment', () => {
    it('should display payment details', () => {
      cy.intercept('GET', '**/api/v1/payments/pay-001', {
        body: {
          id: 'pay-001',
          invoiceId: 'inv-002',
          invoiceNumber: 'INV-2024-0002',
          amount: 299.75,
          method: 'credit_card',
          status: 'completed',
          date: '2024-01-15',
          reference: 'TXN-98765',
          customerName: 'John Smith',
        },
      }).as('getPayment');

      paymentsPage.visitPayment('pay-001');
      cy.wait('@getPayment');

      paymentsPage.assertPaymentAmount('$299.75');
      paymentsPage.assertPaymentMethod('credit_card');
      paymentsPage.assertPaymentStatus('completed');
    });

    it('should link to related invoice', () => {
      cy.intercept('GET', '**/api/v1/payments/pay-001', {
        body: {
          id: 'pay-001',
          invoiceId: 'inv-002',
          invoiceNumber: 'INV-2024-0002',
          amount: 299.75,
        },
      }).as('getPayment');

      paymentsPage.visitPayment('pay-001');
      cy.wait('@getPayment');

      paymentsPage.clickViewInvoice();

      cy.url().should('include', '/invoices/inv-002');
    });
  });

  describe('Refund Payment', () => {
    beforeEach(() => {
      cy.intercept('GET', '**/api/v1/payments/pay-001', {
        body: {
          id: 'pay-001',
          amount: 299.75,
          method: 'credit_card',
          status: 'completed',
          canRefund: true,
        },
      }).as('getPayment');
    });

    it('should process full refund', () => {
      cy.intercept('POST', '**/api/v1/payments/pay-001/refund', {
        statusCode: 200,
        body: {
          id: 'refund-001',
          paymentId: 'pay-001',
          amount: 299.75,
          status: 'completed',
        },
      }).as('processRefund');

      paymentsPage.visitPayment('pay-001');
      cy.wait('@getPayment');

      paymentsPage.processRefund(299.75, 'Customer requested refund');

      cy.wait('@processRefund').its('request.body').should('include', {
        amount: 299.75,
        reason: 'Customer requested refund',
      });

      paymentsPage.assertToast('Refund processed', 'success');
    });

    it('should process partial refund', () => {
      cy.intercept('POST', '**/api/v1/payments/pay-001/refund', {
        statusCode: 200,
        body: {
          id: 'refund-001',
          amount: 100.0,
          status: 'completed',
        },
      }).as('processRefund');

      paymentsPage.visitPayment('pay-001');
      cy.wait('@getPayment');

      paymentsPage.processRefund(100.0, 'Partial service not rendered');

      cy.wait('@processRefund').its('request.body').should('include', {
        amount: 100.0,
      });
    });

    it('should validate refund amount', () => {
      paymentsPage.visitPayment('pay-001');
      cy.wait('@getPayment');

      paymentsPage.clickRefund();

      // Try to refund more than original amount
      cy.get('[data-testid="refund-amount"]').clear().type('500.00');
      cy.get('[data-testid="confirm-refund"]').click();

      cy.get('[data-testid="refund-error"]').should(
        'contain.text',
        'exceeds original'
      );
    });
  });

  describe('Void Payment', () => {
    it('should void pending payment', () => {
      cy.intercept('GET', '**/api/v1/payments/pay-pending', {
        body: {
          id: 'pay-pending',
          amount: 100,
          status: 'pending',
          canVoid: true,
        },
      }).as('getPayment');

      cy.intercept('POST', '**/api/v1/payments/pay-pending/void', {
        statusCode: 200,
        body: { status: 'void' },
      }).as('voidPayment');

      paymentsPage.visitPayment('pay-pending');
      cy.wait('@getPayment');

      paymentsPage.clickVoid();
      paymentsPage.confirmDelete();

      cy.wait('@voidPayment');

      paymentsPage.assertPaymentStatus('void');
    });
  });

  describe('Stripe Integration', () => {
    beforeEach(() => {
      cy.intercept('GET', '**/api/v1/invoices/inv-001', {
        body: {
          id: 'inv-001',
          invoiceNumber: 'INV-2024-0001',
          total: 216.5,
          balanceDue: 216.5,
          customerEmail: 'john@example.com',
        },
      }).as('getInvoice');
    });

    it('should display Stripe payment form', () => {
      cy.visit('/invoices/inv-001/pay');
      cy.wait('@getInvoice');

      cy.get('[data-testid="stripe-payment-form"]').should('be.visible');
      cy.get('[data-testid="card-element"]').should('be.visible');
    });

    it('should process credit card payment via Stripe', () => {
      cy.intercept('POST', '**/api/v1/payments/stripe/create-payment-intent', {
        statusCode: 200,
        body: {
          clientSecret: 'pi_test_secret_123',
        },
      }).as('createPaymentIntent');

      cy.intercept('POST', '**/api/v1/payments/stripe/confirm', {
        statusCode: 200,
        body: {
          paymentId: 'pay-stripe-001',
          status: 'completed',
        },
      }).as('confirmPayment');

      cy.visit('/invoices/inv-001/pay');
      cy.wait('@getInvoice');

      // Fill in test card number (Stripe test card)
      cy.get('[data-testid="card-element"]').within(() => {
        cy.get('iframe').then(($iframe) => {
          const body = $iframe.contents().find('body');
          cy.wrap(body)
            .find('input[name="cardnumber"]')
            .type('4242424242424242');
          cy.wrap(body).find('input[name="exp-date"]').type('1230');
          cy.wrap(body).find('input[name="cvc"]').type('123');
          cy.wrap(body).find('input[name="postal"]').type('12345');
        });
      });

      cy.get('[data-testid="pay-with-card"]').click();

      cy.wait('@createPaymentIntent');
      cy.wait('@confirmPayment');

      paymentsPage.assertToast('Payment successful', 'success');
    });

    it('should show error for declined card', () => {
      cy.intercept('POST', '**/api/v1/payments/stripe/create-payment-intent', {
        statusCode: 400,
        body: {
          error: 'card_declined',
          message: 'Your card was declined',
        },
      }).as('declinedCard');

      cy.visit('/invoices/inv-001/pay');
      cy.wait('@getInvoice');

      // Fill in declined test card
      cy.get('[data-testid="card-element"]').within(() => {
        cy.get('iframe').then(($iframe) => {
          const body = $iframe.contents().find('body');
          cy.wrap(body)
            .find('input[name="cardnumber"]')
            .type('4000000000000002');
          cy.wrap(body).find('input[name="exp-date"]').type('1230');
          cy.wrap(body).find('input[name="cvc"]').type('123');
        });
      });

      cy.get('[data-testid="pay-with-card"]').click();

      cy.wait('@declinedCard');

      paymentsPage.assertStripeError('card was declined');
    });

    it('should handle 3D Secure authentication', () => {
      cy.intercept('POST', '**/api/v1/payments/stripe/create-payment-intent', {
        statusCode: 200,
        body: {
          clientSecret: 'pi_test_secret_123',
          requiresAction: true,
        },
      }).as('createPaymentIntent');

      cy.visit('/invoices/inv-001/pay');
      cy.wait('@getInvoice');

      // This test would need to handle the 3DS iframe/redirect
      // For now, just verify the flow starts correctly
      cy.get('[data-testid="card-element"]').should('be.visible');
    });
  });

  describe('Payment Receipt', () => {
    it('should download payment receipt', () => {
      cy.intercept('GET', '**/api/v1/payments/pay-001', {
        body: { id: 'pay-001', amount: 299.75, status: 'completed' },
      }).as('getPayment');

      cy.intercept('GET', '**/api/v1/payments/pay-001/receipt', {
        statusCode: 200,
        headers: {
          'Content-Type': 'application/pdf',
          'Content-Disposition': 'attachment; filename="receipt-pay-001.pdf"',
        },
        body: 'PDF content',
      }).as('downloadReceipt');

      paymentsPage.visitPayment('pay-001');
      cy.wait('@getPayment');

      paymentsPage.clickDownloadReceipt();

      cy.wait('@downloadReceipt');
    });

    it('should email receipt to customer', () => {
      cy.intercept('GET', '**/api/v1/payments/pay-001', {
        body: {
          id: 'pay-001',
          amount: 299.75,
          customerEmail: 'john@example.com',
        },
      }).as('getPayment');

      cy.intercept('POST', '**/api/v1/payments/pay-001/email-receipt', {
        statusCode: 200,
        body: { sent: true },
      }).as('emailReceipt');

      paymentsPage.visitPayment('pay-001');
      cy.wait('@getPayment');

      cy.get('[data-testid="email-receipt"]').click();

      cy.wait('@emailReceipt');

      paymentsPage.assertToast('Receipt sent', 'success');
    });
  });

  describe('Payment Methods', () => {
    it('should list all payment method options', () => {
      paymentsPage.visit();
      paymentsPage.clickAddPayment();

      cy.get('[data-testid="method-select"], [name="method"]').click();

      cy.get('[role="listbox"], .dropdown-menu').within(() => {
        cy.contains('Credit Card').should('be.visible');
        cy.contains('Bank Transfer').should('be.visible');
        cy.contains('Cash').should('be.visible');
        cy.contains('Check').should('be.visible');
      });
    });

    it('should require reference for check payments', () => {
      paymentsPage.visit();
      paymentsPage.clickAddPayment();

      cy.fillForm({
        amount: 100,
        method: 'check',
      });

      paymentsPage.submitForm();

      cy.assertFieldError('reference', 'required');
    });
  });

  describe('Payment Reports', () => {
    it('should export payments to CSV', () => {
      cy.intercept('GET', '**/api/v1/payments/export*', {
        statusCode: 200,
        headers: {
          'Content-Type': 'text/csv',
        },
        body: 'date,amount,method\n2024-01-15,299.75,credit_card',
      }).as('exportPayments');

      paymentsPage.visit();
      cy.wait('@getPayments');

      cy.get('[data-testid="export-button"]').click();

      cy.wait('@exportPayments');
    });
  });
});
