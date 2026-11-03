import { useState, useEffect, useCallback } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Download, X } from 'lucide-react';
import { DashboardLayout } from '@components/layout';
import { Button, PageHeader } from '@components/shared';
import {
  AGING_BUCKETS,
  AGING_BUCKET_STYLES,
} from '@components/reports/agingStyles';
import { arAgingService } from '../../services/arAgingService';
import { customerService, Customer } from '../../services/customerService';
import {
  ARAgingQuery,
  ARAgingBucketSummary,
  ARAgingInvoice,
  ARAgingCustomerSummary,
  AgingBucket,
} from '../../types/arAging';
import { cn } from '../../utils/cn';
import { formatDollars } from '@/utils/money';

const formatMoney = (value: string | number): string =>
  formatDollars(value, { wholeUnits: true });

// Parse decimal string to number
const toNumber = (value: string | number): number => {
  if (typeof value === 'number') return value;
  return parseFloat(value) || 0;
};

// Format date
const formatDate = (dateString: string): string => {
  return new Date(dateString).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    timeZone: 'UTC',
  });
};

const BREADCRUMBS = [{ label: 'Reports' }, { label: 'A/R aging' }];

/** A customer's amount in each bucket, youngest first. */
function customerBuckets(c: ARAgingCustomerSummary): [AgingBucket, number][] {
  return [
    ['current', toNumber(c.current)],
    ['1-30', toNumber(c.days_1_to_30)],
    ['31-60', toNumber(c.days_31_to_60)],
    ['61-90', toNumber(c.days_61_to_90)],
    ['90+', toNumber(c.days_90_plus)],
  ];
}

/** The oldest bucket a customer still owes in (null when the API sent no split). */
function oldestBucket(c: ARAgingCustomerSummary): AgingBucket | null {
  const owing = customerBuckets(c).filter(([, amount]) => amount > 0);
  return owing.length > 0 ? owing[owing.length - 1][0] : null;
}

/** A real date, or null for a missing / zero ("0001-01-01") timestamp. */
function realDate(value?: string): string | null {
  if (!value) return null;
  const year = new Date(value).getUTCFullYear();
  return year > 1970 ? value : null;
}

export function ARAgingReportPage() {
  const [query, setQuery] = useState<ARAgingQuery>({});
  const [selectedBucket, setSelectedBucket] = useState<AgingBucket | null>(
    null
  );
  const [isExporting, setIsExporting] = useState(false);
  const [customers, setCustomers] = useState<Customer[]>([]);

  // Fetch customers for filter dropdown
  useEffect(() => {
    customerService
      .getCustomers({ page_size: 500, sort_by: 'last_name', sort_order: 'asc' })
      .then((res) => setCustomers(res.customers || []))
      .catch(() => {});
  }, []);

  const handleCustomerChange = (customerId: string) => {
    setQuery((prev) => ({
      ...prev,
      customer_id: customerId || undefined,
    }));
    setSelectedBucket(null);
  };

  const selectedCustomer = customers.find((c) => c.id === query.customer_id);

  // Fetch the main report
  const {
    data: report,
    isLoading,
    error,
  } = useQuery({
    queryKey: ['ar-aging-report', query],
    queryFn: () => arAgingService.getARAgingReport(query),
  });

  // Fetch invoices for selected bucket
  const { data: bucketInvoices, isLoading: isLoadingInvoices } = useQuery({
    queryKey: ['ar-aging-invoices', selectedBucket, query],
    queryFn: () =>
      selectedBucket
        ? arAgingService.getInvoicesByBucket(selectedBucket, query)
        : Promise.resolve([]),
    enabled: !!selectedBucket,
  });

  // Handle export
  const handleExport = useCallback(async () => {
    setIsExporting(true);
    try {
      await arAgingService.downloadCSVExport(query);
    } catch (err) {
      console.error('Export failed:', err);
    } finally {
      setIsExporting(false);
    }
  }, [query]);

  // Close bucket detail on escape key
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setSelectedBucket(null);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);

  if (isLoading) {
    return (
      <DashboardLayout breadcrumbs={BREADCRUMBS}>
        <div className="p-4 sm:p-6 lg:p-8">
          <div className="animate-pulse">
            <div className="h-8 bg-gray-200 rounded-sm w-1/4 mb-6" />
            <div className="grid grid-cols-4 gap-4 mb-8">
              {[...Array(4)].map((_, i) => (
                <div key={i} className="h-24 bg-gray-200 rounded-sm" />
              ))}
            </div>
            <div className="h-64 bg-gray-200 rounded-sm" />
          </div>
        </div>
      </DashboardLayout>
    );
  }

  if (error) {
    return (
      <DashboardLayout breadcrumbs={BREADCRUMBS}>
        <div className="p-4 sm:p-6 lg:p-8">
          <div className="bg-red-50 border border-red-200 rounded-lg p-4 text-red-700">
            Failed to load A/R aging report. Please try again.
          </div>
        </div>
      </DashboardLayout>
    );
  }

  const { summary, buckets } = report || { summary: null, buckets: [] };

  return (
    <DashboardLayout breadcrumbs={BREADCRUMBS}>
      <div className="mx-auto max-w-[1440px] p-4 sm:p-6 lg:p-8">
        <PageHeader
          title="A/R aging"
          subtitle={
            <>
              As of{' '}
              {report?.metadata.as_of_date
                ? formatDate(report.metadata.as_of_date)
                : 'today'}{' '}
              · days past due date
              {selectedCustomer && (
                <>
                  {' '}
                  · for{' '}
                  <span className="font-medium text-neutral-800">
                    {selectedCustomer.company_name ||
                      `${selectedCustomer.first_name} ${selectedCustomer.last_name}`}
                  </span>
                </>
              )}
            </>
          }
          actions={
            <>
              <select
                value={query.customer_id || ''}
                onChange={(e) => handleCustomerChange(e.target.value)}
                aria-label="Customer"
                className="rounded-lg border border-neutral-300 bg-white px-3 py-2 text-[15px] focus:border-primary-500 focus:ring-2 focus:ring-primary-500"
              >
                <option value="">All Customers</option>
                {customers.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.company_name
                      ? `${c.company_name} (${c.first_name} ${c.last_name})`
                      : `${c.first_name} ${c.last_name}`}
                  </option>
                ))}
              </select>
              <Button
                variant="secondary"
                onClick={handleExport}
                disabled={isExporting}
                className="gap-2"
              >
                <Download className="h-4 w-4" />
                {isExporting ? 'Exporting...' : 'Export CSV'}
              </Button>
            </>
          }
        />

        {/* Outstanding, the aging bar and one column per bucket (mockup 11) */}
        <section className="mb-6 rounded-2xl border border-neutral-200 bg-white p-6">
          <div className="flex flex-wrap items-start justify-between gap-4">
            <div>
              <p className="text-[15px] text-neutral-600">Outstanding</p>
              <p className="mt-1 text-3xl font-bold tracking-tight text-neutral-900 tabular-nums">
                {formatMoney(summary?.total_outstanding ?? 0)}
              </p>
            </div>
            {summary && (
              <p className="text-[15px] text-neutral-500">
                {summary.customer_count}{' '}
                {summary.customer_count === 1 ? 'customer' : 'customers'} ·{' '}
                {summary.invoice_count} open{' '}
                {summary.invoice_count === 1 ? 'invoice' : 'invoices'}
              </p>
            )}
          </div>

          {summary && toNumber(summary.total_outstanding) > 0 && (
            <div className="mt-5 flex h-3 gap-0.5 overflow-hidden rounded-full bg-neutral-100">
              {buckets.map((bucket: ARAgingBucketSummary) =>
                toNumber(bucket.total_amount) > 0 ? (
                  <div
                    key={bucket.bucket}
                    className={cn(
                      'h-full',
                      AGING_BUCKET_STYLES[bucket.bucket].bar
                    )}
                    style={{ width: `${toNumber(bucket.percentage)}%` }}
                    title={`${bucket.display_name}: ${formatMoney(bucket.total_amount)}`}
                  />
                ) : null
              )}
            </div>
          )}

          <div className="mt-5 grid grid-cols-2 gap-4 md:grid-cols-5">
            {AGING_BUCKETS.map((key) => {
              const bucket = buckets.find(
                (b: ARAgingBucketSummary) => b.bucket === key
              );
              const style = AGING_BUCKET_STYLES[key];
              return (
                <button
                  key={key}
                  type="button"
                  onClick={() => setSelectedBucket(key)}
                  className={cn(
                    'rounded-r-lg border-l-[3px] py-1 pl-4 text-left transition-colors hover:bg-neutral-50',
                    style.border
                  )}
                  title="See the invoices in this bucket"
                >
                  <p className="text-[15px] text-neutral-700">{style.label}</p>
                  <p className="mt-0.5 text-xl font-semibold text-neutral-900 tabular-nums">
                    {formatMoney(bucket?.total_amount ?? 0)}
                  </p>
                  <p className="text-sm text-neutral-500">
                    {toNumber(bucket?.percentage ?? 0).toFixed(1)}% ·{' '}
                    {bucket?.invoice_count ?? 0} invoice
                    {(bucket?.invoice_count ?? 0) !== 1 ? 's' : ''}
                  </p>
                </button>
              );
            })}
          </div>
        </section>

        {/* Who owes what */}
        <section className="overflow-hidden rounded-2xl border border-neutral-200 bg-white">
          <div className="overflow-x-auto">
            <table className="w-full text-left">
              <thead className="border-b border-neutral-200 bg-neutral-50">
                <tr className="text-sm font-medium text-neutral-900">
                  <th className="px-5 py-3.5 font-medium">Customer</th>
                  <th className="px-5 py-3.5 font-medium">Oldest open</th>
                  <th className="px-5 py-3.5 text-right font-medium">
                    Balance
                  </th>
                  <th className="px-5 py-3.5 text-right font-medium">
                    Invoices
                  </th>
                  <th className="px-5 py-3.5 font-medium">Bucket</th>
                  <th className="px-5 py-3.5 font-medium">Last payment</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-neutral-200 text-[15px]">
                {(report?.customer_summary ?? []).length === 0 ? (
                  <tr>
                    <td
                      colSpan={6}
                      className="px-5 py-10 text-center text-neutral-500"
                    >
                      Nobody owes you anything right now.
                    </td>
                  </tr>
                ) : (
                  (report?.customer_summary ?? []).map((c) => {
                    const bucket = oldestBucket(c);
                    return (
                      <tr key={c.customer_id} className="hover:bg-neutral-50">
                        <td className="px-5 py-3.5">
                          <p className="font-medium text-neutral-900">
                            {c.company_name || c.customer_name}
                          </p>
                          {c.company_name && (
                            <p className="text-sm text-neutral-500">
                              {c.customer_name}
                            </p>
                          )}
                        </td>
                        <td className="px-5 py-3.5 text-neutral-600">
                          {realDate(c.oldest_invoice)
                            ? formatDate(c.oldest_invoice as string)
                            : '—'}
                        </td>
                        <td className="px-5 py-3.5 text-right font-semibold text-neutral-900 tabular-nums">
                          {formatDollars(c.total_owed)}
                        </td>
                        <td className="px-5 py-3.5 text-right text-neutral-700 tabular-nums">
                          {c.invoice_count}
                        </td>
                        <td className="px-5 py-3.5">
                          {bucket ? (
                            <span
                              className={cn(
                                'inline-flex rounded-full border px-2.5 py-0.5 text-sm',
                                AGING_BUCKET_STYLES[bucket].badge
                              )}
                            >
                              {AGING_BUCKET_STYLES[bucket].label}
                            </span>
                          ) : (
                            <span className="text-neutral-400">—</span>
                          )}
                        </td>
                        <td className="px-5 py-3.5 text-neutral-600">
                          {realDate(c.last_payment)
                            ? formatDate(c.last_payment as string)
                            : '—'}
                        </td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>
          {(report?.customer_summary ?? []).length > 0 && (
            <p className="border-t border-neutral-200 bg-neutral-50 px-5 py-3 text-sm text-neutral-600">
              {report?.customer_summary?.length} customers with a balance ·
              click a bucket above to see its invoices
            </p>
          )}
        </section>

        {/* Bucket Detail Slide-out */}
        {selectedBucket && (
          <div className="fixed inset-0 z-50 overflow-hidden">
            <div
              className="absolute inset-0 bg-black/30"
              onClick={() => setSelectedBucket(null)}
            />
            <div className="absolute inset-y-0 right-0 w-full max-w-2xl bg-white shadow-xl">
              <div className="flex flex-col h-full">
                {/* Header */}
                <div className="flex items-center justify-between p-4 border-b border-gray-200">
                  <div>
                    <h3 className="text-lg font-medium text-gray-900">
                      {
                        buckets.find(
                          (b: ARAgingBucketSummary) =>
                            b.bucket === selectedBucket
                        )?.display_name
                      }{' '}
                      Invoices
                    </h3>
                    <p className="text-sm text-gray-500">
                      {bucketInvoices?.length || 0} invoices
                    </p>
                  </div>
                  <button
                    onClick={() => setSelectedBucket(null)}
                    className="p-2 hover:bg-gray-100 rounded-lg"
                  >
                    <X className="h-5 w-5" />
                  </button>
                </div>

                {/* Invoice List */}
                <div className="flex-1 overflow-y-auto p-4">
                  {isLoadingInvoices ? (
                    <div className="space-y-3">
                      {[...Array(5)].map((_, i) => (
                        <div
                          key={i}
                          className="h-20 bg-gray-100 rounded-sm animate-pulse"
                        />
                      ))}
                    </div>
                  ) : bucketInvoices?.length === 0 ? (
                    <div className="text-center py-8 text-gray-500">
                      No invoices in this bucket
                    </div>
                  ) : (
                    <div className="space-y-3">
                      {bucketInvoices?.map((invoice: ARAgingInvoice) => (
                        <div
                          key={invoice.invoice_id}
                          className="p-4 border border-gray-200 rounded-lg hover:border-gray-300"
                        >
                          <div className="flex items-start justify-between">
                            <div>
                              <p className="font-medium text-gray-900">
                                {invoice.invoice_number}
                              </p>
                              <p className="text-sm text-gray-600">
                                {invoice.customer_name}
                                {invoice.company_name && (
                                  <span className="text-gray-400">
                                    {' '}
                                    - {invoice.company_name}
                                  </span>
                                )}
                              </p>
                            </div>
                            <div className="text-right">
                              <p className="font-semibold text-gray-900">
                                {formatMoney(invoice.amount_due)}
                              </p>
                              {invoice.days_overdue > 0 && (
                                <p className="text-sm text-red-600">
                                  {invoice.days_overdue} days overdue
                                </p>
                              )}
                            </div>
                          </div>
                          <div className="mt-2 flex items-center gap-4 text-xs text-gray-500">
                            <span>
                              Issued: {formatDate(invoice.issue_date)}
                            </span>
                            <span>Due: {formatDate(invoice.due_date)}</span>
                            {toNumber(invoice.amount_paid) > 0 && (
                              <span className="text-green-600">
                                Paid: {formatMoney(invoice.amount_paid)}
                              </span>
                            )}
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </DashboardLayout>
  );
}

export default ARAgingReportPage;
