import api from './api';
import {
  ExportConfig,
  ExportJobResponse,
  ExportProgress,
  SupportedFormats,
} from '../types/export';
import { downloadFile } from '../utils/fileDownload';

const EXPORTS_BASE_URL = '/v1/exports';

/**
 * Start an async export job
 */
export const startExport = async (
  config: ExportConfig
): Promise<ExportJobResponse> => {
  const response = await api.post<ExportJobResponse>(EXPORTS_BASE_URL, config);
  return response.data;
};

/**
 * Get export job status
 */
export const getExportStatus = async (
  jobId: string
): Promise<ExportJobResponse> => {
  const response = await api.get<ExportJobResponse>(
    `${EXPORTS_BASE_URL}/${jobId}`
  );
  return response.data;
};

/**
 * Get real-time export progress
 */
export const getExportProgress = async (
  jobId: string
): Promise<ExportProgress> => {
  const response = await api.get<ExportProgress>(
    `${EXPORTS_BASE_URL}/${jobId}/progress`
  );
  return response.data;
};

/**
 * Subscribe to export progress via SSE
 */
export const subscribeToExportProgress = (
  jobId: string,
  onProgress: (progress: ExportProgress) => void,
  onComplete: (progress: ExportProgress) => void,
  onError: (error: Error) => void
): (() => void) => {
  const eventSource = new EventSource(`${EXPORTS_BASE_URL}/${jobId}/stream`);

  eventSource.addEventListener('progress', (event) => {
    const progress: ExportProgress = JSON.parse(event.data);
    onProgress(progress);

    if (['completed', 'failed', 'cancelled'].includes(progress.status)) {
      onComplete(progress);
      eventSource.close();
    }
  });

  eventSource.onerror = (_error) => {
    onError(new Error('SSE connection error'));
    eventSource.close();
  };

  // Return cleanup function
  return () => {
    eventSource.close();
  };
};

/**
 * Download completed export
 */
export const downloadExport = async (
  jobId: string,
  filename?: string
): Promise<void> => {
  const response = await api.get(`${EXPORTS_BASE_URL}/${jobId}/download`, {
    responseType: 'blob',
  });

  const blob = new Blob([response.data]);
  downloadFile(blob, filename || 'export');
};

/**
 * Cancel an in-progress export
 */
export const cancelExport = async (jobId: string): Promise<void> => {
  await api.post(`${EXPORTS_BASE_URL}/${jobId}/cancel`);
};

/**
 * List recent exports
 */
export const listExports = async (
  limit: number = 20
): Promise<ExportJobResponse[]> => {
  const response = await api.get<ExportJobResponse[]>(EXPORTS_BASE_URL, {
    params: { limit },
  });
  return response.data;
};

/**
 * Quick export (synchronous, for small datasets)
 */
export const quickExport = async (config: ExportConfig): Promise<Blob> => {
  const response = await api.post(`${EXPORTS_BASE_URL}/quick`, config, {
    responseType: 'blob',
  });
  return response.data;
};

/**
 * Quick export and download
 */
export const quickExportAndDownload = async (
  config: ExportConfig,
  filename?: string
): Promise<void> => {
  const blob = await quickExport(config);
  downloadFile(
    blob,
    filename || `${config.export_type}_export.${config.format}`
  );
};

/**
 * Get supported export formats and types
 */
export const getSupportedFormats = async (): Promise<SupportedFormats> => {
  const response = await api.get<SupportedFormats>(
    `${EXPORTS_BASE_URL}/formats`
  );
  return response.data;
};

/**
 * Poll export status until completion
 */
export const pollExportStatus = async (
  jobId: string,
  onProgress?: (job: ExportJobResponse) => void,
  intervalMs: number = 1000,
  timeoutMs: number = 300000 // 5 minutes
): Promise<ExportJobResponse> => {
  const startTime = Date.now();

  return new Promise((resolve, reject) => {
    const poll = async () => {
      try {
        const job = await getExportStatus(jobId);
        onProgress?.(job);

        if (job.status === 'completed') {
          resolve(job);
          return;
        }

        if (job.status === 'failed') {
          reject(new Error(job.error_message || 'Export failed'));
          return;
        }

        if (job.status === 'cancelled') {
          reject(new Error('Export was cancelled'));
          return;
        }

        // Check timeout
        if (Date.now() - startTime > timeoutMs) {
          reject(new Error('Export timeout'));
          return;
        }

        // Continue polling
        setTimeout(poll, intervalMs);
      } catch (error) {
        reject(error);
      }
    };

    poll();
  });
};

/**
 * Start export and wait for completion
 */
export const startExportAndWait = async (
  config: ExportConfig,
  onProgress?: (job: ExportJobResponse) => void
): Promise<ExportJobResponse> => {
  const job = await startExport(config);
  return pollExportStatus(job.id, onProgress);
};

/**
 * Start export, wait, and download
 */
export const exportAndDownload = async (
  config: ExportConfig,
  onProgress?: (job: ExportJobResponse) => void
): Promise<void> => {
  const job = await startExportAndWait(config, onProgress);
  await downloadExport(job.id, job.filename);
};

export const exportService = {
  startExport,
  getExportStatus,
  getExportProgress,
  subscribeToExportProgress,
  downloadExport,
  cancelExport,
  listExports,
  quickExport,
  quickExportAndDownload,
  getSupportedFormats,
  pollExportStatus,
  startExportAndWait,
  exportAndDownload,
};

export default exportService;
