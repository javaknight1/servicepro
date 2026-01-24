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
import { quoteService, QuoteFilters } from '@services/quoteService';
import type { Quote } from '@app-types/quote';
import { FileText, Plus, Search } from 'lucide-react';
import { formatCurrency } from '@app-types';

const getQuoteStatusLabel = (status: string): string => {
  const labels: Record<string, string> = {
    draft: 'Draft',
    sent: 'Sent',
    viewed: 'Viewed',
    accepted: 'Accepted',
    declined: 'Declined',
    rejected: 'Rejected',
    expired: 'Expired',
  };
  return labels[status] || status;
};

export function QuotesPage() {
  const navigate = useNavigate();
  const [quotes, setQuotes] = useState<Quote[]>([]);
  const [total, setTotal] = useState(0);
  const [isLoading, setIsLoading] = useState(true);
  const [filters, setFilters] = useState<QuoteFilters>({
    page: 1,
    page_size: 10,
    sort_by: 'created_at',
    sort_order: 'desc',
  });
  const [searchQuery, setSearchQuery] = useState('');

  useEffect(() => {
    loadQuotes();
  }, [filters]);

  const loadQuotes = async () => {
    setIsLoading(true);
    try {
      const response = await quoteService.getQuotes(filters);
      setQuotes(response.quotes);
      setTotal(response.total);
    } catch (error) {
      console.error('Failed to load quotes:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    setFilters({ ...filters, page: 1 });
    loadQuotes();
  };

  const columns: Column<Quote>[] = [
    {
      key: 'quote_number',
      header: 'Quote #',
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
                {row.customer.name || row.customer.email}
              </div>
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
          {getQuoteStatusLabel(value as string)}
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
      key: 'valid_until',
      header: 'Valid Until',
      sortable: true,
      render: (value, row) => (
        <span
          className={row.is_expired ? 'text-error-600' : 'text-neutral-600'}
        >
          {new Date(value as string).toLocaleDateString()}
        </span>
      ),
    },
  ];

  return (
    <DashboardLayout>
      <div className="p-6 lg:p-8">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
          <div>
            <h1 className="text-2xl font-bold text-neutral-900">Quotes</h1>
            <p className="text-neutral-600 mt-1">
              Create and manage customer quotes
            </p>
          </div>
          <Button
            variant="primary"
            onClick={() => navigate('/quotes/new')}
            className="flex items-center gap-2"
          >
            <Plus className="h-4 w-4" />
            Create Quote
          </Button>
        </div>

        <form onSubmit={handleSearch} className="mb-6">
          <div className="relative max-w-md">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-neutral-400" />
            <input
              type="text"
              placeholder="Search quotes..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-10 pr-4 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            />
          </div>
        </form>

        {!isLoading && quotes.length === 0 && !searchQuery ? (
          <EmptyState
            icon={<FileText className="h-12 w-12" />}
            title="No quotes yet"
            description="Create your first quote for a customer"
            action={{
              label: 'Create Quote',
              onClick: () => navigate('/quotes/new'),
            }}
          />
        ) : (
          <DataTable
            data={quotes}
            columns={columns}
            keyExtractor={(row) => row.id}
            onRowClick={(row) => navigate(`/quotes/${row.id}`)}
            isLoading={isLoading}
            emptyMessage="No quotes found"
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
