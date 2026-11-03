import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Button, Input, Modal } from '@components/shared';
import {
  jobService,
  jobApprovalsQueryKey,
  type RecordJobApprovalRequest,
} from '@services/jobService';
import { getErrorMessage, getFieldErrors } from '@/utils/error';

interface RecordApprovalModalProps {
  jobId: string;
  isOpen: boolean;
  onClose: () => void;
  /** Called after the approval is saved (e.g. to retry scheduling). */
  onApproved: () => void;
  /** Why the dialog opened, e.g. the API's approval_required message. */
  reason?: string;
}

/**
 * "Record the approver's OK" (T187): office staff record that the
 * customer's approver agreed to extra work over the not-to-exceed limit.
 * Empty fields default server-side to the customer's billing approver and
 * the job's estimated cost.
 */
export function RecordApprovalModal({
  jobId,
  isOpen,
  onClose,
  onApproved,
  reason,
}: RecordApprovalModalProps) {
  const queryClient = useQueryClient();
  const [approverName, setApproverName] = useState('');
  const [approverEmail, setApproverEmail] = useState('');
  const [amount, setAmount] = useState('');
  const [notes, setNotes] = useState('');

  const record = useMutation({
    mutationFn: () => {
      const body: RecordJobApprovalRequest = {};
      if (approverName.trim()) body.approver_name = approverName.trim();
      if (approverEmail.trim()) body.approver_email = approverEmail.trim();
      if (amount.trim()) body.amount = amount.trim();
      if (notes.trim()) body.notes = notes.trim();
      return jobService.recordApproval(jobId, body);
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: jobApprovalsQueryKey(jobId),
      });
      onApproved();
    },
  });

  const fieldErrors = record.error ? getFieldErrors(record.error) : {};

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title="Record the approver's OK"
      description={
        reason ??
        "Log that the customer's approver agreed to this extra work and its cost."
      }
      size="lg"
    >
      <form
        onSubmit={(e) => {
          e.preventDefault();
          e.stopPropagation();
          record.mutate();
        }}
        className="space-y-4"
      >
        {record.error && Object.keys(fieldErrors).length === 0 && (
          <p
            role="alert"
            className="rounded-lg border border-error-200 bg-error-50 p-3 text-sm text-error-700"
          >
            {getErrorMessage(record.error, 'Could not record the approval')}
          </p>
        )}
        <p className="text-sm text-neutral-500">
          Leave a field empty to use the customer's approver and the job's
          estimated cost.
        </p>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Input
            id="approval-name"
            label="Approver name"
            value={approverName}
            onChange={(e) => setApproverName(e.target.value)}
            maxLength={100}
            error={fieldErrors.approver_name}
            fullWidth
          />
          <Input
            id="approval-email"
            label="Approver email"
            type="email"
            value={approverEmail}
            onChange={(e) => setApproverEmail(e.target.value)}
            maxLength={255}
            error={fieldErrors.approver_email}
            fullWidth
          />
        </div>
        <Input
          id="approval-amount"
          label="Approved amount ($)"
          type="number"
          inputMode="decimal"
          min="0"
          step="0.01"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
          placeholder="Job's estimated cost"
          error={fieldErrors.amount}
          fullWidth
        />
        <div className="flex flex-col">
          <label
            htmlFor="approval-notes"
            className="mb-1.5 text-sm font-medium text-neutral-700"
          >
            Notes
          </label>
          <textarea
            id="approval-notes"
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            rows={2}
            maxLength={2000}
            placeholder="e.g. OK'd by phone at 2:15 PM"
            className="w-full rounded-lg border border-neutral-300 px-3 py-2 focus:border-primary-500 focus:ring-2 focus:ring-primary-500"
          />
        </div>
        <div className="flex justify-end gap-3 pt-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" isLoading={record.isPending}>
            Record approval
          </Button>
        </div>
      </form>
    </Modal>
  );
}
