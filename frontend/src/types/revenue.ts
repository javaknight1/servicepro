// Revenue Report Types

export type PeriodType =
  | 'daily'
  | 'weekly'
  | 'monthly'
  | 'quarterly'
  | 'yearly';

export type ExportFormat = 'csv' | 'json' | 'xlsx' | 'pdf';

export interface RevenueReportQuery {
  start_date?: string;
  end_date?: string;
  period_type?: PeriodType;
  currency?: string;
  category?: string;
  subcategory?: string;
  user_id?: string;
  transaction_type?: string;
  include_refunds?: boolean;
  grouping?: string;
  min_amount?: number;
  max_amount?: number;
  page?: number;
  page_size?: number;
}

export interface RevenueSummary {
  total_gross_revenue: string;
  total_net_revenue: string;
  total_fees: string;
  total_tax: string;
  total_discounts: string;
  total_refunds: string;
  transaction_count: number;
  payment_count: number;
  refund_count: number;
  unique_customer_count: number;
  average_transaction_value: string;
  average_order_value: string;
  growth_rate: string;
  previous_period_revenue: string;
  currency: string;
}

export interface RevenueDataPoint {
  period: string;
  gross_revenue: string;
  net_revenue: string;
  fees: string;
  tax: string;
  discounts: string;
  refunds: string;
  transaction_count: number;
  unique_customers: number;
  average_transaction_value: string;
}

export interface RevenueByCategoryData {
  category: string;
  subcategory?: string;
  revenue: string;
  transaction_count: number;
  average_value: string;
  percentage: string;
}

export interface CustomerRevenueData {
  user_id: string;
  user_email?: string;
  user_name?: string;
  total_revenue: string;
  transaction_count: number;
  average_transaction_value: string;
  first_transaction_date: string;
  last_transaction_date: string;
  lifetime_value: string;
}

export interface RevenueReportMetadata {
  generated_at: string;
  query: RevenueReportQuery;
  data_points: number;
  cache_used: boolean;
  execution_time_ms: number;
}

export interface RevenueReportResponse {
  summary: RevenueSummary;
  time_series: RevenueDataPoint[];
  by_category?: RevenueByCategoryData[];
  top_customers?: CustomerRevenueData[];
  metadata: RevenueReportMetadata;
}

export interface RevenueChartData {
  labels: string[];
  datasets: RevenueChartDataset[];
  options?: Record<string, unknown>;
}

export interface RevenueChartDataset {
  label: string;
  data: number[];
  backgroundColor?: string;
  borderColor?: string;
  fill?: boolean;
  tension?: number;
}

export interface ReportExportRequest {
  query: RevenueReportQuery;
  format: ExportFormat;
  include_sections?: string[];
  include_charts?: boolean;
  filename?: string;
}

export interface ReportExportResponse {
  export_id: string;
  file_name: string;
  format: ExportFormat;
  status: string;
  data?: unknown;
  file_size: number;
  expires_at: string;
  generated_at: string;
}

// Chart configuration types
export interface ChartOptions {
  responsive: boolean;
  maintainAspectRatio: boolean;
  plugins: {
    legend: {
      position: 'top' | 'bottom' | 'left' | 'right';
      display: boolean;
    };
    title: {
      display: boolean;
      text: string;
    };
    tooltip: {
      mode: 'index' | 'point' | 'nearest' | 'dataset';
      intersect: boolean;
    };
  };
  scales?: {
    x?: {
      display: boolean;
      title?: {
        display: boolean;
        text: string;
      };
    };
    y?: {
      display: boolean;
      beginAtZero?: boolean;
      title?: {
        display: boolean;
        text: string;
      };
    };
  };
}
