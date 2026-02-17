// Type-safe analytics event constants
export const AnalyticsEvents = {
  // Quote workflow
  QUOTE_CREATED: 'quote_created',
  QUOTE_SENT: 'quote_sent',
  QUOTE_ACCEPTED: 'quote_accepted',
  QUOTE_REJECTED: 'quote_rejected',

  // Job workflow
  JOB_CREATED: 'job_created',
  JOB_STATUS_CHANGED: 'job_status_changed',
  JOB_COMPLETED: 'job_completed',

  // Invoice workflow
  INVOICE_CREATED: 'invoice_created',
  INVOICE_SENT: 'invoice_sent',
  INVOICE_PAID: 'invoice_paid',

  // General
  PAGE_VIEWED: 'page_viewed',
  FEATURE_USED: 'feature_used',
  REPORT_GENERATED: 'report_generated',
  EXPORT_DOWNLOADED: 'export_downloaded',
  REMINDER_SENT: 'reminder_sent',
} as const;

export type AnalyticsEvent =
  (typeof AnalyticsEvents)[keyof typeof AnalyticsEvents];
