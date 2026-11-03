import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Loader2, Pencil, Plus, Trash2 } from 'lucide-react';
import { Badge, Button } from '@components/shared';
import {
  purchaseOrderService,
  purchaseOrdersQueryKey,
  type PurchaseOrder,
} from '@services/purchaseOrderService';
import { getErrorMessage } from '@/utils/error';
import { formatDollars, toAmount } from '@/utils/money';
import { cn } from '@utils/cn';
import { PurchaseOrderFormModal } from './PurchaseOrderFormModal';

interface CustomerPurchaseOrdersSectionProps {
  customerId: string;
}

/** "Jun 30, 2027" from a YYYY-MM-DD date (no time zone shift). */
function formatDay(ymd: string): string {
  const [y, m, d] = ymd.split('-').map(Number);
  return new Date(Date.UTC(y, m - 1, d)).toLocaleDateString('en-US', {
    timeZone: 'UTC',
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  });
}

/** The PO's state: over its amount, expired, nearly used, or open. */
function poBadge(po: PurchaseOrder): {
  label: string;
  variant: 'success' | 'warning' | 'error' | 'neutral';
} {
  const remaining = toAmount(po.remaining);
  if (remaining < 0) return { label: 'Over', variant: 'error' };
  if (po.expired) return { label: 'Expired', variant: 'neutral' };
  if (remaining === 0) return { label: 'Used up', variant: 'neutral' };
  if (remaining < toAmount(po.amount) * 0.1)
    return { label: 'Nearly used', variant: 'warning' };
  return { label: 'Open', variant: 'success' };
}

/**
 * The customer's purchase orders (T187): each PO's authorised amount, what
 * issued invoices have billed against it and what remains.
 */
export function CustomerPurchaseOrdersSection({
  customerId,
}: CustomerPurchaseOrdersSectionProps) {
  const queryClient = useQueryClient();
  const [editing, setEditing] = useState<PurchaseOrder | null>(null);
  const [isAddOpen, setIsAddOpen] = useState(false);

  const posQuery = useQuery({
    queryKey: purchaseOrdersQueryKey(customerId),
    queryFn: () => purchaseOrderService.list(customerId),
  });

  const remove = useMutation({
    mutationFn: (po: PurchaseOrder) =>
      purchaseOrderService.remove(customerId, po.id),
    onSuccess: () =>
      queryClient.invalidateQueries({
        queryKey: purchaseOrdersQueryKey(customerId),
      }),
  });

  const handleRemove = (po: PurchaseOrder) => {
    if (
      !window.confirm(
        `Stop tracking PO ${po.po_number}? Invoices keep the PO number; they just aren't checked against its amount.`
      )
    ) {
      return;
    }
    remove.mutate(po);
  };

  const pos = posQuery.data ?? [];

  return (
    <section
      aria-label="Purchase orders"
      className="overflow-hidden rounded-2xl border border-neutral-200 bg-white"
    >
      {remove.error && (
        <p
          role="alert"
          className="border-b border-error-200 bg-error-50 px-6 py-3 text-sm text-error-700"
        >
          {getErrorMessage(remove.error, 'Could not remove the purchase order')}
        </p>
      )}
      {posQuery.isPending ? (
        <div className="flex justify-center py-10">
          <Loader2 className="h-6 w-6 animate-spin text-neutral-400" />
        </div>
      ) : posQuery.isError ? (
        <p
          role="alert"
          className="px-6 py-8 text-center text-sm text-error-700"
        >
          {getErrorMessage(posQuery.error, 'Could not load purchase orders')}
        </p>
      ) : pos.length === 0 ? (
        <div className="px-6 py-10 text-center">
          <p className="font-medium text-neutral-900">No purchase orders</p>
          <p className="mt-1 text-sm text-neutral-500">
            Add the customer's PO and its amount to see what's left as you
            invoice against it.
          </p>
          <Button
            variant="outline"
            size="sm"
            className="mt-4"
            onClick={() => setIsAddOpen(true)}
          >
            Add purchase order
          </Button>
        </div>
      ) : (
        <>
          <div className="overflow-x-auto">
            <table className="w-full min-w-[760px] text-left text-[15px]">
              <thead className="border-b border-neutral-200 bg-neutral-50 text-sm font-semibold text-neutral-700">
                <tr>
                  <th className="px-6 py-3">PO number</th>
                  <th className="px-4 py-3 text-right">Authorised</th>
                  <th className="px-4 py-3 text-right">Billed</th>
                  <th className="px-4 py-3 text-right">Remaining</th>
                  <th className="px-4 py-3">Expires</th>
                  <th className="px-4 py-3">Status</th>
                  <th className="px-6 py-3">
                    <span className="sr-only">Actions</span>
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-200">
                {pos.map((po) => {
                  const badge = poBadge(po);
                  const remaining = toAmount(po.remaining);
                  return (
                    <tr key={po.id} className="align-top">
                      <td className="px-6 py-4">
                        <p className="font-semibold text-neutral-900">
                          {po.po_number}
                        </p>
                        {po.notes && (
                          <p className="mt-0.5 text-sm text-neutral-500">
                            {po.notes}
                          </p>
                        )}
                      </td>
                      <td className="px-4 py-4 text-right tabular-nums">
                        {formatDollars(po.amount)}
                      </td>
                      <td className="px-4 py-4 text-right tabular-nums">
                        {formatDollars(po.billed)}
                        <p className="text-xs text-neutral-500">
                          {po.invoice_count}{' '}
                          {po.invoice_count === 1 ? 'invoice' : 'invoices'}
                        </p>
                      </td>
                      <td
                        className={cn(
                          'px-4 py-4 text-right font-semibold tabular-nums',
                          remaining < 0 ? 'text-error-700' : 'text-neutral-900'
                        )}
                      >
                        {formatDollars(po.remaining)}
                      </td>
                      <td className="px-4 py-4 text-neutral-700">
                        {po.expires_on ? formatDay(po.expires_on) : '—'}
                      </td>
                      <td className="px-4 py-4">
                        <Badge size="sm" variant={badge.variant}>
                          {badge.label}
                        </Badge>
                      </td>
                      <td className="px-6 py-4">
                        <div className="flex justify-end gap-1 whitespace-nowrap">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => setEditing(po)}
                            aria-label={`Edit PO ${po.po_number}`}
                            className="gap-1"
                          >
                            <Pencil className="h-3.5 w-3.5" />
                            Edit
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleRemove(po)}
                            disabled={remove.isPending}
                            aria-label={`Remove PO ${po.po_number}`}
                          >
                            <Trash2 className="h-3.5 w-3.5" />
                          </Button>
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
          <footer className="flex justify-end border-t border-neutral-200 bg-neutral-50/60 px-6 py-3">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setIsAddOpen(true)}
              className="gap-1.5 text-primary-600"
            >
              <Plus className="h-3.5 w-3.5" />
              Add purchase order
            </Button>
          </footer>
        </>
      )}
      <PurchaseOrderFormModal
        customerId={customerId}
        purchaseOrder={editing}
        isOpen={isAddOpen || !!editing}
        onClose={() => {
          setIsAddOpen(false);
          setEditing(null);
        }}
      />
    </section>
  );
}
