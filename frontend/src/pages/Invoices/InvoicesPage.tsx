import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { DashboardLayout } from '@components/layout';
import {
  DataTable,
  Badge,
  EmptyState,
  Button,
  getStatusBadgeVariant,
} from '@components/shared';
import type { Column } from '@components/shared';
import {
  invoiceService,
  Invoice,
  InvoiceFilters,
} from '@services/invoiceService';
import { Receipt, Plus, Search } from 'lucide-react';
import { getInvoiceStatusLabel, formatCurrency } from '@app-types';

export function InvoicesPage() {
  const navigate = useNavigate();
  const [invoices, setInvoices] = useState<Invoice[]>([]);
  const [total, setTotal] = useState(0);
  const [isLoading, setIsLoading] = useState(true);
  const [filters, setFilters] = useState<InvoiceFilters>({
    page: 1,
    page_size: 10,
    sort_by: 'created_at',
    sort_order: 'desc',
  });
  const [searchQuery, setSearchQuery] = useState('');

  useEffect(() => {
    loadInvoices();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filters]);

  const loadInvoices = async () => {
    setIsLoading(true);
    try {
      const response = await invoiceService.getInvoices({
        ...filters,
        search: searchQuery || undefined,
      });
      setInvoices(response.invoices);
      setTotal(response.total);
    } catch (error) {
      console.error('Failed to load invoices:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    setFilters({ ...filters, page: 1 });
    loadInvoices();
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
      render: (value) => (
        <Badge variant={getStatusBadgeVariant(value as string)} size="sm">
          {getInvoiceStatusLabel(value as Invoice['status'])}
        </Badge>
      ),
    },
    {
      key: 'total',
      header: 'Total',
      sortable: true,
      className: 'text-right',
      render: (value) => (
        <span className="font-medium text-neutral-900">
          {formatCurrency(value as number)}
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
          {formatCurrency(value as number)}
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
  ];

  return (
    <DashboardLayout>
      <div className="p-6 lg:p-8">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
          <div>
            <h1 className="text-2xl font-bold text-neutral-900">Invoices</h1>
            <p className="text-neutral-600 mt-1">
              Manage invoices and payments
            </p>
          </div>
          <Button
            variant="primary"
            onClick={() => navigate('/invoices/new')}
            className="flex items-center gap-2"
          >
            <Plus className="h-4 w-4" />
            Create Invoice
          </Button>
        </div>

        <form onSubmit={handleSearch} className="mb-6">
          <div className="relative max-w-md">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-neutral-400" />
            <input
              type="text"
              placeholder="Search invoices..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-10 pr-4 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            />
          </div>
        </form>

        {!isLoading && invoices.length === 0 && !searchQuery ? (
          <EmptyState
            icon={<Receipt className="h-12 w-12" />}
            title="No invoices yet"
            description="Create your first invoice for a customer"
            action={{
              label: 'Create Invoice',
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
