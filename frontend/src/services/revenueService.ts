import api from './api';
import {
  RevenueReportQuery,
  RevenueReportResponse,
  RevenueDataPoint,
  RevenueByCategoryData,
  CustomerRevenueData,
  ReportExportRequest,
  ReportExportResponse,
  RevenueChartData,
} from '../types/revenue';
import { downloadCSV, downloadJSON } from '../utils/fileDownload';
import { buildQueryString } from '../utils/queryParams';

const REPORTS_BASE_URL = '/v1/reports/revenue';

/**
 * Get comprehensive revenue report
 */
export const getRevenueReport = async (
  query: RevenueReportQuery
): Promise<RevenueReportResponse> => {
  const queryString = buildQueryString({
    start_date: query.start_date,
    end_date: query.end_date,
    period_type: query.period_type,
    currency: query.currency,
    category: query.category,
    user_id: query.user_id,
  });

  const response = await api.get<RevenueReportResponse>(
    `${REPORTS_BASE_URL}${queryString}`
  );
  return response.data;
};

/**
 * Get revenue time series data
 */
export const getRevenueTimeSeries = async (
  query: RevenueReportQuery
): Promise<RevenueDataPoint[]> => {
  const queryString = buildQueryString({
    start_date: query.start_date,
    end_date: query.end_date,
    period_type: query.period_type,
    currency: query.currency,
  });

  const response = await api.get<{ data: RevenueDataPoint[] }>(
    `${REPORTS_BASE_URL}/time-series${queryString}`
  );
  return response.data.data;
};

/**
 * Get revenue category breakdown
 */
export const getRevenueCategoryBreakdown = async (
  query: RevenueReportQuery
): Promise<RevenueByCategoryData[]> => {
  const queryString = buildQueryString({
    start_date: query.start_date,
    end_date: query.end_date,
    currency: query.currency,
  });

  const response = await api.get<{ data: RevenueByCategoryData[] }>(
    `${REPORTS_BASE_URL}/categories${queryString}`
  );
  return response.data.data;
};

/**
 * Get top customers by revenue
 */
export const getTopCustomers = async (
  query: RevenueReportQuery,
  limit: number = 10
): Promise<CustomerRevenueData[]> => {
  const queryString = buildQueryString({
    start_date: query.start_date,
    end_date: query.end_date,
    limit,
  });

  const response = await api.get<{ data: CustomerRevenueData[] }>(
    `${REPORTS_BASE_URL}/customers${queryString}`
  );
  return response.data.data;
};

/**
 * Get chart-formatted revenue data
 */
export const getRevenueChartData = async (
  query: RevenueReportQuery,
  chartType: 'line' | 'bar' | 'pie' = 'line'
): Promise<RevenueChartData> => {
  const queryString = buildQueryString({
    start_date: query.start_date,
    end_date: query.end_date,
    period_type: query.period_type,
    chart_type: chartType,
  });

  const response = await api.get<RevenueChartData>(
    `${REPORTS_BASE_URL}/chart-data${queryString}`
  );
  return response.data;
};

/**
 * Export revenue report
 */
export const exportRevenueReport = async (
  request: ReportExportRequest
): Promise<ReportExportResponse | Blob> => {
  const response = await api.post<ReportExportResponse | Blob>(
    `${REPORTS_BASE_URL}/export`,
    request,
    {
      responseType: request.format === 'csv' ? 'blob' : 'json',
    }
  );
  return response.data;
};

/**
 * Download CSV export directly
 */
export const downloadCSVExport = async (
  query: RevenueReportQuery,
  filename: string = 'revenue_report.csv'
): Promise<void> => {
  const request: ReportExportRequest = {
    query,
    format: 'csv',
    include_sections: ['summary', 'timeseries', 'categories'],
    filename,
  };

  const response = await api.post(`${REPORTS_BASE_URL}/export`, request, {
    responseType: 'blob',
  });

  downloadCSV(response.data, filename);
};

/**
 * Download JSON export
 */
export const downloadJSONExport = async (
  query: RevenueReportQuery,
  filename: string = 'revenue_report.json'
): Promise<void> => {
  const report = await getRevenueReport(query);
  downloadJSON(report, filename);
};

/**
 * Refresh revenue cache
 */
export const refreshRevenueCache = async (
  periodType: string,
  periodStart: string,
  periodEnd: string
): Promise<{ message: string }> => {
  const queryString = buildQueryString({
    period_type: periodType,
    period_start: periodStart,
    period_end: periodEnd,
  });

  const response = await api.post<{ message: string }>(
    `${REPORTS_BASE_URL}/cache/refresh${queryString}`
  );
  return response.data;
};

// Named export for index.ts re-export
export const revenueService = {
  getRevenueReport,
  getRevenueTimeSeries,
  getRevenueCategoryBreakdown,
  getTopCustomers,
  getRevenueChartData,
  exportRevenueReport,
  downloadCSVExport,
  downloadJSONExport,
  refreshRevenueCache,
};

export default revenueService;
