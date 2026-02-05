import api from './api';
import {
  CustomerReportQuery,
  CustomerReportResponse,
  SegmentBreakdownData,
  TypeBreakdownData,
  GeographyBreakdownData,
  AcquisitionTrendData,
  CustomerMetricsData,
  CohortData,
  CustomerChartData,
} from '../types/customer';
import { ReportExportResponse } from '../types/revenue';
import { downloadCSV } from '../utils/fileDownload';
import { buildQueryString } from '../utils/queryParams';

const REPORTS_BASE_URL = '/v1/reports/customers';

/**
 * Get comprehensive customer report
 */
export const getCustomerReport = async (
  query: CustomerReportQuery
): Promise<CustomerReportResponse> => {
  const queryString = buildQueryString({
    start_date: query.start_date,
    end_date: query.end_date,
    segment: query.segment,
    customer_type: query.customer_type,
    status: query.status,
    state: query.state,
    sort_by: query.sort_by,
    sort_order: query.sort_order,
  });

  const response = await api.get<CustomerReportResponse>(
    `${REPORTS_BASE_URL}${queryString}`
  );
  return response.data;
};

/**
 * Get segment breakdown
 */
export const getSegmentBreakdown = async (
  query: CustomerReportQuery = {}
): Promise<SegmentBreakdownData[]> => {
  const queryString = buildQueryString({
    customer_type: query.customer_type,
    status: query.status,
  });

  const response = await api.get<{ data: SegmentBreakdownData[] }>(
    `${REPORTS_BASE_URL}/segments${queryString}`
  );
  return response.data.data;
};

/**
 * Get type breakdown
 */
export const getTypeBreakdown = async (
  query: CustomerReportQuery = {}
): Promise<TypeBreakdownData[]> => {
  const queryString = buildQueryString({
    state: query.state,
  });

  const response = await api.get<{ data: TypeBreakdownData[] }>(
    `${REPORTS_BASE_URL}/types${queryString}`
  );
  return response.data.data;
};

/**
 * Get geography breakdown
 */
export const getGeographyBreakdown = async (
  query: CustomerReportQuery = {}
): Promise<GeographyBreakdownData[]> => {
  const queryString = buildQueryString({
    customer_type: query.customer_type,
  });

  const response = await api.get<{ data: GeographyBreakdownData[] }>(
    `${REPORTS_BASE_URL}/geography${queryString}`
  );
  return response.data.data;
};

/**
 * Get acquisition trend
 */
export const getAcquisitionTrend = async (
  query: CustomerReportQuery = {}
): Promise<AcquisitionTrendData[]> => {
  const queryString = buildQueryString({
    start_date: query.start_date,
    end_date: query.end_date,
  });

  const response = await api.get<{ data: AcquisitionTrendData[] }>(
    `${REPORTS_BASE_URL}/acquisition${queryString}`
  );
  return response.data.data;
};

/**
 * Get top customers
 */
export const getTopCustomers = async (
  query: CustomerReportQuery = {},
  limit: number = 10
): Promise<CustomerMetricsData[]> => {
  const queryString = buildQueryString({
    limit,
    segment: query.segment,
    customer_type: query.customer_type,
  });

  const response = await api.get<{ data: CustomerMetricsData[] }>(
    `${REPORTS_BASE_URL}/top${queryString}`
  );
  return response.data.data;
};

/**
 * Get at-risk customers
 */
export const getAtRiskCustomers = async (
  limit: number = 10
): Promise<CustomerMetricsData[]> => {
  const response = await api.get<{ data: CustomerMetricsData[] }>(
    `${REPORTS_BASE_URL}/at-risk?limit=${limit}`
  );
  return response.data.data;
};

/**
 * Get cohort data
 */
export const getCohortData = async (
  months: number = 12
): Promise<CohortData[]> => {
  const response = await api.get<{ data: CohortData[] }>(
    `${REPORTS_BASE_URL}/cohorts?months=${months}`
  );
  return response.data.data;
};

/**
 * Get chart data
 */
export const getChartData = async (
  chartType: 'segments' | 'types' | 'geography' | 'acquisition' = 'segments',
  query: CustomerReportQuery = {}
): Promise<CustomerChartData> => {
  const queryString = buildQueryString({
    chart_type: chartType,
    start_date: query.start_date,
    end_date: query.end_date,
  });

  const response = await api.get<CustomerChartData>(
    `${REPORTS_BASE_URL}/chart-data${queryString}`
  );
  return response.data;
};

/**
 * Export customer report
 */
export const exportCustomerReport = async (
  query: CustomerReportQuery,
  format: 'csv' | 'json' = 'csv',
  includeSections?: string[]
): Promise<ReportExportResponse | Blob> => {
  const response = await api.post<ReportExportResponse | Blob>(
    `${REPORTS_BASE_URL}/export`,
    {
      query,
      format,
      include_sections: includeSections,
    },
    {
      responseType: format === 'csv' ? 'blob' : 'json',
    }
  );
  return response.data;
};

/**
 * Download CSV export
 */
export const downloadCSVExport = async (
  query: CustomerReportQuery,
  filename: string = 'customer_report.csv'
): Promise<void> => {
  const response = await api.post(
    `${REPORTS_BASE_URL}/export`,
    {
      query,
      format: 'csv',
      include_sections: ['summary', 'segments', 'top_customers'],
    },
    {
      responseType: 'blob',
    }
  );

  downloadCSV(response.data, filename);
};

/**
 * Refresh metrics cache
 */
export const refreshMetricsCache = async (
  customerId?: string
): Promise<{ message: string }> => {
  const params = customerId ? `?customer_id=${customerId}` : '';
  const response = await api.post<{ message: string }>(
    `${REPORTS_BASE_URL}/cache/refresh${params}`
  );
  return response.data;
};

export const customerReportService = {
  getCustomerReport,
  getSegmentBreakdown,
  getTypeBreakdown,
  getGeographyBreakdown,
  getAcquisitionTrend,
  getTopCustomers,
  getAtRiskCustomers,
  getCohortData,
  getChartData,
  exportCustomerReport,
  downloadCSVExport,
  refreshMetricsCache,
};

export default customerReportService;
