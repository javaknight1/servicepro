/**
 * A/R Aging Report Types
 */

export type AgingBucket = 'current' | '1-30' | '31-60' | '61-90' | '90+';

export interface ARAgingQuery {
  as_of_date?: string;
  customer_id?: string;
  min_amount?: number;
  include_paid?: boolean;
  include_draft?: boolean;
}

export interface ARAgingBucketSummary {
  bucket: AgingBucket;
  display_name: string;
  invoice_count: number;
  total_amount: string; // decimal from backend
  percentage: string; // decimal from backend
  color: string;
}

export interface ARAgingInvoice {
  invoice_id: string;
  invoice_number: string;
  customer_id: string;
  customer_name: string;
  company_name?: string;
  issue_date: string;
  due_date: string;
  total_amount: string; // decimal from backend
  amount_paid: string; // decimal from backend
  amount_due: string; // decimal from backend
  days_overdue: number;
  bucket: AgingBucket;
  status: string;
}

export interface ARAgingCustomerSummary {
  customer_id: string;
  customer_name: string;
  company_name?: string;
  email: string;
  invoice_count: number;
  total_owed: string; // decimal from backend
  current: string; // decimal from backend
  days_1_to_30: string; // decimal from backend
  days_31_to_60: string; // decimal from backend
  days_61_to_90: string; // decimal from backend
  days_90_plus: string; // decimal from backend
  oldest_invoice?: string;
  last_payment?: string;
}

export interface ARAgingReportSummary {
  total_outstanding: string; // decimal from backend
  total_current: string; // decimal from backend
  total_overdue: string; // decimal from backend
  overdue_percentage: string; // decimal from backend
  invoice_count: number;
  overdue_count: number;
  customer_count: number;
  average_days_overdue: string; // decimal from backend
}

export interface ARAgingReportMetadata {
  generated_at: string;
  as_of_date: string;
  query: ARAgingQuery;
  execution_time_ms: number;
}

export interface ARAgingReportResponse {
  summary: ARAgingReportSummary;
  buckets: ARAgingBucketSummary[];
  customer_summary?: ARAgingCustomerSummary[];
  invoices?: ARAgingInvoice[];
  metadata: ARAgingReportMetadata;
}

export interface ARAgingExportRequest {
  query: ARAgingQuery;
  format: 'csv' | 'json' | 'xlsx';
  include_sections?: string[];
  filename?: string;
}
