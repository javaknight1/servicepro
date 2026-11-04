/**
 * The customer's sign-off and the service report (T176). The office card
 * follows mockup 09 (signature, signer, time; report emailed to whom and
 * when; download, email, resend); the phone variant on the technician's
 * visit page shows what was signed and, once the visit is done, offers to
 * email the report. SignoffCapture opens the full-screen signature pad.
 */
import { useState } from 'react';
import { isAxiosError } from 'axios';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Download, Mail, PenLine, Send } from 'lucide-react';
import {
  SIGNABLE_STATUSES,
  jobSignoffQueryKey,
  jobSignoffService,
  type JobSignoff,
} from '@services/jobSignoffService';
import { JobStatus } from '@services/jobService';
import { cn } from '@utils/cn';
import { getErrorMessage } from '@/utils/error';
import { openServiceReport, signoffTime, useJobSignoff } from './jobSignoff';
import { SignaturePad, type SignatureSubmission } from './SignaturePad';

export interface SignoffCaptureProps {
  jobId: string;
  subtitle?: string;
  defaultSignerName?: string;
  onClose: () => void;
}

/** The signature pad wired to the API. */
export function SignoffCapture({
  jobId,
  subtitle,
  defaultSignerName,
  onClose,
}: SignoffCaptureProps) {
  const queryClient = useQueryClient();
  const capture = useMutation({
    mutationFn: (value: SignatureSubmission) =>
      jobSignoffService.capture(jobId, value),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: jobSignoffQueryKey(jobId) });
      onClose();
    },
  });
  return (
    <SignaturePad
      subtitle={subtitle}
      defaultSignerName={defaultSignerName}
      pending={capture.isPending}
      error={
        capture.isError
          ? getErrorMessage(capture.error, "Couldn't save the signature")
          : null
      }
      onSubmit={(v) => capture.mutate(v)}
      onCancel={onClose}
    />
  );
}

export interface JobSignoffSectionProps {
  jobId: string;
  jobStatus: string;
  /** 'phone' for the technician's visit page. */
  variant?: 'office' | 'phone';
  /** May sign and email (jobs.update). */
  canEdit: boolean;
  /** IANA zone for the times shown (the tenant's). */
  timeZone?: string;
  /** For the signature pad's header and default signer. */
  siteName?: string;
  defaultSignerName?: string;
}

function signerLine(s: JobSignoff): string {
  return s.signer_title ? `${s.signer_name}, ${s.signer_title}` : s.signer_name;
}

export function JobSignoffSection({
  jobId,
  jobStatus,
  variant = 'office',
  canEdit,
  timeZone,
  siteName,
  defaultSignerName,
}: JobSignoffSectionProps) {
  const phone = variant === 'phone';
  const queryClient = useQueryClient();
  const [signing, setSigning] = useState(false);
  const [notice, setNotice] = useState<string | null>(null);
  const { data, isLoading, error } = useJobSignoff(jobId);

  const email = useMutation({
    mutationFn: (resend: boolean) =>
      jobSignoffService.emailReport(jobId, resend),
    onSuccess: (res) => {
      queryClient.setQueryData(jobSignoffQueryKey(jobId), {
        signoff: res.signoff,
        recipients: res.signoff.suggested_recipients,
      });
      setNotice(
        res.sent
          ? `Service report sent to ${res.signoff.report_recipients.join(', ')}.`
          : 'The service report had already been emailed.'
      );
    },
  });
  const download = useMutation({
    mutationFn: () => openServiceReport(jobId),
  });

  // No object storage on this server: the feature isn't there.
  if (isAxiosError(error) && error.response?.status === 404) return null;

  const signoff = data?.signoff ?? null;
  const recipients = data?.recipients ?? [];
  const signable = SIGNABLE_STATUSES.has(jobStatus);
  // The phone shows the card once there is something to show; the "Get
  // signature" button lives in its bottom bar.
  if (phone && !signoff && !isLoading && !error) return null;

  const done =
    jobStatus === JobStatus.COMPLETED ||
    jobStatus === JobStatus.INVOICED ||
    jobStatus === JobStatus.PAID;
  const emailed = !!signoff?.report_emailed_at;
  const busy = email.isPending;

  return (
    <section
      aria-labelledby={`signoff-${jobId}`}
      className={cn(
        'rounded-2xl border border-neutral-200 bg-white',
        phone ? 'p-4' : 'p-6'
      )}
    >
      <div className="flex items-center justify-between gap-3">
        <h2
          id={`signoff-${jobId}`}
          className="text-lg font-semibold text-neutral-900"
        >
          Sign-off
        </h2>
        {canEdit && signable && !signoff?.locked && !isLoading && !error && (
          <button
            type="button"
            onClick={() => setSigning(true)}
            className="inline-flex min-h-10 items-center gap-1.5 rounded-xl border-[1.5px] border-primary-600 px-3 text-sm font-semibold text-primary-700 hover:bg-primary-50"
          >
            <PenLine className="h-4 w-4" aria-hidden="true" />
            {signoff ? 'Sign again' : 'Get signature'}
          </button>
        )}
      </div>

      {isLoading ? (
        <p className="mt-3 text-neutral-500">Loading sign-off…</p>
      ) : error ? (
        <p role="alert" className="mt-3 text-sm text-error-600">
          {getErrorMessage(error, "Couldn't load the sign-off")}
        </p>
      ) : !signoff ? (
        <p className="mt-3 text-neutral-500">
          {signable
            ? "Not signed yet. The technician gets the site manager's signature on their phone when the visit is done."
            : 'Not signed yet. The site manager signs once the visit is under way.'}
        </p>
      ) : (
        <div className="mt-4 flex flex-col gap-4 sm:flex-row sm:items-center">
          <div className="flex h-28 w-full shrink-0 items-center justify-center rounded-xl border border-dashed border-neutral-300 bg-white p-2 sm:w-60">
            <img
              src={signoff.signature_url}
              alt={`Signature of ${signoff.signer_name}`}
              className="max-h-full max-w-full object-contain"
            />
          </div>
          <div className="min-w-0">
            <p className="font-semibold text-neutral-900">
              {signerLine(signoff)}
            </p>
            <p className="text-neutral-600">
              Signed
              {signoff.captured_by_name
                ? ` on ${signoff.captured_by_name}'s phone`
                : ''}{' '}
              at {signoffTime(signoff.signed_at, timeZone)}
            </p>
          </div>
        </div>
      )}

      {signoff && (
        <div
          className={cn(
            'mt-4 rounded-xl border p-4',
            phone && done && !emailed
              ? 'border-primary-200 bg-primary-50'
              : 'border-neutral-200'
          )}
        >
          <div className="flex items-start gap-3">
            <span
              className={cn(
                'flex h-9 w-9 shrink-0 items-center justify-center rounded-lg',
                emailed
                  ? 'bg-success-50 text-success-700'
                  : 'bg-primary-50 text-primary-700'
              )}
            >
              <Mail className="h-5 w-5" aria-hidden="true" />
            </span>
            <div className="min-w-0 flex-1">
              {emailed ? (
                <>
                  <p className="font-medium break-words text-neutral-900">
                    Service report emailed to{' '}
                    {signoff.report_recipients.join(', ')}
                  </p>
                  <p className="text-sm text-neutral-600">
                    {signoffTime(signoff.report_emailed_at as string, timeZone)}
                    {signoff.report_email_count > 1
                      ? ` · sent ${signoff.report_email_count} times`
                      : ''}
                  </p>
                </>
              ) : recipients.length > 0 ? (
                <>
                  <p className="font-medium text-neutral-900">
                    {phone && done
                      ? 'Visit complete. Email the service report?'
                      : 'Service report not emailed yet'}
                  </p>
                  <p className="text-sm break-words text-neutral-600">
                    It goes to {recipients.join(', ')}.
                  </p>
                </>
              ) : (
                <>
                  <p className="font-medium text-neutral-900">
                    No email address for the report
                  </p>
                  <p className="text-sm text-neutral-600">
                    Add an email to the customer or the site contact, or sign
                    again with the signer&apos;s email.
                  </p>
                </>
              )}
            </div>
          </div>

          <div
            className={cn(
              'mt-3 flex gap-2',
              phone ? 'flex-col' : 'flex-wrap justify-end'
            )}
          >
            {canEdit && recipients.length > 0 && (
              <button
                type="button"
                onClick={() => {
                  setNotice(null);
                  email.mutate(emailed);
                }}
                disabled={busy}
                className={cn(
                  'inline-flex items-center justify-center gap-2 rounded-xl font-semibold disabled:opacity-60',
                  phone ? 'min-h-12 text-base' : 'min-h-10 px-4 text-sm',
                  emailed
                    ? 'border border-neutral-300 bg-white text-neutral-800 hover:bg-neutral-50'
                    : 'bg-primary-600 text-white hover:bg-primary-700'
                )}
              >
                <Send className="h-4 w-4" aria-hidden="true" />
                {busy ? 'Sending…' : emailed ? 'Resend report' : 'Email report'}
              </button>
            )}
            <button
              type="button"
              onClick={() => download.mutate()}
              disabled={download.isPending}
              className={cn(
                'inline-flex items-center justify-center gap-2 rounded-xl border-[1.5px] border-primary-600 bg-white font-semibold text-primary-700 hover:bg-primary-50 disabled:opacity-60',
                phone ? 'min-h-12 text-base' : 'min-h-10 px-4 text-sm'
              )}
            >
              <Download className="h-4 w-4" aria-hidden="true" />
              {download.isPending ? 'Preparing…' : 'Download PDF'}
            </button>
          </div>
          {notice && (
            <p role="status" className="mt-2 text-sm text-success-700">
              {notice}
            </p>
          )}
          {(email.isError || download.isError) && (
            <p role="alert" className="mt-2 text-sm text-error-600">
              {email.isError
                ? getErrorMessage(email.error, "Couldn't email the report")
                : getErrorMessage(download.error, "Couldn't open the report")}
            </p>
          )}
        </div>
      )}

      {signing && (
        <SignoffCapture
          jobId={jobId}
          subtitle={siteName}
          defaultSignerName={defaultSignerName}
          onClose={() => setSigning(false)}
        />
      )}
    </section>
  );
}
