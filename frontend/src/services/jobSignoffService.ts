import api from './api';
import { JobStatus } from './jobService';

/**
 * The customer's sign-off on a visit (T176, models.JobSignoffResponse):
 * who signed, when, the signature (presigned `signature_url`, about an
 * hour) and whether the service report has been emailed.
 */
export interface JobSignoff {
  id: string;
  job_id: string;
  signer_name: string;
  signer_title?: string;
  signer_email?: string;
  signed_at: string;
  captured_by?: string;
  captured_by_name?: string;
  signature_url: string;
  signature_path: string;
  report_path: string;
  report_generated_at?: string;
  report_emailed_at?: string;
  report_recipients: string[];
  report_email_count: number;
  /** Who the report goes to now. */
  suggested_recipients: string[];
  /** The report was emailed: the sign-off can't be replaced. */
  locked: boolean;
}

interface JobSignoffEnvelope {
  signoff: JobSignoff | null;
  suggested_recipients: string[];
}

export interface JobSignoffState {
  signoff: JobSignoff | null;
  /** Who the report would be emailed to. */
  recipients: string[];
}

export interface CaptureSignoffInput {
  signer_name: string;
  signer_title?: string;
  signer_email?: string;
  /** PNG data URL from the signature canvas. */
  signature: string;
}

export interface EmailReportResult {
  /** False when it had already been emailed (nothing sent). */
  sent: boolean;
  signoff: JobSignoff;
}

/** Statuses a visit can be signed off in: started or done. */
export const SIGNABLE_STATUSES: ReadonlySet<string> = new Set([
  JobStatus.IN_PROGRESS,
  JobStatus.ON_HOLD,
  JobStatus.COMPLETED,
  JobStatus.INVOICED,
  JobStatus.PAID,
]);

const basePath = (jobId: string) => `/v1/jobs/${jobId}/signoff`;

/** Building the report fetches photos and renders a PDF (and emailing
 * sends it): longer than the client's default 10 s. */
const REPORT_TIMEOUT_MS = 60_000;

const trimmed = (v?: string) => {
  const t = v?.trim();
  return t ? t : undefined;
};

/** Customer sign-off and service report (T176), scoped to the job. */
export const jobSignoffService = {
  async get(jobId: string): Promise<JobSignoffState> {
    const response = await api.get<JobSignoffEnvelope>(basePath(jobId));
    return {
      signoff: response.data.signoff ?? null,
      recipients: response.data.suggested_recipients ?? [],
    };
  },

  async capture(
    jobId: string,
    input: CaptureSignoffInput
  ): Promise<JobSignoff> {
    const response = await api.post<JobSignoff>(basePath(jobId), {
      signer_name: input.signer_name.trim(),
      signer_title: trimmed(input.signer_title),
      signer_email: trimmed(input.signer_email),
      signature: input.signature,
    });
    return response.data;
  },

  async emailReport(jobId: string, resend = false): Promise<EmailReportResult> {
    const response = await api.post<EmailReportResult>(
      `${basePath(jobId)}/email`,
      { resend },
      { timeout: REPORT_TIMEOUT_MS }
    );
    return response.data;
  },

  /** A presigned link (about an hour) to the service report PDF. */
  async reportLink(jobId: string): Promise<string> {
    const response = await api.get<{ url: string }>(
      `${basePath(jobId)}/report/link`,
      { timeout: REPORT_TIMEOUT_MS }
    );
    return response.data.url;
  },
};

/** React Query key for a job's sign-off. */
export const jobSignoffQueryKey = (jobId: string) =>
  ['job-signoff', jobId] as const;
