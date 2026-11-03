import { useState, useEffect, useRef } from 'react';
import { useNavigate } from 'react-router';
import { DashboardLayout } from '@components/layout';
import { getErrorMessage } from '@/utils/error';
import {
  DataTable,
  Badge,
  EmptyState,
  ErrorState,
  Button,
  getStatusBadgeVariant,
  PageHeader,
} from '@components/shared';
import type { Column } from '@components/shared';
import {
  invoiceService,
  Invoice,
  InvoiceFilters,
} from '@services/invoiceService';
import { Receipt, Plus, Search, Layers } from 'lucide-react';
import { getInvoiceStatusLabel } from '@app-types';
import {
  ConsolidatedInvoiceModal,
  InvoiceActionsMenu,
} from '@components/invoices';
import { formatDollars } from '@/utils/money';

export function InvoicesPage() {
  const navigate = useNavigate();
  const [invoices, setInvoices] = useState<Invoice[]>([]);
  const [total, setTotal] = useState(0);
  const [isLoading, setIsLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  // Ignore responses from superseded requests (fast paging/searching)
  const requestIdRef = useRef(0);
  const [filters, setFilters] = useState<InvoiceFilters>({
    page: 1,
    page_size: 10,
    sort_by: 'created_at',
    sort_order: 'desc',
  });
  const [searchQuery, setSearchQuery] = useState('');
  const [isConsolidateOpen, setIsConsolidateOpen] = useState(false);

  const loadInvoices = async () => {
    const requestId = ++requestIdRef.current;
    setIsLoading(true);
    setLoadError(null);
    try {
      const response = await invoiceService.getInvoices({
        ...filters,
        search: searchQuery || undefined,
      });
      if (requestId !== requestIdRef.current) return;
      setInvoices(response.invoices);
      setTotal(response.total);
    } catch (error) {
      if (requestId !== requestIdRef.current) return;
      console.error('Failed to load invoices:', error);
      setLoadError(getErrorMessage(error, 'Failed to load invoices'));
    } finally {
      if (requestId === requestIdRef.current) setIsLoading(false);
    }
  };

  useEffect(() => {
    loadInvoices();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filters]);

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    // New filters object re-triggers the load effect (no double request)
    setFilters({ ...filters, page: 1 });
  };

  const columns: Column<Invoice>[] = [
    {
      key: 'invoice_number',
      header: 'Invoice #',
      sortable: true,
      render: (value) => (
        <span className="font-mono text-sm font-medium text-primary-600">
          {value as string}
        </span>
      ),
    },
    {
      key: 'customer',
      header: 'Customer',
      render: (_, row) => (
        <div>
          {row.customer ? (
            <>
              <div className="font-medium text-neutral-900">
                {row.customer.first_name} {row.customer.last_name}
              </div>
              {row.customer.company_name && (
                <div className="text-sm text-neutral-500">
                  {row.customer.company_name}
                </div>
              )}
            </>
          ) : (
            <span className="text-neutral-400">-</span>
          )}
        </div>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      sortable: true,
      render: (value, row) => (
        <span className="inline-flex flex-wrap gap-1">
          <Badge variant={getStatusBadgeVariant(value as string)} size="sm">
            {getInvoiceStatusLabel(value as Invoice['status'])}
          </Badge>
          {row.payment_pending_at && (
            <Badge variant="teal" size="sm">
              Payment pending
            </Badge>
          )}
        </span>
      ),
    },
    {
      key: 'total_amount',
      header: 'Total',
      sortable: true,
      className: 'text-right',
      render: (value) => (
        <span className="font-medium text-neutral-900">
          {formatDollars(value as number)}
        </span>
      ),
    },
    {
      key: 'amount_due',
      header: 'Amount Due',
      sortable: true,
      className: 'text-right',
      render: (value, row) => (
        <span
          className={
            row.status === 'overdue'
              ? 'text-error-600 font-medium'
              : 'text-neutral-600'
          }
        >
          {formatDollars(value as number)}
        </span>
      ),
    },
    {
      key: 'due_date',
      header: 'Due Date',
      sortable: true,
      render: (value, row) => {
        const isOverdue = row.status === 'overdue';
        return (
          <span className={isOverdue ? 'text-error-600' : 'text-neutral-600'}>
            {new Date(value as string).toLocaleDateString()}
          </span>
        );
      },
    },
    {
      key: 'actions',
      header: '',
      className: 'w-12',
      render: (_, row) => (
        <InvoiceActionsMenu
          invoiceId={row.id}
          status={row.status}
          onInvoiceSent={loadInvoices}
        />
      ),
    },
  ];

  return (
    <DashboardLayout>
      <div className="mx-auto max-w-[1440px] p-4 sm:p-6 lg:p-8">
        <PageHeader
          title="Invoices"
          subtitle="Send invoices, record payments and see what is still owed."
          actions={
            <div className="flex flex-wrap gap-2">
              <Button
                variant="outline"
                onClick={() => setIsConsolidateOpen(true)}
                className="gap-2"
              >
                <Layers className="h-4 w-4" />
                Create consolidated invoice
              </Button>
              <Button
                variant="primary"
                onClick={() => navigate('/invoices/new')}
                className="gap-2"
              >
                <Plus className="h-4 w-4" />
                New invoice
              </Button>
            </div>
          }
        />
        {/* One invoice per customer and period (T188). */}
        <ConsolidatedInvoiceModal
          isOpen={isConsolidateOpen}
          onClose={() => setIsConsolidateOpen(false)}
          onCreated={(invoice) => navigate(`/invoices/${invoice.id}`)}
        />

        <form onSubmit={handleSearch} className="mb-6">
          <div className="relative max-w-md">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-neutral-400" />
            <input
              type="text"
              placeholder="Search invoices..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full rounded-xl border border-neutral-300 bg-white py-2.5 pl-10 pr-4 text-[15px] focus:border-primary-500 focus:ring-2 focus:ring-primary-500"
            />
          </div>
        </form>

        {loadError ? (
          <ErrorState
            title="Couldn't load invoices"
            message={loadError}
            onRetry={loadInvoices}
          />
        ) : !isLoading && invoices.length === 0 && !searchQuery ? (
          <EmptyState
            icon={<Receipt className="h-12 w-12" />}
            title="No invoices yet"
            description="Create your first invoice for a customer"
            action={{
              label: 'New invoice',
              onClick: () => navigate('/invoices/new'),
            }}
          />
        ) : (
          <DataTable
            data={invoices}
            columns={columns}
            keyExtractor={(row) => row.id}
            onRowClick={(row) => navigate(`/invoices/${row.id}`)}
            isLoading={isLoading}
            emptyMessage="No invoices found"
            pagination={{
              page: filters.page || 1,
              pageSize: filters.page_size || 10,
              total,
              onPageChange: (page) => setFilters({ ...filters, page }),
            }}
            sorting={{
              sortBy: filters.sort_by || 'created_at',
              sortOrder: filters.sort_order || 'desc',
              onSort: (sortBy, sortOrder) =>
                setFilters({
                  ...filters,
                  sort_by: sortBy,
                  sort_order: sortOrder,
                }),
            }}
          />
        )}
      </div>
    </DashboardLayout>
  );
}
