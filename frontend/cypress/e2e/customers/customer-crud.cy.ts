// =============================================================================
// Customer CRUD E2E Tests
// =============================================================================

import { CustomersPage } from '../../pages';

describe('Customer Management', () => {
  const customersPage = new CustomersPage();

  beforeEach(() => {
    // Login before each test
    cy.login();

    // Set up common stubs
    cy.intercept('GET', '**/api/v1/customers*', {
      fixture: 'customers.json',
    }).as('getCustomers');
    cy.intercept('GET', '**/api/v1/users/me', { fixture: 'user.json' }).as(
      'getProfile'
    );
  });

  describe('Customer List', () => {
    beforeEach(() => {
      customersPage.visit();
      cy.wait('@getCustomers');
    });

    it('should display list of customers', () => {
      customersPage.assertCustomerCount(3);
    });

    it('should display customer information in list', () => {
      customersPage.assertCustomerInList('John Smith');
      customersPage.assertCustomerInList('Jane Doe');
      customersPage.assertCustomerInList('Bob Johnson');
    });

    it('should search customers by name', () => {
      cy.intercept('GET', '**/api/v1/customers*q=John*', {
        body: {
          customers: [
            {
              id: 'cust-001',
              firstName: 'John',
              lastName: 'Smith',
              email: 'john.smith@example.com',
            },
          ],
          pagination: { total: 1 },
        },
      }).as('searchCustomers');

      customersPage.searchCustomer('John');

      cy.wait('@searchCustomers');

      customersPage.assertCustomerInList('John Smith');
      customersPage.assertCustomerNotInList('Jane Doe');
    });

    it('should filter customers by status', () => {
      cy.intercept('GET', '**/api/v1/customers*status=active*', {
        body: {
          customers: [
            {
              id: 'cust-001',
              firstName: 'John',
              lastName: 'Smith',
              status: 'active',
            },
            {
              id: 'cust-002',
              firstName: 'Jane',
              lastName: 'Doe',
              status: 'active',
            },
          ],
          pagination: { total: 2 },
        },
      }).as('filterCustomers');

      customersPage.filterByStatus('Active');

      cy.wait('@filterCustomers');

      customersPage.assertCustomerCount(2);
    });

    it('should sort customers', () => {
      cy.intercept('GET', '**/api/v1/customers*sort=name*', {
        body: {
          customers: [
            { id: 'cust-003', firstName: 'Bob', lastName: 'Johnson' },
            { id: 'cust-002', firstName: 'Jane', lastName: 'Doe' },
            { id: 'cust-001', firstName: 'John', lastName: 'Smith' },
          ],
          pagination: { total: 3 },
        },
      }).as('sortCustomers');

      customersPage.sortBy('Name');

      cy.wait('@sortCustomers');
    });

    it('should show empty state when no customers', () => {
      cy.intercept('GET', '**/api/v1/customers*', {
        body: {
          customers: [],
          pagination: { total: 0 },
        },
      }).as('emptyCustomers');

      customersPage.visit();
      cy.wait('@emptyCustomers');

      customersPage.assertEmptyState();
    });

    it('should paginate customers', () => {
      cy.intercept('GET', '**/api/v1/customers*page=2*', {
        body: {
          customers: [
            { id: 'cust-004', firstName: 'Alice', lastName: 'Wonder' },
          ],
          pagination: { page: 2, total: 4, totalPages: 2 },
        },
      }).as('page2');

      customersPage.goToNextPage();

      cy.wait('@page2');
    });
  });

  describe('Create Customer', () => {
    beforeEach(() => {
      customersPage.visit();
      cy.wait('@getCustomers');
    });

    it('should create a new customer successfully', () => {
      const newCustomer = {
        firstName: 'New',
        lastName: 'Customer',
        email: 'new.customer@example.com',
        phone: '(555) 111-2222',
        companyName: 'New Company',
        address: {
          street: '100 New St',
          city: 'New City',
          state: 'TX',
          zip: '78702',
        },
      };

      cy.intercept('POST', '**/api/v1/customers', {
        statusCode: 201,
        body: {
          id: 'cust-new',
          ...newCustomer,
        },
      }).as('createCustomer');

      customersPage.createCustomer(newCustomer);

      cy.wait('@createCustomer').its('request.body').should('deep.include', {
        firstName: 'New',
        lastName: 'Customer',
        email: 'new.customer@example.com',
      });

      // Should show success message
      customersPage.assertToast('Customer created', 'success');
    });

    it('should validate required fields', () => {
      customersPage.clickAddCustomer();
      customersPage.submitForm();

      // Check for validation errors
      cy.assertFieldError('firstName', 'required');
      cy.assertFieldError('lastName', 'required');
      cy.assertFieldError('email', 'required');
    });

    it('should validate email format', () => {
      customersPage.clickAddCustomer();

      cy.fillForm({
        firstName: 'Test',
        lastName: 'User',
        email: 'invalid-email',
      });

      customersPage.submitForm();

      cy.assertFieldError('email', 'valid email');
    });

    it('should handle duplicate email error', () => {
      cy.intercept('POST', '**/api/v1/customers', {
        statusCode: 409,
        body: {
          error: true,
          message: 'A customer with this email already exists',
        },
      }).as('createCustomer');

      customersPage.createCustomer({
        firstName: 'Test',
        lastName: 'User',
        email: 'existing@example.com',
      });

      cy.wait('@createCustomer');

      customersPage.assertToast('already exists', 'error');
    });

    it('should cancel customer creation', () => {
      customersPage.clickAddCustomer();

      cy.fillForm({
        firstName: 'Test',
        lastName: 'User',
      });

      customersPage.cancel();

      // Should return to list
      customersPage.assertUrl('/customers');
    });
  });

  describe('View Customer', () => {
    it('should display customer details', () => {
      cy.intercept('GET', '**/api/v1/customers/cust-001', {
        body: {
          id: 'cust-001',
          firstName: 'John',
          lastName: 'Smith',
          email: 'john.smith@example.com',
          phone: '(555) 123-4567',
          companyName: 'Smith Enterprises',
          totalJobs: 5,
          totalInvoiced: 2500,
        },
      }).as('getCustomer');

      customersPage.visitCustomer('cust-001');
      cy.wait('@getCustomer');

      customersPage.assertCustomerDetailVisible();
      customersPage.assertCustomerName('John Smith');
      customersPage.assertCustomerEmail('john.smith@example.com');
    });

    it('should display customer jobs', () => {
      cy.intercept('GET', '**/api/v1/customers/cust-001', {
        fixture: 'customers.json',
      });
      cy.intercept('GET', '**/api/v1/customers/cust-001/jobs', {
        body: {
          jobs: [
            { id: 'job-001', title: 'Lawn Mowing', status: 'completed' },
            { id: 'job-002', title: 'Pool Cleaning', status: 'scheduled' },
          ],
        },
      }).as('getCustomerJobs');

      customersPage.visitCustomer('cust-001');
      cy.wait('@getCustomerJobs');

      customersPage.getCustomerJobs().should('have.length', 2);
    });

    it('should display customer invoices', () => {
      cy.intercept('GET', '**/api/v1/customers/cust-001', {
        fixture: 'customers.json',
      });
      cy.intercept('GET', '**/api/v1/customers/cust-001/invoices', {
        body: {
          invoices: [
            {
              id: 'inv-001',
              invoiceNumber: 'INV-001',
              total: 150,
              status: 'paid',
            },
          ],
        },
      }).as('getCustomerInvoices');

      customersPage.visitCustomer('cust-001');
      cy.wait('@getCustomerInvoices');

      customersPage.getCustomerInvoices().should('have.length', 1);
    });
  });

  describe('Edit Customer', () => {
    beforeEach(() => {
      cy.intercept('GET', '**/api/v1/customers/cust-001', {
        body: {
          id: 'cust-001',
          firstName: 'John',
          lastName: 'Smith',
          email: 'john.smith@example.com',
          phone: '(555) 123-4567',
        },
      }).as('getCustomer');

      customersPage.visitCustomer('cust-001');
      cy.wait('@getCustomer');
    });

    it('should edit customer successfully', () => {
      cy.intercept('PUT', '**/api/v1/customers/cust-001', {
        statusCode: 200,
        body: {
          id: 'cust-001',
          firstName: 'John',
          lastName: 'Smith-Updated',
          email: 'john.smith@example.com',
        },
      }).as('updateCustomer');

      customersPage.clickEdit();

      cy.get('input[name="lastName"]').clear().type('Smith-Updated');

      customersPage.save();

      cy.wait('@updateCustomer').its('request.body').should('include', {
        lastName: 'Smith-Updated',
      });

      customersPage.assertToast('Customer updated', 'success');
    });

    it('should handle update errors', () => {
      cy.intercept('PUT', '**/api/v1/customers/cust-001', {
        statusCode: 500,
        body: {
          error: true,
          message: 'Failed to update customer',
        },
      }).as('updateCustomer');

      customersPage.clickEdit();
      cy.get('input[name="lastName"]').clear().type('Updated');
      customersPage.save();

      cy.wait('@updateCustomer');

      customersPage.assertToast('Failed to update', 'error');
    });
  });

  describe('Delete Customer', () => {
    beforeEach(() => {
      cy.intercept('GET', '**/api/v1/customers/cust-001', {
        body: {
          id: 'cust-001',
          firstName: 'John',
          lastName: 'Smith',
          email: 'john.smith@example.com',
        },
      }).as('getCustomer');

      customersPage.visitCustomer('cust-001');
      cy.wait('@getCustomer');
    });

    it('should delete customer after confirmation', () => {
      cy.intercept('DELETE', '**/api/v1/customers/cust-001', {
        statusCode: 200,
        body: { success: true },
      }).as('deleteCustomer');

      customersPage.clickDelete();
      customersPage.confirmDelete();

      cy.wait('@deleteCustomer');

      customersPage.assertToast('Customer deleted', 'success');
      customersPage.assertUrl('/customers');
    });

    it('should cancel delete when not confirmed', () => {
      customersPage.clickDelete();
      customersPage.closeModal();

      // Should still be on customer detail page
      customersPage.assertCustomerDetailVisible();
    });

    it('should handle delete error for customer with jobs', () => {
      cy.intercept('DELETE', '**/api/v1/customers/cust-001', {
        statusCode: 400,
        body: {
          error: true,
          message: 'Cannot delete customer with active jobs',
        },
      }).as('deleteCustomer');

      customersPage.clickDelete();
      customersPage.confirmDelete();

      cy.wait('@deleteCustomer');

      customersPage.assertToast('Cannot delete', 'error');
    });
  });

  describe('Customer Actions', () => {
    beforeEach(() => {
      cy.intercept('GET', '**/api/v1/customers/cust-001', {
        body: {
          id: 'cust-001',
          firstName: 'John',
          lastName: 'Smith',
          email: 'john.smith@example.com',
        },
      }).as('getCustomer');

      customersPage.visitCustomer('cust-001');
      cy.wait('@getCustomer');
    });

    it('should create job for customer', () => {
      customersPage.clickCreateJob();

      cy.url().should('include', '/jobs/new');
      cy.url().should('include', 'customerId=cust-001');
    });

    it('should create quote for customer', () => {
      customersPage.clickCreateQuote();

      cy.url().should('include', '/quotes/new');
      cy.url().should('include', 'customerId=cust-001');
    });

    it('should create invoice for customer', () => {
      customersPage.clickCreateInvoice();

      cy.url().should('include', '/invoices/new');
      cy.url().should('include', 'customerId=cust-001');
    });
  });

  describe('Import/Export', () => {
    beforeEach(() => {
      customersPage.visit();
      cy.wait('@getCustomers');
    });

    it('should export customers to CSV', () => {
      cy.intercept('GET', '**/api/v1/customers/export*', {
        statusCode: 200,
        headers: {
          'Content-Type': 'text/csv',
          'Content-Disposition': 'attachment; filename="customers.csv"',
        },
        body: 'firstName,lastName,email\nJohn,Smith,john@example.com',
      }).as('exportCustomers');

      customersPage.clickExport();

      cy.wait('@exportCustomers');
    });

    it('should import customers from CSV', () => {
      cy.intercept('POST', '**/api/v1/customers/import', {
        statusCode: 200,
        body: {
          imported: 5,
          failed: 1,
          errors: ['Row 3: Invalid email format'],
        },
      }).as('importCustomers');

      customersPage.clickImport();

      // Upload file
      cy.get('input[type="file"]').selectFile(
        'cypress/fixtures/import-customers.csv'
      );
      cy.get('[data-testid="import-submit"]').click();

      cy.wait('@importCustomers');

      customersPage.assertToast('imported 5', 'success');
    });
  });
});
