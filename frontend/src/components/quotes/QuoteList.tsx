import React, { useState, useMemo } from 'react';
import { useQuery, keepPreviousData } from '@tanstack/react-query';
import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  getPaginationRowModel,
  flexRender,
  createColumnHelper,
  SortingState,
} from '@tanstack/react-table';
import { quoteService, QuoteFilters } from '../../services/quoteService';
import { Quote, QuoteStatus } from '../../types/quote';

interface QuoteListProps {
  onEdit?: (quote: Quote) => void;
  onDelete?: (quote: Quote) => void;
  onConvert?: (quote: Quote) => void;
  customerId?: string;
}

const columnHelper = createColumnHelper<Quote>();

export const QuoteList: React.FC<QuoteListProps> = ({
  onEdit,
  onDelete,
  onConvert,
  customerId,
}) => {
  // State for filters and pagination
  const [filters, setFilters] = useState<QuoteFilters>({
    customer_id: customerId,
    page: 1,
    page_size: 25,
    sort_by: 'created_at',
    sort_order: 'desc',
  });

  const [sorting, setSorting] = useState<SortingState>([]);
  const [activeQuoteMenu, setActiveQuoteMenu] = useState<string | null>(null);

  // Fetch quotes with React Query
  const {
    data: quotesData,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ['quotes', filters],
    queryFn: () => quoteService.getQuotes(filters),
    placeholderData: keepPreviousData,
  });

  // Define columns
  const columns = useMemo(
    () => [
      columnHelper.accessor('quote_number', {
        header: 'Quote #',
        cell: (info) => (
          <span className="font-mono text-sm font-medium text-gray-900">
            {info.getValue()}
          </span>
        ),
      }),
      columnHelper.accessor((row) => row.customer?.name ?? '', {
        id: 'customer_name',
        header: 'Customer',
        cell: (info) => (
          <div className="flex flex-col">
            <span className="font-medium text-gray-900">{info.getValue()}</span>
            {info.row.original.customer?.email && (
              <span className="text-xs text-gray-500">
                {info.row.original.customer.email}
              </span>
            )}
          </div>
        ),
      }),
      columnHelper.accessor('created_at', {
        header: 'Date',
        cell: (info) => (
          <span className="text-sm text-gray-900">
            {new Date(info.getValue()).toLocaleDateString('en-US', {
              year: 'numeric',
              month: 'short',
              day: 'numeric',
            })}
          </span>
        ),
      }),
      columnHelper.accessor('total', {
        header: 'Amount',
        cell: (info) => (
          <span className="font-medium text-gray-900">
            $
            {info.getValue().toLocaleString('en-US', {
              minimumFractionDigits: 2,
              maximumFractionDigits: 2,
            })}
          </span>
        ),
      }),
      columnHelper.accessor('status', {
        header: 'Status',
        cell: (info) => {
          const status = info.getValue();
          const statusColors: Record<QuoteStatus, string> = {
            [QuoteStatus.DRAFT]: 'bg-gray-100 text-gray-800',
            [QuoteStatus.SENT]: 'bg-blue-100 text-blue-800',
            [QuoteStatus.VIEWED]: 'bg-purple-100 text-purple-800',
            [QuoteStatus.ACCEPTED]: 'bg-green-100 text-green-800',
            [QuoteStatus.DECLINED]: 'bg-red-100 text-red-800',
            [QuoteStatus.EXPIRED]: 'bg-yellow-100 text-yellow-800',
          };

          return (
            <span
              className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${
                statusColors[status as QuoteStatus] ||
                'bg-gray-100 text-gray-800'
              }`}
            >
              {status.charAt(0).toUpperCase() + status.slice(1)}
            </span>
          );
        },
      }),
      columnHelper.display({
        id: 'actions',
        header: '',
        cell: (info) => {
          const quote = info.row.original;
          const isMenuOpen = activeQuoteMenu === quote.id;

          return (
            <div className="relative">
              <button
                onClick={() => setActiveQuoteMenu(isMenuOpen ? null : quote.id)}
                className="rounded p-1 hover:bg-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-500"
                aria-label="Actions"
              >
                <svg
                  className="h-5 w-5 text-gray-500"
                  fill="currentColor"
                  viewBox="0 0 20 20"
                >
                  <path d="M10 6a2 2 0 110-4 2 2 0 010 4zM10 12a2 2 0 110-4 2 2 0 010 4zM10 18a2 2 0 110-4 2 2 0 010 4z" />
                </svg>
              </button>

              {isMenuOpen && (
                <>
                  <div
                    className="fixed inset-0 z-10"
                    onClick={() => setActiveQuoteMenu(null)}
                  />
                  <div className="absolute right-0 z-20 mt-2 w-48 origin-top-right rounded-md bg-white shadow-lg ring-1 ring-black ring-opacity-5">
                    <div className="py-1">
                      {onEdit && (
                        <button
                          onClick={() => {
                            onEdit(quote);
                            setActiveQuoteMenu(null);
                          }}
                          className="block w-full px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100"
                        >
                          Edit Quote
                        </button>
                      )}
                      {onConvert && quote.status === QuoteStatus.ACCEPTED && (
                        <button
                          onClick={() => {
                            onConvert(quote);
                            setActiveQuoteMenu(null);
                          }}
                          className="block w-full px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100"
                        >
                          Convert to Job
                        </button>
                      )}
                      {quote.status === QuoteStatus.DRAFT && (
                        <button
                          onClick={async () => {
                            await quoteService.sendQuote(quote.id);
                            refetch();
                            setActiveQuoteMenu(null);
                          }}
                          className="block w-full px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100"
                        >
                          Send to Customer
                        </button>
                      )}
                      {onDelete && (
                        <button
                          onClick={() => {
                            onDelete(quote);
                            setActiveQuoteMenu(null);
                          }}
                          className="block w-full px-4 py-2 text-left text-sm text-red-700 hover:bg-red-50"
                        >
                          Delete Quote
                        </button>
                      )}
                    </div>
                  </div>
                </>
              )}
            </div>
          );
        },
      }),
    ],
    [activeQuoteMenu, onEdit, onDelete, onConvert, refetch]
  );

  // Create table instance
  const table = useReactTable({
    data: quotesData?.quotes || [],
    columns,
    state: {
      sorting,
    },
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
    manualPagination: true,
    pageCount:
      quotesData?.total && quotesData?.page_size
        ? Math.ceil(quotesData.total / quotesData.page_size)
        : 0,
  });

  // Update filters when sorting changes
  React.useEffect(() => {
    if (sorting.length > 0) {
      const sort = sorting[0];
      setFilters((prev) => ({
        ...prev,
        sort_by: sort.id,
        sort_order: sort.desc ? 'desc' : 'asc',
        page: 1,
      }));
    }
  }, [sorting]);

  // Handle filter changes
  const handleStatusFilter = (status: string) => {
    setFilters((prev) => ({
      ...prev,
      status: status === 'all' ? undefined : status,
      page: 1,
    }));
  };

  const handlePageSizeChange = (pageSize: number) => {
    setFilters((prev) => ({
      ...prev,
      page_size: pageSize,
      page: 1,
    }));
  };

  const handlePageChange = (page: number) => {
    setFilters((prev) => ({
      ...prev,
      page,
    }));
  };

  // Loading state
  if (isLoading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="flex flex-col items-center gap-2">
          <div className="h-8 w-8 animate-spin rounded-full border-4 border-blue-600 border-t-transparent"></div>
          <p className="text-sm text-gray-500">Loading quotes...</p>
        </div>
      </div>
    );
  }

  // Error state
  if (isError) {
    return (
      <div className="rounded-md bg-red-50 p-4">
        <div className="flex">
          <div className="flex-shrink-0">
            <svg
              className="h-5 w-5 text-red-400"
              viewBox="0 0 20 20"
              fill="currentColor"
            >
              <path
                fillRule="evenodd"
                d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                clipRule="evenodd"
              />
            </svg>
          </div>
          <div className="ml-3">
            <h3 className="text-sm font-medium text-red-800">
              Error loading quotes
            </h3>
            <div className="mt-2 text-sm text-red-700">
              <p>
                {error instanceof Error ? error.message : 'An error occurred'}
              </p>
            </div>
            <div className="mt-4">
              <button
                onClick={() => refetch()}
                className="rounded-md bg-red-100 px-3 py-2 text-sm font-medium text-red-800 hover:bg-red-200 focus:outline-none focus:ring-2 focus:ring-red-600 focus:ring-offset-2 focus:ring-offset-red-50"
              >
                Try again
              </button>
            </div>
          </div>
        </div>
      </div>
    );
  }

  const quotes = quotesData?.quotes || [];
  const pagination = quotesData
    ? {
        total: quotesData.total,
        page: quotesData.page,
        page_size: quotesData.page_size,
      }
    : undefined;

  return (
    <div className="space-y-4">
      {/* Filter Controls */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex flex-wrap gap-2">
          <button
            onClick={() => handleStatusFilter('all')}
            className={`rounded-md px-3 py-2 text-sm font-medium ${
              !filters.status
                ? 'bg-blue-600 text-white'
                : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
            }`}
          >
            All
          </button>
          <button
            onClick={() => handleStatusFilter(QuoteStatus.DRAFT)}
            className={`rounded-md px-3 py-2 text-sm font-medium ${
              filters.status === QuoteStatus.DRAFT
                ? 'bg-blue-600 text-white'
                : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
            }`}
          >
            Draft
          </button>
          <button
            onClick={() => handleStatusFilter(QuoteStatus.SENT)}
            className={`rounded-md px-3 py-2 text-sm font-medium ${
              filters.status === QuoteStatus.SENT
                ? 'bg-blue-600 text-white'
                : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
            }`}
          >
            Sent
          </button>
          <button
            onClick={() => handleStatusFilter(QuoteStatus.ACCEPTED)}
            className={`rounded-md px-3 py-2 text-sm font-medium ${
              filters.status === QuoteStatus.ACCEPTED
                ? 'bg-blue-600 text-white'
                : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
            }`}
          >
            Accepted
          </button>
        </div>

        {/* Page size selector */}
        <div className="flex items-center gap-2">
          <span className="text-sm text-gray-700">Show:</span>
          <select
            value={filters.page_size}
            onChange={(e) => handlePageSizeChange(Number(e.target.value))}
            className="rounded-md border-gray-300 py-1 pl-3 pr-10 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          >
            <option value={25}>25</option>
            <option value={50}>50</option>
            <option value={100}>100</option>
          </select>
        </div>
      </div>

      {/* Table */}
      <div className="overflow-hidden rounded-lg border border-gray-200 bg-white shadow">
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              {table.getHeaderGroups().map((headerGroup) => (
                <tr key={headerGroup.id}>
                  {headerGroup.headers.map((header) => (
                    <th
                      key={header.id}
                      className="px-6 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500"
                    >
                      {header.isPlaceholder ? null : (
                        <div
                          className={
                            header.column.getCanSort()
                              ? 'flex cursor-pointer select-none items-center gap-2'
                              : ''
                          }
                          onClick={header.column.getToggleSortingHandler()}
                        >
                          {flexRender(
                            header.column.columnDef.header,
                            header.getContext()
                          )}
                          {header.column.getCanSort() && (
                            <span className="text-gray-400">
                              {{
                                asc: '↑',
                                desc: '↓',
                              }[header.column.getIsSorted() as string] ?? '↕'}
                            </span>
                          )}
                        </div>
                      )}
                    </th>
                  ))}
                </tr>
              ))}
            </thead>
            <tbody className="divide-y divide-gray-200 bg-white">
              {quotes.length === 0 ? (
                <tr>
                  <td
                    colSpan={columns.length}
                    className="px-6 py-12 text-center text-sm text-gray-500"
                  >
                    No quotes found
                  </td>
                </tr>
              ) : (
                table.getRowModel().rows.map((row) => (
                  <tr
                    key={row.id}
                    className="hover:bg-gray-50 transition-colors"
                  >
                    {row.getVisibleCells().map((cell) => (
                      <td key={cell.id} className="whitespace-nowrap px-6 py-4">
                        {flexRender(
                          cell.column.columnDef.cell,
                          cell.getContext()
                        )}
                      </td>
                    ))}
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>

        {/* Pagination */}
        {pagination && pagination.total > 0 && (
          <div className="flex items-center justify-between border-t border-gray-200 bg-white px-4 py-3 sm:px-6">
            <div className="flex flex-1 justify-between sm:hidden">
              <button
                onClick={() => handlePageChange(pagination.page - 1)}
                disabled={pagination.page === 1}
                className="relative inline-flex items-center rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
              >
                Previous
              </button>
              <button
                onClick={() => handlePageChange(pagination.page + 1)}
                disabled={
                  pagination.page >=
                  Math.ceil(pagination.total / pagination.page_size)
                }
                className="relative ml-3 inline-flex items-center rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
              >
                Next
              </button>
            </div>
            <div className="hidden sm:flex sm:flex-1 sm:items-center sm:justify-between">
              <div>
                <p className="text-sm text-gray-700">
                  Showing{' '}
                  <span className="font-medium">
                    {(pagination.page - 1) * pagination.page_size + 1}
                  </span>{' '}
                  to{' '}
                  <span className="font-medium">
                    {Math.min(
                      pagination.page * pagination.page_size,
                      pagination.total
                    )}
                  </span>{' '}
                  of <span className="font-medium">{pagination.total}</span>{' '}
                  results
                </p>
              </div>
              <div>
                <nav
                  className="isolate inline-flex -space-x-px rounded-md shadow-sm"
                  aria-label="Pagination"
                >
                  <button
                    onClick={() => handlePageChange(pagination.page - 1)}
                    disabled={pagination.page === 1}
                    className="relative inline-flex items-center rounded-l-md border border-gray-300 bg-white px-2 py-2 text-sm font-medium text-gray-500 hover:bg-gray-50 focus:z-20 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    <span className="sr-only">Previous</span>
                    <svg
                      className="h-5 w-5"
                      viewBox="0 0 20 20"
                      fill="currentColor"
                    >
                      <path
                        fillRule="evenodd"
                        d="M12.707 5.293a1 1 0 010 1.414L9.414 10l3.293 3.293a1 1 0 01-1.414 1.414l-4-4a1 1 0 010-1.414l4-4a1 1 0 011.414 0z"
                        clipRule="evenodd"
                      />
                    </svg>
                  </button>

                  {/* Page numbers */}
                  {Array.from(
                    {
                      length: Math.ceil(
                        pagination.total / pagination.page_size
                      ),
                    },
                    (_, i) => i + 1
                  )
                    .filter((page) => {
                      // Show first page, last page, current page, and pages around current
                      const totalPages = Math.ceil(
                        pagination.total / pagination.page_size
                      );
                      return (
                        page === 1 ||
                        page === totalPages ||
                        Math.abs(page - pagination.page) <= 1
                      );
                    })
                    .map((page, index, array) => {
                      // Add ellipsis if there's a gap
                      const prevPage = array[index - 1];
                      const showEllipsis = prevPage && page - prevPage > 1;

                      return (
                        <React.Fragment key={page}>
                          {showEllipsis && (
                            <span className="relative inline-flex items-center border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700">
                              ...
                            </span>
                          )}
                          <button
                            onClick={() => handlePageChange(page)}
                            className={`relative inline-flex items-center border px-4 py-2 text-sm font-medium focus:z-20 ${
                              page === pagination.page
                                ? 'z-10 border-blue-500 bg-blue-50 text-blue-600'
                                : 'border-gray-300 bg-white text-gray-500 hover:bg-gray-50'
                            }`}
                          >
                            {page}
                          </button>
                        </React.Fragment>
                      );
                    })}

                  <button
                    onClick={() => handlePageChange(pagination.page + 1)}
                    disabled={
                      pagination.page >=
                      Math.ceil(pagination.total / pagination.page_size)
                    }
                    className="relative inline-flex items-center rounded-r-md border border-gray-300 bg-white px-2 py-2 text-sm font-medium text-gray-500 hover:bg-gray-50 focus:z-20 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    <span className="sr-only">Next</span>
                    <svg
                      className="h-5 w-5"
                      viewBox="0 0 20 20"
                      fill="currentColor"
                    >
                      <path
                        fillRule="evenodd"
                        d="M7.293 14.707a1 1 0 010-1.414L10.586 10 7.293 6.707a1 1 0 011.414-1.414l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0z"
                        clipRule="evenodd"
                      />
                    </svg>
                  </button>
                </nav>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
