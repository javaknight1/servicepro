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
import { customerService } from '@services/customerService';
import type { Customer, CustomerFilters } from '@services/customerService';
import { Users, Plus, Search } from 'lucide-react';
import { getCustomerStatusLabel, getCustomerTypeLabel } from '@types';

export function CustomersPage() {
  const navigate = useNavigate();
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [total, setTotal] = useState(0);
  const [isLoading, setIsLoading] = useState(true);
  const [filters, setFilters] = useState<CustomerFilters>({
    page: 1,
    page_size: 10,
    sort_by: 'created_at',
    sort_order: 'desc',
  });
  const [searchQuery, setSearchQuery] = useState('');

  useEffect(() => {
    loadCustomers();
  }, [filters]);

  const loadCustomers = async () => {
    setIsLoading(true);
    try {
      const response = await customerService.getCustomers({
        ...filters,
        search: searchQuery || undefined,
      });
      setCustomers(response.customers);
      setTotal(response.total);
    } catch (error) {
      console.error('Failed to load customers:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    setFilters({ ...filters, page: 1 });
    loadCustomers();
  };

  const columns: Column<Customer>[] = [
    {
      key: 'name',
      header: 'Name',
      sortable: true,
      render: (_, row) => (
        <div>
          <div className="font-medium text-neutral-900">
            {row.first_name} {row.last_name}
          </div>
          {row.company_name && (
            <div className="text-sm text-neutral-500">{row.company_name}</div>
          )}
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
        <Badge variant="info" size="sm">
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
      <div className="p-6 lg:p-8">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
          <div>
            <h1 className="text-2xl font-bold text-neutral-900">Customers</h1>
            <p className="text-neutral-600 mt-1">
              Manage your customer database
            </p>
          </div>
          <Button
            variant="primary"
            onClick={() => navigate('/customers/new')}
            className="flex items-center gap-2"
          >
            <Plus className="h-4 w-4" />
            Add Customer
          </Button>
        </div>

        <form onSubmit={handleSearch} className="mb-6">
          <div className="relative max-w-md">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-neutral-400" />
            <input
              type="text"
              placeholder="Search customers..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-10 pr-4 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            />
          </div>
        </form>

        {!isLoading && customers.length === 0 && !searchQuery ? (
          <EmptyState
            icon={<Users className="h-12 w-12" />}
            title="No customers yet"
            description="Get started by adding your first customer"
            action={{
              label: 'Add Customer',
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
