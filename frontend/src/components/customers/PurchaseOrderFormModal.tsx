import { useEffect, useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Button, Input, Modal } from '@components/shared';
import {
  purchaseOrderService,
  purchaseOrdersQueryKey,
  type PurchaseOrder,
} from '@services/purchaseOrderService';
import { getErrorMessage, getFieldErrors } from '@/utils/error';

interface PurchaseOrderFormModalProps {
  customerId: string;
  /** The PO to edit; omitted to record a new one. */
  purchaseOrder?: PurchaseOrder | null;
  isOpen: boolean;
  onClose: () => void;
}

interface POForm {
  po_number: string;
  amount: string;
  expires_on: string;
  notes: string;
}

function toForm(po: PurchaseOrder | null | undefined): POForm {
  return {
    po_number: po?.po_number ?? '',
    amount: po?.amount ?? '',
    expires_on: po?.expires_on ?? '',
    notes: po?.notes ?? '',
  };
}

/** Record or edit a customer's purchase order and its amount (T187). */
export function PurchaseOrderFormModal({
  customerId,
  purchaseOrder,
  isOpen,
  onClose,
}: PurchaseOrderFormModalProps) {
  const queryClient = useQueryClient();
  const [form, setForm] = useState<POForm>(() => toForm(purchaseOrder));

  const save = useMutation({
    mutationFn: (f: POForm) => {
      const poNumber = f.po_number.trim();
      const amount = f.amount.trim();
      if (purchaseOrder) {
        return purchaseOrderService.update(customerId, purchaseOrder.id, {
          po_number: poNumber,
          amount,
          notes: f.notes.trim(),
          ...(f.expires_on
            ? { expires_on: f.expires_on }
            : { clear_expires_on: true }),
        });
      }
      return purchaseOrderService.create(customerId, {
        po_number: poNumber,
        amount,
        expires_on: f.expires_on || undefined,
        notes: f.notes.trim() || undefined,
      });
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: purchaseOrdersQueryKey(customerId),
      });
      onClose();
    },
  });

  const resetSave = save.reset;
  useEffect(() => {
    if (!isOpen) return;
    setForm(toForm(purchaseOrder));
    resetSave();
  }, [isOpen, purchaseOrder, resetSave]);

  const fieldErrors = save.error ? getFieldErrors(save.error) : {};
  const set =
    (name: keyof POForm) =>
    (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>): void => {
      const { value } = e.target;
      setForm((prev) => ({ ...prev, [name]: value }));
    };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={
        purchaseOrder
          ? `Edit PO ${purchaseOrder.po_number}`
          : 'Add purchase order'
      }
      description="Invoices for this customer with this PO number count against its amount. Sending one that goes over the remaining balance asks you to confirm."
      size="md"
    >
      <form
        onSubmit={(e) => {
          e.preventDefault();
          e.stopPropagation();
          save.mutate(form);
        }}
        className="space-y-4"
      >
        {save.error && Object.keys(fieldErrors).length === 0 && (
          <p
            role="alert"
            className="rounded-lg border border-error-200 bg-error-50 p-3 text-sm text-error-700"
          >
            {getErrorMessage(save.error, 'Could not save the purchase order')}
          </p>
        )}
        <Input
          id="po-number"
          label="PO number *"
          value={form.po_number}
          onChange={set('po_number')}
          required
          maxLength={100}
          placeholder="PO-7731-HV"
          error={fieldErrors.po_number}
          fullWidth
        />
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Input
            id="po-amount"
            label="Authorised amount ($) *"
            type="number"
            inputMode="decimal"
            min="0.01"
            step="0.01"
            value={form.amount}
            onChange={set('amount')}
            required
            error={fieldErrors.amount}
            fullWidth
          />
          <Input
            id="po-expires"
            label="Expires"
            type="date"
            value={form.expires_on}
            onChange={set('expires_on')}
            error={fieldErrors.expires_on}
            fullWidth
          />
        </div>
        <div className="flex flex-col">
          <label
            htmlFor="po-notes"
            className="mb-1.5 text-sm font-medium text-neutral-700"
          >
            Notes
          </label>
          <textarea
            id="po-notes"
            value={form.notes}
            onChange={set('notes')}
            rows={2}
            maxLength={2000}
            placeholder="What the PO covers"
            className="w-full rounded-lg border border-neutral-300 px-3 py-2 focus:border-primary-500 focus:ring-2 focus:ring-primary-500"
          />
        </div>
        <div className="flex justify-end gap-3 pt-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" variant="primary" disabled={save.isPending}>
            {save.isPending
              ? 'Saving...'
              : purchaseOrder
                ? 'Save PO'
                : 'Add PO'}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
