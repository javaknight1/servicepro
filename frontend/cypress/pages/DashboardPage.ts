// =============================================================================
// Dashboard Page Object
// =============================================================================

import { BasePage } from './BasePage';

export class DashboardPage extends BasePage {
  private pageSelectors = {
    // Stats widgets
    statsContainer: '[data-testid="stats-container"], .dashboard-stats',
    revenueWidget: '[data-testid="revenue-widget"], .revenue-stat',
    jobsWidget: '[data-testid="jobs-widget"], .jobs-stat',
    customersWidget: '[data-testid="customers-widget"], .customers-stat',
    invoicesWidget: '[data-testid="invoices-widget"], .invoices-stat',

    // Recent items
    recentJobs: '[data-testid="recent-jobs"], .recent-jobs',
    recentInvoices: '[data-testid="recent-invoices"], .recent-invoices',
    recentCustomers: '[data-testid="recent-customers"], .recent-customers',

    // Charts
    revenueChart: '[data-testid="revenue-chart"], .revenue-chart',
    jobsChart: '[data-testid="jobs-chart"], .jobs-chart',

    // Quick actions
    quickActions: '[data-testid="quick-actions"], .quick-actions',
    newJobButton: '[data-testid="new-job-button"], button:contains("New Job")',
    newCustomerButton:
      '[data-testid="new-customer-button"], button:contains("New Customer")',
    newQuoteButton:
      '[data-testid="new-quote-button"], button:contains("New Quote")',
    newInvoiceButton:
      '[data-testid="new-invoice-button"], button:contains("New Invoice")',

    // Calendar preview
    calendarPreview: '[data-testid="calendar-preview"], .calendar-widget',
    todaySchedule: '[data-testid="today-schedule"], .today-schedule',

    // Notifications
    notificationBell: '[data-testid="notification-bell"], .notification-icon',
    notificationDropdown:
      '[data-testid="notification-dropdown"], .notifications-list',
    notificationCount:
      '[data-testid="notification-count"], .notification-badge',

    // Date range selector
    dateRangeSelector: '[data-testid="date-range"], .date-range-picker',
  };

  // Visit dashboard
  visit(): void {
    super.visit('/dashboard');
  }

  // Get revenue widget value
  getRevenueValue(): Cypress.Chainable<JQuery<HTMLElement>> {
    return cy
      .get(this.pageSelectors.revenueWidget)
      .find('.value, .amount, [data-testid="value"]');
  }

  // Assert revenue value
  assertRevenueValue(expectedValue: string): void {
    this.getRevenueValue().should('contain.text', expectedValue);
  }

  // Get jobs count
  getJobsCount(): Cypress.Chainable<JQuery<HTMLElement>> {
    return cy
      .get(this.pageSelectors.jobsWidget)
      .find('.value, .count, [data-testid="value"]');
  }

  // Get customers count
  getCustomersCount(): Cypress.Chainable<JQuery<HTMLElement>> {
    return cy
      .get(this.pageSelectors.customersWidget)
      .find('.value, .count, [data-testid="value"]');
  }

  // Get invoices count
  getInvoicesCount(): Cypress.Chainable<JQuery<HTMLElement>> {
    return cy
      .get(this.pageSelectors.invoicesWidget)
      .find('.value, .count, [data-testid="value"]');
  }

  // Click new job quick action
  clickNewJob(): void {
    cy.get(this.pageSelectors.newJobButton).click();
  }

  // Click new customer quick action
  clickNewCustomer(): void {
    cy.get(this.pageSelectors.newCustomerButton).click();
  }

  // Click new quote quick action
  clickNewQuote(): void {
    cy.get(this.pageSelectors.newQuoteButton).click();
  }

  // Click new invoice quick action
  clickNewInvoice(): void {
    cy.get(this.pageSelectors.newInvoiceButton).click();
  }

  // Get recent jobs list
  getRecentJobs(): Cypress.Chainable<JQuery<HTMLElement>> {
    return cy.get(this.pageSelectors.recentJobs).find('li, .job-item, tr');
  }

  // Assert recent jobs count
  assertRecentJobsCount(count: number): void {
    this.getRecentJobs().should('have.length', count);
  }

  // Click recent job
  clickRecentJob(index: number): void {
    this.getRecentJobs().eq(index).click();
  }

  // Get recent invoices list
  getRecentInvoices(): Cypress.Chainable<JQuery<HTMLElement>> {
    return cy
      .get(this.pageSelectors.recentInvoices)
      .find('li, .invoice-item, tr');
  }

  // Open notifications
  openNotifications(): void {
    cy.get(this.pageSelectors.notificationBell).click();
  }

  // Get notification count
  getNotificationCount(): Cypress.Chainable<JQuery<HTMLElement>> {
    return cy.get(this.pageSelectors.notificationCount);
  }

  // Assert notification count
  assertNotificationCount(count: number): void {
    this.getNotificationCount().should('contain.text', count.toString());
  }

  // Get today's schedule items
  getTodaySchedule(): Cypress.Chainable<JQuery<HTMLElement>> {
    return cy.get(this.pageSelectors.todaySchedule).find('.schedule-item, li');
  }

  // Select date range
  selectDateRange(
    range: 'today' | 'week' | 'month' | 'quarter' | 'year'
  ): void {
    cy.get(this.pageSelectors.dateRangeSelector).click();
    cy.get('[role="listbox"], .dropdown-menu')
      .contains(new RegExp(range, 'i'))
      .click();
  }

  // Assert chart is visible
  assertRevenueChartVisible(): void {
    cy.get(this.pageSelectors.revenueChart).should('be.visible');
  }

  // Assert jobs chart is visible
  assertJobsChartVisible(): void {
    cy.get(this.pageSelectors.jobsChart).should('be.visible');
  }

  // Assert dashboard loaded
  assertDashboardLoaded(): void {
    cy.get(this.pageSelectors.statsContainer).should('be.visible');
  }
}

export default DashboardPage;
