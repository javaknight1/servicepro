import { useEffect, useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { Button, Input, Modal } from '@components/shared';
import {
  customerService,
  type Customer,
  type CustomerBillingRulesUpdate,
} from '@services/customerService';
import { getErrorMessage, getFieldErrors } from '@/utils/error';

interface BillingRulesModalProps {
  customer: Customer;
  isOpen: boolean;
  onClose: () => void;
  onSaved: (customer: Customer) => void;
}

interface RulesForm {
  consolidated_billing: boolean;
  po_required: boolean;
  default_po_number: string;
  billing_approver_name: string;
  billing_approver_email: string;
  nte_limit: string;
}

function toForm(c: Customer): RulesForm {
  return {
    consolidated_billing: !!c.consolidated_billing,
    po_required: !!c.po_required,
    default_po_number: c.default_po_number ?? '',
    billing_approver_name: c.billing_approver_name ?? '',
    billing_approver_email: c.billing_approver_email ?? '',
    nte_limit: c.nte_limit ?? '',
  };
}

/** The update body: every rule is sent, "" clears text, no limit clears NTE. */
function toBillingRulesUpdate(form: RulesForm): CustomerBillingRulesUpdate {
  const nte = form.nte_limit.trim();
  return {
    consolidated_billing: form.consolidated_billing,
    po_required: form.po_required,
    default_po_number: form.default_po_number.trim(),
    billing_approver_name: form.billing_approver_name.trim(),
    billing_approver_email: form.billing_approver_email.trim(),
    ...(nte === '' ? { clear_nte_limit: true } : { nte_limit: nte }),
  };
}

const checkboxClass =
  'mt-0.5 h-4 w-4 rounded-sm border-neutral-300 text-primary-600 focus:ring-primary-500';

/** Edit a customer's billing rules (T187): POs, approver, not-to-exceed. */
export function BillingRulesModal({
  customer,
  isOpen,
  onClose,
  onSaved,
}: BillingRulesModalProps) {
  const [form, setForm] = useState<RulesForm>(() => toForm(customer));

  const save = useMutation({
    mutationFn: (data: RulesForm) =>
      customerService.updateCustomer(customer.id, toBillingRulesUpdate(data)),
    onSuccess: (updated) => {
      onSaved(updated);
      onClose();
    },
  });

  const resetSave = save.reset;
  useEffect(() => {
    if (!isOpen) return;
    setForm(toForm(customer));
    resetSave();
  }, [isOpen, customer, resetSave]);

  const fieldErrors = save.error ? getFieldErrors(save.error) : {};
  const setText =
    (name: keyof RulesForm) => (e: React.ChangeEvent<HTMLInputElement>) => {
      const { value } = e.target;
      setForm((prev) => ({ ...prev, [name]: value }));
    };

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title="Billing rules"
      description="How this customer wants to be billed: purchase orders, who approves extra work and how much it can cost without asking."
      size="xl"
    >
      <form
        onSubmit={(e) => {
          e.preventDefault();
          save.mutate(form);
        }}
        className="space-y-5"
      >
        {save.error && (
          <p
            role="alert"
            className="rounded-lg border border-error-200 bg-error-50 p-3 text-sm text-error-700"
          >
            {getErrorMessage(save.error, 'Could not save the billing rules')}
          </p>
        )}

        <label className="flex cursor-pointer items-start gap-3">
          <input
            type="checkbox"
            checked={form.consolidated_billing}
            onChange={(e) =>
              setForm((p) => ({
                ...p,
                consolidated_billing: e.target.checked,
              }))
            }
            className={checkboxClass}
          />
          <span>
            <span className="block text-sm font-medium text-neutral-800">
              Consolidated monthly billing
            </span>
            <span className="block text-sm text-neutral-500">
              One invoice a month, a line per site.
            </span>
          </span>
        </label>

        <div className="space-y-3 rounded-xl border border-neutral-200 p-4">
          <label className="flex cursor-pointer items-start gap-3">
            <input
              type="checkbox"
              checked={form.po_required}
              onChange={(e) =>
                setForm((p) => ({ ...p, po_required: e.target.checked }))
              }
              className={checkboxClass}
            />
            <span>
              <span className="block text-sm font-medium text-neutral-800">
                PO required
              </span>
              <span className="block text-sm text-neutral-500">
                Invoices can't be sent without a purchase order number.
              </span>
            </span>
          </label>
          <Input
            id="default_po_number"
            label="Current PO"
            value={form.default_po_number}
            onChange={setText('default_po_number')}
            maxLength={100}
            placeholder="7731-HV"
            helperText="Filled in on new invoices that have no PO."
            error={fieldErrors.default_po_number}
            fullWidth
          />
        </div>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <Input
            id="billing_approver_name"
            label="Approver name"
            value={form.billing_approver_name}
            onChange={setText('billing_approver_name')}
            maxLength={100}
            error={fieldErrors.billing_approver_name}
            fullWidth
          />
          <Input
            id="billing_approver_email"
            label="Approver email"
            type="email"
            value={form.billing_approver_email}
            onChange={setText('billing_approver_email')}
            maxLength={255}
            error={fieldErrors.billing_approver_email}
            fullWidth
          />
        </div>

        <Input
          id="nte_limit"
          label="Extra work limit (not-to-exceed, $)"
          type="number"
          inputMode="decimal"
          min="0"
          step="0.01"
          value={form.nte_limit}
          onChange={setText('nte_limit')}
          placeholder="No limit"
          helperText="Extra work estimated above this can't be scheduled until the approver says OK. Leave empty for no limit."
          error={fieldErrors.nte_limit}
          fullWidth
        />

        <div className="flex justify-end gap-3 pt-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" isLoading={save.isPending}>
            Save rules
          </Button>
        </div>
      </form>
    </Modal>
  );
}
