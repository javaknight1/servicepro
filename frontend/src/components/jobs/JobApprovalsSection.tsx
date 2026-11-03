import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { CheckCircle2, Loader2, Plus } from 'lucide-react';
import { Button } from '@components/shared';
import {
  jobService,
  jobApprovalsQueryKey,
  type Job,
} from '@services/jobService';
import { formatDollars } from '@/utils/money';
import { getErrorMessage } from '@/utils/error';
import { RecordApprovalModal } from './RecordApprovalModal';

interface JobApprovalsSectionProps {
  job: Job;
}

/** The customer approver's OKs recorded for a job's extra work (T187). */
export function JobApprovalsSection({ job }: JobApprovalsSectionProps) {
  const [isRecording, setIsRecording] = useState(false);
  const approvalsQuery = useQuery({
    queryKey: jobApprovalsQueryKey(job.id),
    queryFn: () => jobService.getApprovals(job.id),
  });
  const approvals = approvalsQuery.data ?? [];

  return (
    <section
      aria-label="Approvals"
      className="rounded-2xl border border-neutral-200 bg-white p-6"
    >
      <div className="mb-4 flex items-center justify-between gap-4">
        <div>
          <h2 className="text-lg font-semibold text-neutral-900">Approvals</h2>
          <p className="text-sm text-neutral-500">
            {job.agreement_id
              ? 'A visit from a service agreement: no approval needed.'
              : "Extra work over the customer's not-to-exceed limit needs the approver's OK before it's scheduled."}
          </p>
        </div>
        {!job.agreement_id && (
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => setIsRecording(true)}
            className="shrink-0 gap-1.5"
          >
            <Plus className="h-4 w-4" />
            Record approval
          </Button>
        )}
      </div>
      {approvalsQuery.isPending ? (
        <Loader2 className="h-5 w-5 animate-spin text-neutral-400" />
      ) : approvalsQuery.isError ? (
        <p className="text-sm text-error-700">
          {getErrorMessage(approvalsQuery.error, 'Could not load approvals')}
        </p>
      ) : approvals.length === 0 ? (
        <p className="text-sm text-neutral-500">No approvals recorded.</p>
      ) : (
        <ul className="divide-y divide-neutral-200">
          {approvals.map((a) => (
            <li key={a.id} className="flex items-start gap-3 py-3">
              <CheckCircle2 className="mt-0.5 h-5 w-5 shrink-0 text-success-600" />
              <div className="min-w-0 flex-1">
                <p className="text-[15px] text-neutral-900">
                  <span className="font-medium">
                    {a.approver_name || a.approver_email}
                  </span>{' '}
                  approved up to{' '}
                  <span className="font-semibold">
                    {formatDollars(a.amount)}
                  </span>
                </p>
                <p className="text-sm text-neutral-500">
                  {new Date(a.approved_at).toLocaleString('en-US', {
                    dateStyle: 'medium',
                    timeStyle: 'short',
                  })}
                  {a.approver_name ? ` · ${a.approver_email}` : ''}
                  {a.notes ? ` · ${a.notes}` : ''}
                </p>
              </div>
            </li>
          ))}
        </ul>
      )}
      {isRecording && (
        <RecordApprovalModal
          jobId={job.id}
          isOpen
          onClose={() => setIsRecording(false)}
          onApproved={() => setIsRecording(false)}
        />
      )}
    </section>
  );
}
