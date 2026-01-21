// =============================================================================
// Jobs Page Object
// =============================================================================

import { BasePage } from './BasePage';

export interface JobData {
  title: string;
  description?: string;
  customerId?: string;
  customerName?: string;
  scheduledDate?: string;
  scheduledTime?: string;
  duration?: number;
  price?: number;
  assignedTo?: string;
  status?: string;
  notes?: string;
  address?: {
    street?: string;
    city?: string;
    state?: string;
    zip?: string;
  };
}

export class JobsPage extends BasePage {
  private pageSelectors = {
    // List view
    jobList: '[data-testid="job-list"], .job-list',
    jobCard: '[data-testid="job-card"], .job-card',
    jobRow: '[data-testid="job-row"], tbody tr',

    // Actions
    addJobButton:
      '[data-testid="add-job"], button:contains("Add Job"), button:contains("New Job")',
    viewCalendarButton:
      '[data-testid="view-calendar"], button:contains("Calendar")',

    // Filters
    statusFilter: '[data-testid="status-filter"]',
    dateFilter: '[data-testid="date-filter"]',
    assigneeFilter: '[data-testid="assignee-filter"]',
    customerFilter: '[data-testid="customer-filter"]',

    // Form fields
    titleInput: '[name="title"], [data-testid="title-input"]',
    descriptionInput: '[name="description"], [data-testid="description-input"]',
    customerSelect: '[name="customerId"], [data-testid="customer-select"]',
    customerSearch: '[data-testid="customer-search"]',
    scheduledDateInput:
      '[name="scheduledDate"], [data-testid="scheduled-date"]',
    scheduledTimeInput:
      '[name="scheduledTime"], [data-testid="scheduled-time"]',
    durationInput: '[name="duration"], [data-testid="duration-input"]',
    priceInput: '[name="price"], [data-testid="price-input"]',
    assignedToSelect: '[name="assignedTo"], [data-testid="assigned-to"]',
    statusSelect: '[name="status"], [data-testid="status-select"]',
    notesInput: '[name="notes"], [data-testid="notes-input"]',
    streetInput:
      '[name="street"], [name="address.street"], [data-testid="street-input"]',
    cityInput:
      '[name="city"], [name="address.city"], [data-testid="city-input"]',
    stateInput:
      '[name="state"], [name="address.state"], [data-testid="state-input"]',
    zipInput: '[name="zip"], [name="address.zip"], [data-testid="zip-input"]',

    // Detail view
    jobDetail: '[data-testid="job-detail"]',
    jobTitle: '[data-testid="job-title"], .job-title',
    jobStatus: '[data-testid="job-status"], .job-status',
    jobCustomer: '[data-testid="job-customer"], .job-customer',
    jobSchedule: '[data-testid="job-schedule"], .job-schedule',

    // Actions in detail view
    editButton: '[data-testid="edit-job"], button:contains("Edit")',
    deleteButton: '[data-testid="delete-job"], button:contains("Delete")',
    completeButton: '[data-testid="complete-job"], button:contains("Complete")',
    cancelButton: '[data-testid="cancel-job"], button:contains("Cancel Job")',
    rescheduleButton:
      '[data-testid="reschedule-job"], button:contains("Reschedule")',
    createInvoiceButton:
      '[data-testid="create-invoice"], button:contains("Create Invoice")',

    // Calendar view
    calendarView: '[data-testid="calendar-view"], .calendar',
    calendarDay: '[data-testid="calendar-day"], .calendar-day',
    calendarEvent: '[data-testid="calendar-event"], .calendar-event',
  };

  // Visit jobs list
  visit(): void {
    super.visit('/jobs');
  }

  // Visit job detail page
  visitJob(jobId: string): void {
    super.visit(`/jobs/${jobId}`);
  }

  // Visit calendar view
  visitCalendar(): void {
    super.visit('/schedule');
  }

  // Click add job button
  clickAddJob(): void {
    cy.get(this.pageSelectors.addJobButton).click();
  }

  // Fill job form
  fillJobForm(data: JobData): void {
    cy.get(this.pageSelectors.titleInput).clear().type(data.title);

    if (data.description) {
      cy.get(this.pageSelectors.descriptionInput)
        .clear()
        .type(data.description);
    }

    if (data.customerName) {
      cy.get(
        this.pageSelectors.customerSearch || this.pageSelectors.customerSelect
      )
        .click()
        .type(data.customerName);
      cy.get('[role="listbox"], .autocomplete-results')
        .contains(data.customerName)
        .click();
    }

    if (data.scheduledDate) {
      cy.get(this.pageSelectors.scheduledDateInput)
        .clear()
        .type(data.scheduledDate);
    }

    if (data.scheduledTime) {
      cy.get(this.pageSelectors.scheduledTimeInput)
        .clear()
        .type(data.scheduledTime);
    }

    if (data.duration) {
      cy.get(this.pageSelectors.durationInput)
        .clear()
        .type(data.duration.toString());
    }

    if (data.price) {
      cy.get(this.pageSelectors.priceInput).clear().type(data.price.toFixed(2));
    }

    if (data.assignedTo) {
      cy.get(this.pageSelectors.assignedToSelect).click();
      cy.get('[role="listbox"], .dropdown-menu')
        .contains(data.assignedTo)
        .click();
    }

    if (data.notes) {
      cy.get(this.pageSelectors.notesInput).clear().type(data.notes);
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
  }

  // Create job (full flow)
  createJob(data: JobData): void {
    this.clickAddJob();
    this.fillJobForm(data);
    this.submitForm();
  }

  // Get jobs
  getJobs(): Cypress.Chainable<JQuery<HTMLElement>> {
    return cy.get(
      `${this.pageSelectors.jobCard}, ${this.pageSelectors.jobRow}`
    );
  }

  // Assert job count
  assertJobCount(count: number): void {
    this.getJobs().should('have.length', count);
  }

  // Click job to view details
  clickJob(title: string): void {
    cy.get(`${this.pageSelectors.jobCard}, ${this.pageSelectors.jobRow}`)
      .contains(title)
      .click();
  }

  // Filter by status
  filterByStatus(status: string): void {
    cy.get(this.pageSelectors.statusFilter).click();
    cy.get('[role="listbox"], .dropdown-menu').contains(status).click();
  }

  // Filter by date
  filterByDate(date: string): void {
    cy.get(this.pageSelectors.dateFilter).click().type(date);
  }

  // Filter by assignee
  filterByAssignee(assignee: string): void {
    cy.get(this.pageSelectors.assigneeFilter).click();
    cy.get('[role="listbox"], .dropdown-menu').contains(assignee).click();
  }

  // Click edit button (detail view)
  clickEdit(): void {
    cy.get(this.pageSelectors.editButton).click();
  }

  // Click delete button
  clickDelete(): void {
    cy.get(this.pageSelectors.deleteButton).click();
  }

  // Mark job as complete
  clickComplete(): void {
    cy.get(this.pageSelectors.completeButton).click();
  }

  // Cancel job
  clickCancelJob(): void {
    cy.get(this.pageSelectors.cancelButton).click();
  }

  // Reschedule job
  clickReschedule(): void {
    cy.get(this.pageSelectors.rescheduleButton).click();
  }

  // Create invoice from job
  clickCreateInvoice(): void {
    cy.get(this.pageSelectors.createInvoiceButton).click();
  }

  // Switch to calendar view
  switchToCalendar(): void {
    cy.get(this.pageSelectors.viewCalendarButton).click();
  }

  // Assert job status
  assertJobStatus(status: string): void {
    cy.get(this.pageSelectors.jobStatus).should('contain.text', status);
  }

  // Assert job title in detail view
  assertJobTitle(title: string): void {
    cy.get(this.pageSelectors.jobTitle).should('contain.text', title);
  }

  // Get calendar events
  getCalendarEvents(): Cypress.Chainable<JQuery<HTMLElement>> {
    return cy.get(this.pageSelectors.calendarEvent);
  }

  // Click calendar day
  clickCalendarDay(day: number): void {
    cy.get(this.pageSelectors.calendarDay).contains(day.toString()).click();
  }

  // Assert job in calendar
  assertJobInCalendar(jobTitle: string): void {
    cy.get(this.pageSelectors.calendarEvent).should('contain.text', jobTitle);
  }

  // Assert job in list
  assertJobInList(title: string): void {
    cy.get(`${this.pageSelectors.jobList}, ${this.selectors.table}`).should(
      'contain.text',
      title
    );
  }
}

export default JobsPage;
