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
  PageHeader,
  Avatar,
  getStatusBadgeVariant,
} from '@components/shared';
import type { Column } from '@components/shared';
import { customerService } from '@services/customerService';
import type { Customer, CustomerFilters } from '@services/customerService';
import { Users, Plus, Search } from 'lucide-react';
import { getCustomerStatusLabel, getCustomerTypeLabel } from '@app-types';

export function CustomersPage() {
  const navigate = useNavigate();
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [total, setTotal] = useState(0);
  const [isLoading, setIsLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  // Ignore responses from superseded requests (fast paging/searching)
  const requestIdRef = useRef(0);
  const [filters, setFilters] = useState<CustomerFilters>({
    page: 1,
    page_size: 10,
    sort_by: 'created_at',
    sort_order: 'desc',
  });
  const [searchQuery, setSearchQuery] = useState('');

  const loadCustomers = async () => {
    const requestId = ++requestIdRef.current;
    setIsLoading(true);
    setLoadError(null);
    try {
      const response = await customerService.getCustomers({
        ...filters,
        search: searchQuery || undefined,
      });
      if (requestId !== requestIdRef.current) return;
      setCustomers(response.customers);
      setTotal(response.total);
    } catch (error) {
      if (requestId !== requestIdRef.current) return;
      console.error('Failed to load customers:', error);
      setLoadError(getErrorMessage(error, 'Failed to load customers'));
    } finally {
      if (requestId === requestIdRef.current) setIsLoading(false);
    }
  };

  useEffect(() => {
    loadCustomers();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filters]);

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    // New filters object re-triggers the load effect (no double request)
    setFilters({ ...filters, page: 1 });
  };

  const columns: Column<Customer>[] = [
    {
      key: 'name',
      header: 'Name',
      sortable: true,
      render: (_, row) => (
        <div className="flex items-center gap-3">
          <Avatar
            email={row.email || row.id}
            firstName={row.company_name || row.first_name}
            lastName={row.company_name ? undefined : row.last_name}
            shape="tile"
            size="md"
          />
          <div className="min-w-0">
            <div className="font-medium text-neutral-900">
              {row.first_name} {row.last_name}
            </div>
            {row.company_name && (
              <div className="text-sm text-neutral-500">{row.company_name}</div>
            )}
          </div>
        </div>
      ),
    },
    {
      key: 'email',
      header: 'Email',
      sortable: true,
      render: (value) => (
        <span className="text-neutral-600">{value as string}</span>
      ),
    },
    {
      key: 'phone_primary',
      header: 'Phone',
      render: (value) => (
        <span className="text-neutral-600">{(value as string) || '-'}</span>
      ),
    },
    {
      key: 'customer_type',
      header: 'Type',
      sortable: true,
      render: (value) => (
        <Badge variant="neutral" size="sm">
          {getCustomerTypeLabel(value as Customer['customer_type'])}
        </Badge>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      sortable: true,
      render: (value) => (
        <Badge variant={getStatusBadgeVariant(value as string)} size="sm">
          {getCustomerStatusLabel(value as Customer['status'])}
        </Badge>
      ),
    },
  ];

  return (
    <DashboardLayout>
      <div className="mx-auto max-w-[1440px] p-4 sm:p-6 lg:p-8">
        <PageHeader
          title="Customers"
          subtitle={
            total > 0
              ? `${total} ${total === 1 ? 'customer' : 'customers'} · the businesses you bill`
              : 'The businesses and people you bill'
          }
          actions={
            <Button
              variant="primary"
              onClick={() => navigate('/customers/new')}
              className="gap-2"
            >
              <Plus className="h-4 w-4" />
              Add customer
            </Button>
          }
        />

        <form onSubmit={handleSearch} className="mb-6">
          <div className="relative max-w-md">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-neutral-400" />
            <input
              type="text"
              placeholder="Search customers..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full rounded-xl border border-neutral-300 bg-white py-2.5 pl-10 pr-4 text-[15px] focus:border-primary-500 focus:ring-2 focus:ring-primary-500"
            />
          </div>
        </form>

        {loadError ? (
          <ErrorState
            title="Couldn't load customers"
            message={loadError}
            onRetry={loadCustomers}
          />
        ) : !isLoading && customers.length === 0 && !searchQuery ? (
          <EmptyState
            icon={<Users className="h-12 w-12" />}
            title="No customers yet"
            description="Get started by adding your first customer"
            action={{
              label: 'Add customer',
              onClick: () => navigate('/customers/new'),
            }}
          />
        ) : (
          <DataTable
            data={customers}
            columns={columns}
            keyExtractor={(row) => row.id}
            onRowClick={(row) => navigate(`/customers/${row.id}`)}
            isLoading={isLoading}
            emptyMessage="No customers found"
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
