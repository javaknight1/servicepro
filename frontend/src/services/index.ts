// Core API instance with auth interceptors
export { default as api, authApi, userApi } from './api';

// Domain-specific services (all use the shared api instance)
export { customerService } from './customerService';
export type {
  Customer,
  CustomerFilters,
  CustomerListResponse,
  CustomerStats,
} from './customerService';

export { jobService } from './jobService';
export type { Job, JobFilters, JobListResponse, JobStats } from './jobService';
export { JobStatus, JobPriority } from './jobService';

export { invoiceService } from './invoiceService';
export type {
  Invoice,
  InvoiceFilters,
  InvoiceListResponse,
  InvoiceStats,
  InvoiceLineItem,
  InvoicePayment,
  InvoiceRefund,
  PaymentRequest,
  RefundRequest,
} from './invoiceService';
export { InvoiceStatus } from './invoiceService';

export { quoteService } from './quoteService';
export type { QuoteFilters } from './quoteService';

export { revenueService } from './revenueService';

export { customerReportService } from './customerReportService';

export { roleApi } from './roleApi';

export { membershipApi } from './membershipService';

export { billingApi } from './billingService';
