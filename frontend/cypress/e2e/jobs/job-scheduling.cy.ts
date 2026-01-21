// =============================================================================
// Job Scheduling E2E Tests
// =============================================================================

import { JobsPage } from '../../pages';

describe('Job Scheduling', () => {
  const jobsPage = new JobsPage();

  beforeEach(() => {
    cy.login();

    cy.intercept('GET', '**/api/v1/jobs*', { fixture: 'jobs.json' }).as(
      'getJobs'
    );
    cy.intercept('GET', '**/api/v1/users/me', { fixture: 'user.json' }).as(
      'getProfile'
    );
    cy.intercept('GET', '**/api/v1/customers*', {
      fixture: 'customers.json',
    }).as('getCustomers');
  });

  describe('Job List', () => {
    beforeEach(() => {
      jobsPage.visit();
      cy.wait('@getJobs');
    });

    it('should display list of jobs', () => {
      jobsPage.getJobs().should('have.length', 4);
    });

    it('should display job information', () => {
      jobsPage.assertJobInList('Lawn Mowing Service');
      jobsPage.assertJobInList('Pool Cleaning');
      jobsPage.assertJobInList('HVAC Maintenance');
    });

    it('should filter jobs by status', () => {
      cy.intercept('GET', '**/api/v1/jobs*status=scheduled*', {
        body: {
          jobs: [
            {
              id: 'job-001',
              title: 'Lawn Mowing Service',
              status: 'scheduled',
            },
          ],
          pagination: { total: 1 },
        },
      }).as('filterJobs');

      jobsPage.filterByStatus('Scheduled');

      cy.wait('@filterJobs');
      jobsPage.assertJobCount(1);
    });

    it('should filter jobs by assignee', () => {
      cy.intercept('GET', '**/api/v1/jobs*assignedTo=user-456*', {
        body: {
          jobs: [
            {
              id: 'job-001',
              title: 'Lawn Mowing',
              assignedTo: { id: 'user-456', name: 'Mike Tech' },
            },
            {
              id: 'job-003',
              title: 'HVAC Maintenance',
              assignedTo: { id: 'user-456', name: 'Mike Tech' },
            },
          ],
          pagination: { total: 2 },
        },
      }).as('filterByAssignee');

      jobsPage.filterByAssignee('Mike Tech');

      cy.wait('@filterByAssignee');
      jobsPage.assertJobCount(2);
    });

    it('should search jobs', () => {
      cy.intercept('GET', '**/api/v1/jobs*q=Pool*', {
        body: {
          jobs: [
            { id: 'job-002', title: 'Pool Cleaning', status: 'completed' },
          ],
          pagination: { total: 1 },
        },
      }).as('searchJobs');

      jobsPage.search('Pool');

      cy.wait('@searchJobs');
      jobsPage.assertJobInList('Pool Cleaning');
    });
  });

  describe('Create Job', () => {
    beforeEach(() => {
      jobsPage.visit();
      cy.wait('@getJobs');
    });

    it('should create a new job successfully', () => {
      const newJob = {
        title: 'New Service Job',
        description: 'Service description',
        customerName: 'John Smith',
        scheduledDate: '2024-02-15',
        scheduledTime: '10:00',
        duration: 60,
        price: 100,
        assignedTo: 'Mike Tech',
      };

      cy.intercept('POST', '**/api/v1/jobs', {
        statusCode: 201,
        body: { id: 'job-new', ...newJob, status: 'scheduled' },
      }).as('createJob');

      jobsPage.createJob(newJob);

      cy.wait('@createJob').its('request.body').should('include', {
        title: 'New Service Job',
        description: 'Service description',
      });

      jobsPage.assertToast('Job created', 'success');
    });

    it('should validate required fields', () => {
      jobsPage.clickAddJob();
      jobsPage.submitForm();

      cy.assertFieldError('title', 'required');
    });

    it('should require customer selection', () => {
      jobsPage.clickAddJob();

      cy.fillForm({
        title: 'Test Job',
        scheduledDate: '2024-02-15',
      });

      jobsPage.submitForm();

      cy.assertFieldError('customerId', 'required');
    });

    it('should pre-fill customer when coming from customer page', () => {
      cy.visit('/jobs/new?customerId=cust-001');

      cy.get('[data-testid="customer-select"], [name="customerId"]').should(
        'contain.text',
        'John Smith'
      );
    });
  });

  describe('View Job', () => {
    it('should display job details', () => {
      cy.intercept('GET', '**/api/v1/jobs/job-001', {
        body: {
          id: 'job-001',
          title: 'Lawn Mowing Service',
          description: 'Regular weekly lawn mowing',
          status: 'scheduled',
          scheduledDate: '2024-02-01',
          scheduledTime: '09:00',
          duration: 60,
          price: 75,
          customerName: 'John Smith',
          assignedTo: { id: 'user-456', name: 'Mike Tech' },
        },
      }).as('getJob');

      jobsPage.visitJob('job-001');
      cy.wait('@getJob');

      jobsPage.assertJobTitle('Lawn Mowing Service');
      jobsPage.assertJobStatus('scheduled');
    });
  });

  describe('Job Status Workflow', () => {
    beforeEach(() => {
      cy.intercept('GET', '**/api/v1/jobs/job-001', {
        body: {
          id: 'job-001',
          title: 'Lawn Mowing',
          status: 'scheduled',
        },
      }).as('getJob');

      jobsPage.visitJob('job-001');
      cy.wait('@getJob');
    });

    it('should mark job as complete', () => {
      cy.intercept('PATCH', '**/api/v1/jobs/job-001/complete', {
        statusCode: 200,
        body: {
          id: 'job-001',
          status: 'completed',
          completedAt: new Date().toISOString(),
        },
      }).as('completeJob');

      jobsPage.clickComplete();

      cy.wait('@completeJob');

      jobsPage.assertToast('marked as complete', 'success');
      jobsPage.assertJobStatus('completed');
    });

    it('should cancel job', () => {
      cy.intercept('PATCH', '**/api/v1/jobs/job-001/cancel', {
        statusCode: 200,
        body: { id: 'job-001', status: 'cancelled' },
      }).as('cancelJob');

      jobsPage.clickCancelJob();
      jobsPage.confirmDelete(); // Confirm cancellation

      cy.wait('@cancelJob');

      jobsPage.assertToast('cancelled', 'success');
    });

    it('should reschedule job', () => {
      cy.intercept('PATCH', '**/api/v1/jobs/job-001', {
        statusCode: 200,
        body: {
          id: 'job-001',
          status: 'scheduled',
          scheduledDate: '2024-02-20',
          scheduledTime: '14:00',
        },
      }).as('rescheduleJob');

      jobsPage.clickReschedule();

      cy.get('[name="scheduledDate"]').clear().type('2024-02-20');
      cy.get('[name="scheduledTime"]').clear().type('14:00');

      jobsPage.save();

      cy.wait('@rescheduleJob');

      jobsPage.assertToast('rescheduled', 'success');
    });
  });

  describe('Job to Invoice', () => {
    it('should create invoice from completed job', () => {
      cy.intercept('GET', '**/api/v1/jobs/job-002', {
        body: {
          id: 'job-002',
          title: 'Pool Cleaning',
          status: 'completed',
          price: 150,
          customerId: 'cust-001',
        },
      }).as('getJob');

      cy.intercept('POST', '**/api/v1/invoices', {
        statusCode: 201,
        body: {
          id: 'inv-new',
          invoiceNumber: 'INV-2024-0005',
          total: 150,
        },
      }).as('createInvoice');

      jobsPage.visitJob('job-002');
      cy.wait('@getJob');

      jobsPage.clickCreateInvoice();

      cy.wait('@createInvoice');

      cy.url().should('include', '/invoices/inv-new');
    });
  });

  describe('Calendar View', () => {
    beforeEach(() => {
      cy.intercept('GET', '**/api/v1/schedule*', {
        body: {
          jobs: [
            {
              id: 'job-001',
              title: 'Lawn Mowing',
              scheduledDate: '2024-02-01',
              scheduledTime: '09:00',
              duration: 60,
            },
            {
              id: 'job-003',
              title: 'HVAC Maintenance',
              scheduledDate: '2024-02-01',
              scheduledTime: '14:00',
              duration: 120,
            },
          ],
        },
      }).as('getSchedule');
    });

    it('should display jobs on calendar', () => {
      jobsPage.visitCalendar();
      cy.wait('@getSchedule');

      jobsPage.assertJobInCalendar('Lawn Mowing');
      jobsPage.assertJobInCalendar('HVAC Maintenance');
    });

    it('should switch between list and calendar view', () => {
      jobsPage.visit();
      cy.wait('@getJobs');

      jobsPage.switchToCalendar();

      cy.get('[data-testid="calendar-view"], .calendar').should('be.visible');
    });

    it('should navigate calendar months', () => {
      cy.intercept('GET', '**/api/v1/schedule*month=2024-03*', {
        body: { jobs: [] },
      }).as('getMarchSchedule');

      jobsPage.visitCalendar();
      cy.wait('@getSchedule');

      cy.get('[data-testid="next-month"], button:contains("Next")').click();

      cy.wait('@getMarchSchedule');
    });

    it('should create job from calendar day click', () => {
      jobsPage.visitCalendar();
      cy.wait('@getSchedule');

      cy.get('[data-testid="calendar-day"]:contains("15")').click();

      cy.url().should('include', '/jobs/new');
      cy.get('[name="scheduledDate"]').should('have.value', /2024-02-15/);
    });

    it('should show job details on calendar event click', () => {
      cy.intercept('GET', '**/api/v1/jobs/job-001', {
        body: { id: 'job-001', title: 'Lawn Mowing' },
      }).as('getJob');

      jobsPage.visitCalendar();
      cy.wait('@getSchedule');

      cy.contains('.calendar-event', 'Lawn Mowing').click();

      cy.url().should('include', '/jobs/job-001');
    });
  });

  describe('Recurring Jobs', () => {
    it('should create recurring job', () => {
      cy.intercept('POST', '**/api/v1/jobs', {
        statusCode: 201,
        body: {
          id: 'job-recurring',
          title: 'Weekly Lawn Service',
          isRecurring: true,
          recurrencePattern: 'weekly',
        },
      }).as('createRecurringJob');

      jobsPage.visit();
      jobsPage.clickAddJob();

      cy.fillForm({
        title: 'Weekly Lawn Service',
        customerName: 'John Smith',
        scheduledDate: '2024-02-01',
      });

      // Enable recurring
      cy.get('[data-testid="recurring-toggle"], [name="isRecurring"]').click();
      cy.get('[data-testid="recurrence-pattern"]').select('weekly');
      cy.get('[data-testid="recurrence-end-date"]').type('2024-06-01');

      jobsPage.submitForm();

      cy.wait('@createRecurringJob').its('request.body').should('include', {
        isRecurring: true,
        recurrencePattern: 'weekly',
      });
    });
  });

  describe('Job Assignment', () => {
    it('should assign technician to job', () => {
      cy.intercept('GET', '**/api/v1/jobs/job-004', {
        body: {
          id: 'job-004',
          title: 'Plumbing Repair',
          status: 'pending',
          assignedTo: null,
        },
      }).as('getJob');

      cy.intercept('GET', '**/api/v1/users*role=technician*', {
        body: {
          users: [
            { id: 'user-456', name: 'Mike Tech' },
            { id: 'user-789', name: 'Sarah Pool' },
          ],
        },
      }).as('getTechnicians');

      cy.intercept('PATCH', '**/api/v1/jobs/job-004', {
        statusCode: 200,
        body: {
          id: 'job-004',
          assignedTo: { id: 'user-456', name: 'Mike Tech' },
        },
      }).as('assignJob');

      jobsPage.visitJob('job-004');
      cy.wait('@getJob');

      cy.get('[data-testid="assign-button"]').click();
      cy.wait('@getTechnicians');

      cy.get('[data-testid="technician-select"]').select('Mike Tech');
      cy.get('[data-testid="confirm-assign"]').click();

      cy.wait('@assignJob');

      jobsPage.assertToast('assigned', 'success');
    });
  });

  describe('Conflict Detection', () => {
    it('should warn about scheduling conflicts', () => {
      cy.intercept('POST', '**/api/v1/jobs/check-conflicts', {
        body: {
          hasConflicts: true,
          conflicts: [
            {
              jobId: 'job-001',
              title: 'Existing Job',
              scheduledDate: '2024-02-15',
              scheduledTime: '10:00',
            },
          ],
        },
      }).as('checkConflicts');

      jobsPage.visit();
      jobsPage.clickAddJob();

      cy.fillForm({
        title: 'New Job',
        customerName: 'John Smith',
        scheduledDate: '2024-02-15',
        scheduledTime: '10:00',
        assignedTo: 'Mike Tech',
      });

      cy.wait('@checkConflicts');

      cy.get('[data-testid="conflict-warning"]')
        .should('be.visible')
        .and('contain.text', 'scheduling conflict');
    });
  });
});
