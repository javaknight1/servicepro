import { useEffect, useState } from 'react';
import { Loader2 } from 'lucide-react';
import { Button, Modal } from '@components/shared';
import {
  invoiceService,
  type ConsolidatedInvoiceLine,
  type ConsolidatedInvoicePreview,
  type Invoice,
} from '@services/invoiceService';
import { fetchAllCustomers } from '@services/customerLookup';
import type { Customer } from '@services/customerService';
import { getErrorMessage } from '@/utils/error';
import { formatDollars, toAmount } from '@/utils/money';

/** YYYY-MM-DD of a local date. */
function isoDate(d: Date): string {
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${d.getFullYear()}-${m}-${day}`;
}

/** The first and last day of last month (local time). */
function lastMonthPeriod(today: Date = new Date()): {
  start: string;
  end: string;
} {
  const start = new Date(today.getFullYear(), today.getMonth() - 1, 1);
  const end = new Date(today.getFullYear(), today.getMonth(), 0);
  return { start: isoDate(start), end: isoDate(end) };
}

function customerLabel(c: Customer): string {
  return c.company_name || `${c.first_name} ${c.last_name}`.trim() || c.email;
}

/** Lines grouped by site, in the order the API returned them. */
function groupBySite(
  lines: ConsolidatedInvoiceLine[]
): { site: string; lines: ConsolidatedInvoiceLine[] }[] {
  const groups: { site: string; lines: ConsolidatedInvoiceLine[] }[] = [];
  for (const line of lines) {
    const site = line.site_name || 'Other work';
    const last = groups[groups.length - 1];
    if (last && last.site === site) last.lines.push(line);
    else groups.push({ site, lines: [line] });
  }
  return groups;
}

interface ConsolidatedInvoiceModalProps {
  isOpen: boolean;
  onClose: () => void;
  /** The customer to invoice; without it the modal shows a customer picker. */
  customerId?: string;
  customerName?: string;
  /** Called with the new draft (open it). */
  onCreated: (invoice: Invoice) => void;
}

/**
 * "Create consolidated invoice" (T188): pick the period (default last
 * month), preview the lines grouped by site with extra work and its
 * approver, then create the draft. Nothing is written until "Create draft".
 */
export function ConsolidatedInvoiceModal({
  isOpen,
  onClose,
  customerId,
  customerName,
  onCreated,
}: ConsolidatedInvoiceModalProps) {
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [selectedCustomer, setSelectedCustomer] = useState(customerId ?? '');
  const [period, setPeriod] = useState(lastMonthPeriod);
  const [preview, setPreview] = useState<ConsolidatedInvoicePreview | null>(
    null
  );
  const [isLoading, setIsLoading] = useState(false);
  const [isCreating, setIsCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Reset when opened.
  useEffect(() => {
    if (!isOpen) return;
    setSelectedCustomer(customerId ?? '');
    setPeriod(lastMonthPeriod());
    setPreview(null);
    setError(null);
  }, [isOpen, customerId]);

  // The picker lists consolidated-billing customers first.
  useEffect(() => {
    if (!isOpen || customerId) return;
    let cancelled = false;
    fetchAllCustomers()
      .then((all) => {
        if (cancelled) return;
        const sorted = [...all].sort(
          (a, b) =>
            Number(Boolean(b.consolidated_billing)) -
              Number(Boolean(a.consolidated_billing)) ||
            customerLabel(a).localeCompare(customerLabel(b))
        );
        setCustomers(sorted);
      })
      .catch((err: unknown) => {
        if (!cancelled)
          setError(getErrorMessage(err, "Couldn't load customers"));
      });
    return () => {
      cancelled = true;
    };
  }, [isOpen, customerId]);

  // Preview whenever the customer or period changes.
  useEffect(() => {
    if (!isOpen || !selectedCustomer || !period.start || !period.end) {
      setPreview(null);
      return;
    }
    let cancelled = false;
    setIsLoading(true);
    setError(null);
    invoiceService
      .previewConsolidatedInvoice({
        customer_id: selectedCustomer,
        period_start: period.start,
        period_end: period.end,
      })
      .then((p) => {
        if (!cancelled) setPreview(p);
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        setPreview(null);
        setError(getErrorMessage(err, "Couldn't preview the invoice"));
      })
      .finally(() => {
        if (!cancelled) setIsLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [isOpen, selectedCustomer, period.start, period.end]);

  const handleCreate = async () => {
    if (!selectedCustomer) return;
    setIsCreating(true);
    setError(null);
    try {
      const invoice = await invoiceService.createConsolidatedInvoice({
        customer_id: selectedCustomer,
        period_start: period.start,
        period_end: period.end,
      });
      onCreated(invoice);
    } catch (err) {
      setError(getErrorMessage(err, "Couldn't create the invoice"));
    } finally {
      setIsCreating(false);
    }
  };

  const hasLines = (preview?.lines.length ?? 0) > 0;

  return (
    <Modal
      isOpen={isOpen}
      onClose={isCreating ? () => {} : onClose}
      size="xl"
      title="Create consolidated invoice"
      description={
        customerName
          ? `One invoice for ${customerName}: a line per site plus approved extra work.`
          : 'One invoice per customer: a line per site plus approved extra work.'
      }
    >
      <div className="space-y-4">
        {!customerId && (
          <div className="flex flex-col">
            <label
              htmlFor="consolidate-customer"
              className="mb-1.5 text-sm font-medium text-neutral-700"
            >
              Customer
            </label>
            <select
              id="consolidate-customer"
              value={selectedCustomer}
              onChange={(e) => setSelectedCustomer(e.target.value)}
              className="rounded-lg border border-neutral-300 px-3 py-2 focus:border-primary-500 focus:ring-2 focus:ring-primary-500"
            >
              <option value="">Select a customer</option>
              {customers.map((c) => (
                <option key={c.id} value={c.id}>
                  {customerLabel(c)}
                  {c.consolidated_billing ? ' · monthly invoice' : ''}
                </option>
              ))}
            </select>
          </div>
        )}

        <div className="grid grid-cols-2 gap-3">
          <div className="flex flex-col">
            <label
              htmlFor="consolidate-start"
              className="mb-1.5 text-sm font-medium text-neutral-700"
            >
              From
            </label>
            <input
              id="consolidate-start"
              type="date"
              value={period.start}
              max={period.end || undefined}
              onChange={(e) => setPeriod({ ...period, start: e.target.value })}
              className="rounded-lg border border-neutral-300 px-3 py-2 focus:border-primary-500 focus:ring-2 focus:ring-primary-500"
            />
          </div>
          <div className="flex flex-col">
            <label
              htmlFor="consolidate-end"
              className="mb-1.5 text-sm font-medium text-neutral-700"
            >
              To
            </label>
            <input
              id="consolidate-end"
              type="date"
              value={period.end}
              min={period.start || undefined}
              onChange={(e) => setPeriod({ ...period, end: e.target.value })}
              className="rounded-lg border border-neutral-300 px-3 py-2 focus:border-primary-500 focus:ring-2 focus:ring-primary-500"
            />
          </div>
        </div>

        {error && (
          <div
            role="alert"
            className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700"
          >
            {error}
          </div>
        )}

        {isLoading && (
          <div className="flex items-center gap-2 text-sm text-neutral-500">
            <Loader2 className="h-4 w-4 animate-spin" /> Loading preview…
          </div>
        )}

        {!isLoading && preview && !hasLines && (
          <p className="rounded-lg bg-neutral-50 p-4 text-sm text-neutral-600">
            Nothing to invoice for this period: no unbilled visits, agreement
            months or completed extra work.
          </p>
        )}

        {!isLoading && preview && hasLines && (
          <div data-testid="consolidated-preview">
            <p className="mb-2 text-sm text-neutral-600">
              {preview.visits} visit{preview.visits === 1 ? '' : 's'}
              {preview.extra_work > 0 &&
                ` + ${preview.extra_work} extra work order${preview.extra_work === 1 ? '' : 's'}`}
              {preview.sites > 0 &&
                ` · ${preview.sites} site${preview.sites === 1 ? '' : 's'}`}
              {preview.po_number && ` · PO ${preview.po_number}`}
            </p>
            <div className="max-h-80 overflow-y-auto rounded-lg border border-neutral-200">
              <table className="w-full text-sm">
                <thead className="bg-neutral-50 text-left text-neutral-600">
                  <tr>
                    <th className="px-3 py-2 font-medium">Site</th>
                    <th className="px-3 py-2 font-medium">Description</th>
                    <th className="px-3 py-2 text-right font-medium">Amount</th>
                  </tr>
                </thead>
                <tbody>
                  {groupBySite(preview.lines).map((group) =>
                    group.lines.map((line, i) => (
                      <tr
                        key={`${group.site}-${line.agreement_id ?? line.job_id}-${line.month ?? ''}`}
                        className="border-t border-neutral-100 align-top"
                      >
                        <td className="px-3 py-2 font-medium text-neutral-900">
                          {i === 0 ? group.site : ''}
                        </td>
                        <td className="px-3 py-2 text-neutral-700">
                          {line.description}
                          {line.approval_note && (
                            <div className="mt-1">
                              <span className="inline-block rounded-full bg-success-50 px-2 py-0.5 text-xs text-success-700">
                                {line.approval_note}
                              </span>
                            </div>
                          )}
                        </td>
                        <td className="px-3 py-2 text-right text-neutral-900">
                          {formatDollars(toAmount(line.amount))}
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
            <dl className="mt-3 ml-auto w-64 space-y-1 text-sm">
              <div className="flex justify-between text-neutral-600">
                <dt>Subtotal</dt>
                <dd>{formatDollars(toAmount(preview.subtotal))}</dd>
              </div>
              {toAmount(preview.tax_amount) > 0 && (
                <div className="flex justify-between text-neutral-600">
                  <dt>Tax</dt>
                  <dd>{formatDollars(toAmount(preview.tax_amount))}</dd>
                </div>
              )}
              <div className="flex justify-between text-base font-semibold text-neutral-900">
                <dt>Total</dt>
                <dd>{formatDollars(toAmount(preview.total))}</dd>
              </div>
            </dl>
          </div>
        )}

        <div className="flex justify-end gap-3 pt-2">
          <Button
            type="button"
            variant="secondary"
            onClick={onClose}
            disabled={isCreating}
          >
            Cancel
          </Button>
          <Button
            type="button"
            variant="primary"
            onClick={handleCreate}
            disabled={!hasLines || isLoading || isCreating}
            className="flex items-center gap-2"
          >
            {isCreating && <Loader2 className="h-4 w-4 animate-spin" />}
            Create draft
          </Button>
        </div>
      </div>
    </Modal>
  );
}
