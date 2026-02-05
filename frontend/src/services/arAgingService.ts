import api from './api';
import {
  ARAgingQuery,
  ARAgingReportResponse,
  ARAgingBucketSummary,
  ARAgingInvoice,
  ARAgingCustomerSummary,
  ARAgingExportRequest,
  AgingBucket,
} from '../types/arAging';
import { buildQueryString } from '../utils/queryParams';
import { downloadCSV } from '../utils/fileDownload';

const REPORTS_BASE_URL = '/v1/reports/ar-aging';

/**
 * Get full A/R aging report
 */
export const getARAgingReport = async (
  query: ARAgingQuery = {}
): Promise<ARAgingReportResponse> => {
  const queryString = buildQueryString({
    as_of_date: query.as_of_date,
    customer_id: query.customer_id,
    min_amount: query.min_amount,
    include_paid: query.include_paid,
    include_draft: query.include_draft,
  });

  const response = await api.get<ARAgingReportResponse>(
    `${REPORTS_BASE_URL}${queryString}`
  );
  return response.data;
};

/**
 * Get bucket summary only
 */
export const getBucketSummary = async (
  query: ARAgingQuery = {}
): Promise<ARAgingBucketSummary[]> => {
  const queryString = buildQueryString({
    as_of_date: query.as_of_date,
    customer_id: query.customer_id,
  });

  const response = await api.get<{ data: ARAgingBucketSummary[] }>(
    `${REPORTS_BASE_URL}/buckets${queryString}`
  );
  return response.data.data;
};

/**
 * Get invoices for a specific aging bucket (drill-down)
 */
export const getInvoicesByBucket = async (
  bucket: AgingBucket,
  query: ARAgingQuery = {}
): Promise<ARAgingInvoice[]> => {
  const queryString = buildQueryString({
    as_of_date: query.as_of_date,
    customer_id: query.customer_id,
    min_amount: query.min_amount,
  });

  const response = await api.get<{ data: ARAgingInvoice[]; count: number }>(
    `${REPORTS_BASE_URL}/buckets/${bucket}/invoices${queryString}`
  );
  return response.data.data;
};

/**
 * Get A/R aging grouped by customer
 */
export const getCustomerSummary = async (
  query: ARAgingQuery = {}
): Promise<ARAgingCustomerSummary[]> => {
  const queryString = buildQueryString({
    as_of_date: query.as_of_date,
    min_amount: query.min_amount,
  });

  const response = await api.get<{
    data: ARAgingCustomerSummary[];
    count: number;
  }>(`${REPORTS_BASE_URL}/customers${queryString}`);
  return response.data.data;
};

/**
 * Export A/R aging report
 */
export const exportARAgingReport = async (
  request: ARAgingExportRequest
): Promise<void> => {
  const response = await api.post(`${REPORTS_BASE_URL}/export`, request, {
    responseType: request.format === 'csv' ? 'blob' : 'json',
  });

  if (request.format === 'csv') {
    const filename =
      request.filename ||
      `ar_aging_report_${new Date().toISOString().split('T')[0]}.csv`;
    downloadCSV(response.data, filename);
  }
};

/**
 * Download CSV export directly
 */
export const downloadCSVExport = async (
  query: ARAgingQuery = {},
  filename?: string
): Promise<void> => {
  const request: ARAgingExportRequest = {
    query,
    format: 'csv',
    include_sections: ['summary', 'invoices'],
    filename,
  };

  await exportARAgingReport(request);
};

export const arAgingService = {
  getARAgingReport,
  getBucketSummary,
  getInvoicesByBucket,
  getCustomerSummary,
  exportARAgingReport,
  downloadCSVExport,
};

export default arAgingService;
