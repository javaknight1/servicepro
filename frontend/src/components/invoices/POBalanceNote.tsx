import type { PurchaseOrder } from '@services/purchaseOrderService';
import { formatDollars, roundToCents, toAmount } from '@/utils/money';
import { cn } from '@utils/cn';

interface POBalanceNoteProps {
  po: PurchaseOrder;
  /** A draft's total: shown as what would be left once it is sent. Omit
   * for an issued invoice (already counted in the PO's billed amount). */
  invoiceTotal?: number;
}

/** The tracked PO's remaining balance under the invoice's PO field (T187). */
export function POBalanceNote({ po, invoiceTotal }: POBalanceNoteProps) {
  const remaining = toAmount(po.remaining);
  const after =
    invoiceTotal === undefined
      ? undefined
      : roundToCents(remaining - invoiceTotal);
  const over = (after ?? remaining) < 0;
  return (
    <div
      data-testid="po-balance"
      className={cn(
        'mt-2 rounded-lg border px-3 py-2 text-xs',
        over || po.expired
          ? 'border-warning-200 bg-warning-50 text-warning-800'
          : 'border-neutral-200 bg-neutral-50 text-neutral-600'
      )}
    >
      <p>
        <span className="font-semibold">
          {formatDollars(po.remaining)} of {formatDollars(po.amount)}
        </span>{' '}
        left on PO {po.po_number}
        {po.expired && po.expires_on ? ` · expired ${po.expires_on}` : ''}
      </p>
      {after !== undefined && (
        <p className="mt-0.5">
          After this invoice:{' '}
          <span className="font-semibold">{formatDollars(after)}</span>
          {after < 0 ? ' (over the PO)' : ''}
        </p>
      )}
    </div>
  );
}
