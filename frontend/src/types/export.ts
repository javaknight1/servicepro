// Export Types

export type ExportFormat = 'csv' | 'json' | 'xlsx' | 'pdf';
export type ExportType =
  | 'customers'
  | 'jobs'
  | 'invoices'
  | 'quotes'
  | 'revenue_report'
  | 'customer_report';
export type ExportJobStatus =
  | 'pending'
  | 'processing'
  | 'completed'
  | 'failed'
  | 'cancelled';

export interface ExportJobResponse {
  id: string;
  export_type: ExportType;
  format: ExportFormat;
  status: ExportJobStatus;
  total_records: number;
  processed_records: number;
  progress: number;
  filename?: string;
  file_size?: number;
  download_url?: string;
  error_message?: string;
  started_at?: string;
  completed_at?: string;
  expires_at?: string;
  created_at: string;
}

export interface ExportProgress {
  job_id: string;
  status: ExportJobStatus;
  total_records: number;
  processed_records: number;
  progress: number;
  current_phase: string;
  estimated_time_ms: number;
  elapsed_time_ms: number;
  message?: string;
}

export interface CSVExportOptions {
  delimiter: string;
  include_header: boolean;
  quote_all: boolean;
  line_ending: string;
  bom: boolean;
}

export interface PDFExportOptions {
  page_size: string;
  orientation: string;
  title: string;
  include_summary: boolean;
  include_charts: boolean;
  watermark_text: string;
  watermark_opacity: number;
  header_text: string;
  footer_text: string;
  show_page_numbers: boolean;
}

export interface ExcelExportOptions {
  sheet_name: string;
  include_formulas: boolean;
  freeze_panes: boolean;
  auto_filter: boolean;
  column_widths?: Record<string, number>;
}

export interface ExportConfig {
  format: ExportFormat;
  export_type: ExportType;
  include_sections?: string[];
  fields?: string[];
  encoding?: string;
  query?: Record<string, unknown>;
  csv_options?: CSVExportOptions;
  pdf_options?: PDFExportOptions;
  excel_options?: ExcelExportOptions;
}

export interface SupportedFormats {
  formats: Array<{
    value: ExportFormat;
    label: string;
    mime_type: string;
  }>;
  export_types: Array<{
    value: ExportType;
    label: string;
  }>;
  csv_options: {
    delimiters: Array<{ value: string; label: string }>;
  };
  pdf_options: {
    page_sizes: string[];
    orientations: string[];
  };
}

// Helper functions
export const getFormatLabel = (format: ExportFormat): string => {
  const labels: Record<ExportFormat, string> = {
    csv: 'CSV',
    json: 'JSON',
    xlsx: 'Excel',
    pdf: 'PDF',
  };
  return labels[format] || format;
};

export const getFormatIcon = (format: ExportFormat): string => {
  const icons: Record<ExportFormat, string> = {
    csv: '📊',
    json: '{}',
    xlsx: '📗',
    pdf: '📄',
  };
  return icons[format] || '📁';
};

export const getStatusColor = (status: ExportJobStatus): string => {
  const colors: Record<ExportJobStatus, string> = {
    pending: '#F59E0B',
    processing: '#3B82F6',
    completed: '#22C55E',
    failed: '#EF4444',
    cancelled: '#6B7280',
  };
  return colors[status] || '#9CA3AF';
};

export const formatFileSize = (bytes: number): string => {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`;
};

export const formatDuration = (ms: number): string => {
  if (ms < 1000) return `${ms}ms`;
  const seconds = Math.floor(ms / 1000);
  if (seconds < 60) return `${seconds}s`;
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = seconds % 60;
  return `${minutes}m ${remainingSeconds}s`;
};
