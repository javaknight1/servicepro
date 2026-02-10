import { useState, useEffect, useCallback } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  DollarSign,
  Clock,
  AlertTriangle,
  Users,
  Download,
  ChevronRight,
  X,
} from 'lucide-react';
import { DashboardLayout } from '@components/layout';
import { arAgingService } from '../../services/arAgingService';
import { customerService, Customer } from '../../services/customerService';
import {
  ARAgingQuery,
  ARAgingBucketSummary,
  ARAgingInvoice,
  AgingBucket,
} from '../../types/arAging';
import { cn } from '../../utils/cn';

// Format currency (handles both string and number inputs from decimal.Decimal)
const formatCurrency = (amount: string | number): string => {
  const numAmount = typeof amount === 'string' ? parseFloat(amount) : amount;
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(numAmount || 0);
};

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
  });
};

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
      <DashboardLayout>
        <div className="p-6">
          <div className="animate-pulse">
            <div className="h-8 bg-gray-200 rounded w-1/4 mb-6" />
            <div className="grid grid-cols-4 gap-4 mb-8">
              {[...Array(4)].map((_, i) => (
                <div key={i} className="h-24 bg-gray-200 rounded" />
              ))}
            </div>
            <div className="h-64 bg-gray-200 rounded" />
          </div>
        </div>
      </DashboardLayout>
    );
  }

  if (error) {
    return (
      <DashboardLayout>
        <div className="p-6">
          <div className="bg-red-50 border border-red-200 rounded-lg p-4 text-red-700">
            Failed to load A/R aging report. Please try again.
          </div>
        </div>
      </DashboardLayout>
    );
  }

  const { summary, buckets } = report || { summary: null, buckets: [] };

  return (
    <DashboardLayout>
      <div className="p-6 max-w-7xl mx-auto">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <div>
            <h1 className="text-2xl font-semibold text-gray-900">
              A/R Aging Report
            </h1>
            <p className="text-gray-500 mt-1">
              Outstanding receivables by age as of{' '}
              {report?.metadata.as_of_date
                ? formatDate(report.metadata.as_of_date)
                : 'today'}
              {selectedCustomer && (
                <span>
                  {' '}
                  for{' '}
                  <span className="font-medium text-gray-700">
                    {selectedCustomer.company_name ||
                      `${selectedCustomer.first_name} ${selectedCustomer.last_name}`}
                  </span>
                </span>
              )}
            </p>
          </div>
          <div className="flex items-center gap-3">
            <select
              value={query.customer_id || ''}
              onChange={(e) => handleCustomerChange(e.target.value)}
              className="px-3 py-2 bg-white border border-gray-300 rounded-lg text-sm focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
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
            <button
              onClick={handleExport}
              disabled={isExporting}
              className="flex items-center gap-2 px-4 py-2 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50"
            >
              <Download className="h-4 w-4" />
              {isExporting ? 'Exporting...' : 'Export CSV'}
            </button>
          </div>
        </div>

        {/* Summary Cards */}
        {summary && (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
            <div className="bg-white rounded-lg border border-gray-200 p-4">
              <div className="flex items-center gap-3">
                <div className="p-2 bg-blue-100 rounded-lg">
                  <DollarSign className="h-5 w-5 text-blue-600" />
                </div>
                <div>
                  <p className="text-sm text-gray-500">Total Outstanding</p>
                  <p className="text-xl font-semibold text-gray-900">
                    {formatCurrency(summary.total_outstanding)}
                  </p>
                </div>
              </div>
            </div>

            <div className="bg-white rounded-lg border border-gray-200 p-4">
              <div className="flex items-center gap-3">
                <div className="p-2 bg-green-100 rounded-lg">
                  <Clock className="h-5 w-5 text-green-600" />
                </div>
                <div>
                  <p className="text-sm text-gray-500">Current (Not Due)</p>
                  <p className="text-xl font-semibold text-gray-900">
                    {formatCurrency(summary.total_current)}
                  </p>
                </div>
              </div>
            </div>

            <div className="bg-white rounded-lg border border-gray-200 p-4">
              <div className="flex items-center gap-3">
                <div className="p-2 bg-red-100 rounded-lg">
                  <AlertTriangle className="h-5 w-5 text-red-600" />
                </div>
                <div>
                  <p className="text-sm text-gray-500">Total Overdue</p>
                  <p className="text-xl font-semibold text-gray-900">
                    {formatCurrency(summary.total_overdue)}
                  </p>
                  <p className="text-xs text-gray-400">
                    {toNumber(summary.overdue_percentage).toFixed(1)}% of total
                  </p>
                </div>
              </div>
            </div>

            <div className="bg-white rounded-lg border border-gray-200 p-4">
              <div className="flex items-center gap-3">
                <div className="p-2 bg-purple-100 rounded-lg">
                  <Users className="h-5 w-5 text-purple-600" />
                </div>
                <div>
                  <p className="text-sm text-gray-500">
                    Customers with Balance
                  </p>
                  <p className="text-xl font-semibold text-gray-900">
                    {summary.customer_count}
                  </p>
                  <p className="text-xs text-gray-400">
                    {summary.invoice_count} invoices
                  </p>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Aging Buckets */}
        <div className="bg-white rounded-lg border border-gray-200 mb-8">
          <div className="p-4 border-b border-gray-200">
            <h2 className="text-lg font-medium text-gray-900">
              Aging Breakdown
            </h2>
            <p className="text-sm text-gray-500">
              Click a bucket to see individual invoices
            </p>
          </div>

          <div className="p-4">
            <div className="grid grid-cols-1 md:grid-cols-5 gap-4">
              {buckets.map((bucket: ARAgingBucketSummary) => (
                <button
                  key={bucket.bucket}
                  onClick={() => setSelectedBucket(bucket.bucket)}
                  className={cn(
                    'p-4 rounded-lg border-2 text-left transition-all hover:shadow-md',
                    selectedBucket === bucket.bucket
                      ? 'border-blue-500 bg-blue-50'
                      : 'border-gray-200 hover:border-gray-300'
                  )}
                >
                  <div className="flex items-center justify-between mb-2">
                    <span
                      className="w-3 h-3 rounded-full"
                      style={{ backgroundColor: bucket.color }}
                    />
                    <ChevronRight className="h-4 w-4 text-gray-400" />
                  </div>
                  <p className="text-sm font-medium text-gray-900">
                    {bucket.display_name}
                  </p>
                  <p
                    className="text-xl font-semibold mt-1"
                    style={{ color: bucket.color }}
                  >
                    {formatCurrency(bucket.total_amount)}
                  </p>
                  <p className="text-xs text-gray-500 mt-1">
                    {bucket.invoice_count} invoice
                    {bucket.invoice_count !== 1 ? 's' : ''} (
                    {toNumber(bucket.percentage).toFixed(1)}%)
                  </p>
                </button>
              ))}
            </div>
          </div>

          {/* Visual Bar */}
          {summary && toNumber(summary.total_outstanding) > 0 && (
            <div className="px-4 pb-4">
              <div className="h-4 rounded-full overflow-hidden flex bg-gray-100">
                {buckets.map((bucket: ARAgingBucketSummary) => (
                  <div
                    key={bucket.bucket}
                    className="h-full transition-all"
                    style={{
                      width: `${toNumber(bucket.percentage)}%`,
                      backgroundColor: bucket.color,
                    }}
                    title={`${bucket.display_name}: ${formatCurrency(bucket.total_amount)}`}
                  />
                ))}
              </div>
            </div>
          )}
        </div>

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
                          className="h-20 bg-gray-100 rounded animate-pulse"
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
                                {formatCurrency(invoice.amount_due)}
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
                                Paid: {formatCurrency(invoice.amount_paid)}
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
