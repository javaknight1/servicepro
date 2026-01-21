import type { BillingScheduleProps, Invoice } from '../../types/subscription';
import {
  formatPrice,
  formatDateShort,
  formatBillingCycle,
} from '../../types/subscription';
import { cn } from '../../utils/cn';

// Invoice status badge
function InvoiceStatusBadge({ status }: { status: Invoice['status'] }) {
  const statusConfig = {
    draft: { label: 'Draft', className: 'bg-neutral-100 text-neutral-600' },
    open: { label: 'Open', className: 'bg-amber-100 text-amber-700' },
    paid: { label: 'Paid', className: 'bg-green-100 text-green-700' },
    uncollectible: {
      label: 'Uncollectible',
      className: 'bg-red-100 text-red-700',
    },
    void: { label: 'Void', className: 'bg-neutral-100 text-neutral-500' },
  };

  const config = statusConfig[status];

  return (
    <span
      className={cn(
        'rounded-full px-2 py-0.5 text-xs font-medium',
        config.className
      )}
    >
      {config.label}
    </span>
  );
}

// Invoice row component
function InvoiceRow({
  invoice,
  onView,
  onDownload,
}: {
  invoice: Invoice;
  onView?: () => void;
  onDownload?: () => void;
}) {
  return (
    <div className="flex items-center justify-between border-b border-neutral-100 py-4 last:border-0">
      <div className="flex items-center gap-4">
        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-neutral-100">
          <svg
            className="h-5 w-5 text-neutral-600"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
            />
          </svg>
        </div>
        <div>
          <p className="text-sm font-medium text-neutral-900">
            {formatDateShort(invoice.periodStart)} -{' '}
            {formatDateShort(invoice.periodEnd)}
          </p>
          <p className="text-sm text-neutral-500">
            Invoice #
            {invoice.stripeInvoiceId?.slice(-8) || invoice.id.slice(-8)}
          </p>
        </div>
      </div>

      <div className="flex items-center gap-4">
        <div className="text-right">
          <p className="text-sm font-medium text-neutral-900">
            {formatPrice(invoice.total, 'USD')}
          </p>
          <InvoiceStatusBadge status={invoice.status} />
        </div>

        <div className="flex items-center gap-2">
          {onView && invoice.hostedInvoiceUrl && (
            <button
              type="button"
              onClick={onView}
              className="rounded-lg p-2 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-600"
              title="View invoice"
            >
              <svg
                className="h-4 w-4"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
                />
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"
                />
              </svg>
            </button>
          )}
          {onDownload && invoice.invoicePdf && (
            <button
              type="button"
              onClick={onDownload}
              className="rounded-lg p-2 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-600"
              title="Download PDF"
            >
              <svg
                className="h-4 w-4"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
                />
              </svg>
            </button>
          )}
        </div>
      </div>
    </div>
  );
}

export function BillingSchedule({
  schedule,
  invoices = [],
  onViewInvoice,
  onDownloadInvoice,
  showPastInvoices = true,
  maxInvoices = 5,
  isLoading = false,
  className,
}: BillingScheduleProps) {
  if (isLoading) {
    return (
      <div className={cn('animate-pulse space-y-4', className)}>
        <div className="h-6 w-32 rounded bg-neutral-200" />
        <div className="h-24 rounded-lg bg-neutral-200" />
        <div className="h-6 w-32 rounded bg-neutral-200" />
        <div className="h-32 rounded-lg bg-neutral-200" />
      </div>
    );
  }

  const displayedInvoices = invoices.slice(0, maxInvoices);

  return (
    <div className={cn('space-y-6', className)}>
      {/* Next billing */}
      <div className="rounded-lg border border-neutral-200 bg-white p-6">
        <h3 className="text-sm font-medium text-neutral-500 uppercase tracking-wide">
          Next Billing
        </h3>
        <div className="mt-4 flex items-end justify-between">
          <div>
            <p className="text-3xl font-bold text-neutral-900">
              {formatPrice(schedule.amountDue, schedule.currency)}
            </p>
            <p className="mt-1 text-sm text-neutral-600">
              {formatBillingCycle(schedule.billingCycle)} subscription
            </p>
          </div>
          <div className="text-right">
            <p className="text-sm text-neutral-500">Due date</p>
            <p className="text-lg font-semibold text-neutral-900">
              {formatDateShort(schedule.nextBillingDate)}
            </p>
          </div>
        </div>
      </div>

      {/* Current period */}
      <div className="rounded-lg border border-neutral-200 bg-white p-6">
        <h3 className="text-sm font-medium text-neutral-500 uppercase tracking-wide">
          Current Period
        </h3>
        <div className="mt-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="h-3 w-3 rounded-full bg-green-500" />
              <span className="text-sm font-medium text-neutral-900">
                Active
              </span>
            </div>
            <span className="text-sm text-neutral-600">
              {formatDateShort(schedule.currentPeriod.startDate)} -{' '}
              {formatDateShort(schedule.currentPeriod.endDate)}
            </span>
          </div>

          {/* Progress bar */}
          <div className="mt-4">
            {(() => {
              const start = new Date(
                schedule.currentPeriod.startDate
              ).getTime();
              const end = new Date(schedule.currentPeriod.endDate).getTime();
              const now = Date.now();
              const progress = Math.min(
                100,
                Math.max(0, ((now - start) / (end - start)) * 100)
              );

              return (
                <div className="h-2 overflow-hidden rounded-full bg-neutral-100">
                  <div
                    className="h-full rounded-full bg-primary-500 transition-all"
                    style={{ width: `${progress}%` }}
                  />
                </div>
              );
            })()}
            <div className="mt-2 flex justify-between text-xs text-neutral-500">
              <span>
                Started {formatDateShort(schedule.currentPeriod.startDate)}
              </span>
              <span>
                Ends {formatDateShort(schedule.currentPeriod.endDate)}
              </span>
            </div>
          </div>
        </div>
      </div>

      {/* Upcoming periods */}
      {schedule.upcomingPeriods.length > 0 && (
        <div className="rounded-lg border border-neutral-200 bg-white p-6">
          <h3 className="text-sm font-medium text-neutral-500 uppercase tracking-wide">
            Upcoming Billing
          </h3>
          <div className="mt-4 space-y-3">
            {schedule.upcomingPeriods.map((period, index) => (
              <div
                key={index}
                className="flex items-center justify-between rounded-lg bg-neutral-50 px-4 py-3"
              >
                <div className="flex items-center gap-3">
                  <svg
                    className="h-5 w-5 text-neutral-400"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"
                    />
                  </svg>
                  <span className="text-sm text-neutral-700">
                    {formatDateShort(period.startDate)} -{' '}
                    {formatDateShort(period.endDate)}
                  </span>
                </div>
                <span className="text-sm font-medium text-neutral-900">
                  {formatPrice(period.amount, schedule.currency)}
                </span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Past invoices */}
      {showPastInvoices && displayedInvoices.length > 0 && (
        <div className="rounded-lg border border-neutral-200 bg-white p-6">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-medium text-neutral-500 uppercase tracking-wide">
              Invoice History
            </h3>
            {invoices.length > maxInvoices && (
              <button
                type="button"
                className="text-sm font-medium text-primary-600 hover:text-primary-700"
              >
                View all ({invoices.length})
              </button>
            )}
          </div>
          <div className="mt-4">
            {displayedInvoices.map((invoice) => (
              <InvoiceRow
                key={invoice.id}
                invoice={invoice}
                onView={
                  onViewInvoice ? () => onViewInvoice(invoice) : undefined
                }
                onDownload={
                  onDownloadInvoice
                    ? () => onDownloadInvoice(invoice)
                    : undefined
                }
              />
            ))}
          </div>
        </div>
      )}

      {/* Empty invoices state */}
      {showPastInvoices && invoices.length === 0 && (
        <div className="rounded-lg border border-neutral-200 bg-white p-6">
          <h3 className="text-sm font-medium text-neutral-500 uppercase tracking-wide">
            Invoice History
          </h3>
          <div className="mt-4 text-center py-8">
            <svg
              className="mx-auto h-12 w-12 text-neutral-300"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
              />
            </svg>
            <p className="mt-4 text-sm text-neutral-500">No invoices yet</p>
          </div>
        </div>
      )}

      {/* Payment method reminder */}
      <div className="rounded-lg bg-neutral-50 px-4 py-3 text-sm text-neutral-600">
        <div className="flex items-center gap-2">
          <svg
            className="h-5 w-5 text-neutral-400"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
            />
          </svg>
          <span>
            Payments are automatically charged to your default payment method on
            file.
          </span>
        </div>
      </div>
    </div>
  );
}
