import api from './api';

/**
 * A photo or file on a job (T174, models.JobAttachmentResponse). Photos are
 * stored as JPEG with a thumbnail; files (PDF) have none. `url` and
 * `thumb_url` are presigned links valid for about an hour, so the list is
 * refetched well before they expire.
 */
export interface JobAttachment {
  id: string;
  job_id: string;
  kind: 'photo' | 'file';
  filename: string;
  content_type: string;
  size_bytes: number;
  width?: number;
  height?: number;
  caption?: string;
  uploaded_by?: string;
  uploaded_by_name?: string;
  url: string;
  thumb_url?: string;
  /** API path that redirects to a fresh link (for invoices and reports). */
  file_path: string;
  thumb_path?: string;
  created_at: string;
}

interface JobAttachmentListResponse {
  attachments: JobAttachment[];
  total: number;
}

/** Largest upload the API accepts (15 MB). */
export const MAX_ATTACHMENT_BYTES = 15 * 1024 * 1024;

/** What the file picker offers: photos and PDFs. */
export const ATTACHMENT_ACCEPT =
  'image/jpeg,image/png,image/webp,application/pdf';

const basePath = (jobId: string) => `/v1/jobs/${jobId}/attachments`;

/** Job photos and files (T174), scoped to the job in the URL. */
export const jobAttachmentService = {
  async list(jobId: string): Promise<JobAttachment[]> {
    const response = await api.get<JobAttachmentListResponse>(basePath(jobId));
    return response.data.attachments ?? [];
  },

  async upload(
    jobId: string,
    file: File,
    caption?: string,
    onProgress?: (percent: number) => void
  ): Promise<JobAttachment> {
    const form = new FormData();
    form.append('file', file, file.name);
    if (caption?.trim()) form.append('caption', caption.trim());
    const response = await api.post<JobAttachment>(basePath(jobId), form, {
      headers: { 'Content-Type': 'multipart/form-data' },
      onUploadProgress: (e) => {
        if (onProgress && e.total) {
          onProgress(Math.round((e.loaded / e.total) * 100));
        }
      },
    });
    return response.data;
  },

  async updateCaption(
    jobId: string,
    attachmentId: string,
    caption: string
  ): Promise<JobAttachment> {
    const response = await api.patch<JobAttachment>(
      `${basePath(jobId)}/${attachmentId}`,
      { caption }
    );
    return response.data;
  },

  async remove(jobId: string, attachmentId: string): Promise<void> {
    await api.delete(`${basePath(jobId)}/${attachmentId}`);
  },
};

/** React Query key for a job's attachments. */
export const jobAttachmentsQueryKey = (jobId: string) =>
  ['job-attachments', jobId] as const;
