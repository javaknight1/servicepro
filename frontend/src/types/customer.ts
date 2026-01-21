// Customer Types

export type CustomerSegment =
  | 'high_value'
  | 'regular'
  | 'occasional'
  | 'at_risk'
  | 'churned'
  | 'new';
export type CustomerType = 'residential' | 'commercial';
export type CustomerStatus = 'active' | 'inactive' | 'prospect';

export interface Customer {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  company_name?: string;
  phone?: string;
  address?: string;
  city?: string;
  state?: string;
  zip_code?: string;
  customer_type: CustomerType;
  status: CustomerStatus;
  notes?: string;
  created_at: string;
  updated_at: string;
}

export interface CustomerFilters {
  search?: string;
  status?: CustomerStatus;
  customer_type?: CustomerType;
  page?: number;
  page_size?: number;
  sort_by?: string;
  sort_order?: 'asc' | 'desc';
}

export interface CustomerListResponse {
  customers: Customer[];
  total: number;
  page: number;
  page_size: number;
}

export interface CustomerStats {
  total_customers: number;
  active_customers: number;
  inactive_customers: number;
  prospects: number;
  residential_customers: number;
  commercial_customers: number;
}

export const getCustomerStatusLabel = (status: CustomerStatus): string => {
  const labels: Record<CustomerStatus, string> = {
    active: 'Active',
    inactive: 'Inactive',
    prospect: 'Prospect',
  };
  return labels[status] || status;
};

export const getCustomerTypeLabel = (type: CustomerType): string => {
  const labels: Record<CustomerType, string> = {
    residential: 'Residential',
    commercial: 'Commercial',
  };
  return labels[type] || type;
};

// Customer Report Types

export interface CustomerReportQuery {
  start_date?: string;
  end_date?: string;
  segment?: CustomerSegment;
  customer_type?: CustomerType;
  status?: CustomerStatus;
  state?: string;
  city?: string;
  min_revenue?: number;
  max_revenue?: number;
  min_jobs?: number;
  max_jobs?: number;
  sort_by?: string;
  sort_order?: 'asc' | 'desc';
  page?: number;
  page_size?: number;
}

export interface CustomerReportSummary {
  total_customers: number;
  active_customers: number;
  inactive_customers: number;
  prospects: number;
  new_customers_this_period: number;
  churned_customers: number;
  total_revenue: string;
  average_revenue_per_customer: string;
  average_lifetime_value: string;
  average_jobs_per_customer: string;
  overall_quote_acceptance_rate: string;
  growth_rate: string;
  retention_rate: string;
}

export interface SegmentBreakdownData {
  segment: CustomerSegment;
  display_name: string;
  count: number;
  percentage: string;
  total_revenue: string;
  average_revenue: string;
  color: string;
}

export interface TypeBreakdownData {
  customer_type: CustomerType;
  status: CustomerStatus;
  count: number;
  percentage: string;
  total_revenue: string;
}

export interface GeographyBreakdownData {
  state: string;
  city?: string;
  customer_count: number;
  active_count: number;
  percentage: string;
  total_revenue: string;
}

export interface AcquisitionTrendData {
  period: string;
  new_customers: number;
  active_customers: number;
  prospects: number;
  churned_customers: number;
  net_growth: number;
}

export interface CustomerMetricsData {
  customer_id: string;
  first_name: string;
  last_name: string;
  company_name?: string;
  email: string;
  customer_type: CustomerType;
  status: CustomerStatus;
  segment: CustomerSegment;
  total_revenue: string;
  total_jobs: number;
  completed_jobs: number;
  total_quotes: number;
  accepted_quotes: number;
  quote_acceptance_rate: string;
  average_job_value: string;
  last_activity_date?: string;
  days_since_last_activity: number;
  customer_lifetime_days: number;
  state: string;
  city: string;
}

export interface CustomerReportMetadata {
  generated_at: string;
  query: CustomerReportQuery;
  total_records: number;
  cache_used: boolean;
  execution_time_ms: number;
}

export interface CustomerReportResponse {
  summary: CustomerReportSummary;
  segment_breakdown: SegmentBreakdownData[];
  type_breakdown: TypeBreakdownData[];
  geography_breakdown?: GeographyBreakdownData[];
  acquisition_trend?: AcquisitionTrendData[];
  top_customers?: CustomerMetricsData[];
  at_risk_customers?: CustomerMetricsData[];
  metadata: CustomerReportMetadata;
}

export interface CohortData {
  cohort_month: string;
  cohort_size: number;
  month_1_retained: number;
  month_2_retained: number;
  month_3_retained: number;
  month_6_retained: number;
  retention_rate_1m: string;
  retention_rate_3m: string;
  retention_rate_6m: string;
}

export interface CustomerChartData {
  labels: string[];
  datasets: CustomerChartDataset[];
  options?: Record<string, unknown>;
}

export interface CustomerChartDataset {
  label: string;
  data: number[];
  backgroundColor?: string[];
  borderColor?: string;
  fill?: boolean;
}

// Helper functions
export const getSegmentDisplayName = (segment: CustomerSegment): string => {
  const names: Record<CustomerSegment, string> = {
    high_value: 'High Value',
    regular: 'Regular',
    occasional: 'Occasional',
    at_risk: 'At Risk',
    churned: 'Churned',
    new: 'New',
  };
  return names[segment] || segment;
};

export const getSegmentColor = (segment: CustomerSegment): string => {
  const colors: Record<CustomerSegment, string> = {
    high_value: '#22C55E',
    regular: '#3B82F6',
    occasional: '#F59E0B',
    at_risk: '#EF4444',
    churned: '#6B7280',
    new: '#8B5CF6',
  };
  return colors[segment] || '#9CA3AF';
};
