import api from './api';

/** Where an import file came from (T193). */
export type ImportSource = 'jobber' | 'housecall_pro' | 'generic';

/** What one import file holds, in the order they should be imported. */
export type ImportEntity = 'customers' | 'sites' | 'jobs' | 'invoices';

export const IMPORT_ENTITIES: ImportEntity[] = [
  'customers',
  'sites',
  'jobs',
  'invoices',
];

/** Largest file the API accepts (10 MiB; it also caps files at 50,000 rows). */
export const MAX_IMPORT_BYTES = 10 * 1024 * 1024;

/** What the file picker offers. */
export const IMPORT_ACCEPT =
  '.csv,.xlsx,text/csv,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet';

/** A target field a column can map to (models.ImportField). */
export interface ImportField {
  key: string;
  label: string;
  required: boolean;
  /** At least one field of a group must be mapped. */
  group?: string;
  help?: string;
}

export interface ImportRowError {
  row: number;
  field?: string;
  message: string;
}

export interface ImportPreviewRow {
  row: number;
  values: Record<string, string>;
  errors?: ImportRowError[];
  /** Why a valid row will not be imported (e.g. already paid). */
  skip?: string;
}

/** POST /imports/{entity}/preview (models.ImportPreviewResponse). */
export interface ImportPreview {
  source: ImportSource;
  entity: ImportEntity;
  filename: string;
  headers: string[];
  fields: ImportField[];
  /** Field key → column header. */
  mapping: Record<string, string>;
  /** What the mapping lacks; the import can't start until it's empty. */
  missing: string[];
  total_rows: number;
  valid_rows: number;
  skip_rows: number;
  error_rows: number;
  rows: ImportPreviewRow[];
  errors: ImportRowError[];
}

export type ImportJobStatus =
  'pending' | 'processing' | 'completed' | 'failed' | 'cancelled';

/** An import job (models.ImportStatusResponse). */
export interface ImportJob {
  job_id: string;
  status: ImportJobStatus;
  source: string;
  entity: ImportEntity;
  filename: string;
  total_rows: number;
  processed_rows: number;
  successful_rows: number;
  failed_rows: number;
  skipped_rows: number;
  progress: number;
  error_summary?: string | null;
  errors?: {
    row_number: number;
    field?: string;
    error: string;
    row_data?: string;
  }[];
  created_at: string;
  completed_at?: string | null;
}

export function isImportFinished(job: ImportJob): boolean {
  return (
    job.status === 'completed' ||
    job.status === 'failed' ||
    job.status === 'cancelled'
  );
}

// Starting or continuing an import runs up to 20 s of work on the server.
const RUN_TIMEOUT_MS = 60_000;

function uploadForm(
  file: File,
  source: ImportSource,
  mapping?: Record<string, string>
): FormData {
  const form = new FormData();
  form.append('file', file, file.name);
  form.append('source', source);
  if (mapping && Object.keys(mapping).length > 0) {
    form.append('mapping', JSON.stringify(mapping));
  }
  return form;
}

const multipart = { 'Content-Type': 'multipart/form-data' };

/** Imports from Jobber / Housecall Pro exports or any CSV (T193). */
export const importService = {
  /** Maps and validates the file; nothing is saved. */
  async preview(
    entity: ImportEntity,
    file: File,
    source: ImportSource,
    mapping?: Record<string, string>
  ): Promise<ImportPreview> {
    const response = await api.post<ImportPreview>(
      `/v1/imports/${entity}/preview`,
      uploadForm(file, source, mapping),
      { headers: multipart, timeout: RUN_TIMEOUT_MS }
    );
    return response.data;
  },

  /** Starts the import; the first chunk runs during the request. */
  async start(
    entity: ImportEntity,
    file: File,
    source: ImportSource,
    mapping?: Record<string, string>
  ): Promise<ImportJob> {
    const response = await api.post<ImportJob>(
      `/v1/imports/${entity}`,
      uploadForm(file, source, mapping),
      { headers: multipart, timeout: RUN_TIMEOUT_MS }
    );
    return response.data;
  },

  /** Runs the next chunk of a pending import and returns its status. */
  async run(entity: ImportEntity, jobId: string): Promise<ImportJob> {
    const response = await api.post<ImportJob>(
      `/v1/imports/${entity}/${jobId}/run`,
      undefined,
      { timeout: RUN_TIMEOUT_MS }
    );
    return response.data;
  },

  async get(entity: ImportEntity, jobId: string): Promise<ImportJob> {
    const response = await api.get<ImportJob>(`/v1/imports/${entity}/${jobId}`);
    return response.data;
  },
};
