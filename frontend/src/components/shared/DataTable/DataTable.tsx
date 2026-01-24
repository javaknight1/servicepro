import { useState, useMemo } from 'react';
import { cn } from '@utils/cn';
import {
  ChevronUp,
  ChevronDown,
  ChevronsUpDown,
  ChevronLeft,
  ChevronRight,
} from 'lucide-react';

export interface Column<T> {
  key: string;
  header: string;
  sortable?: boolean;
  className?: string;
  render?: (value: unknown, row: T) => React.ReactNode;
}

export interface DataTableProps<T> {
  data: T[];
  columns: Column<T>[];
  keyExtractor: (row: T) => string;
  onRowClick?: (row: T) => void;
  isLoading?: boolean;
  emptyMessage?: string;
  pagination?: {
    page: number;
    pageSize: number;
    total: number;
    onPageChange: (page: number) => void;
  };
  sorting?: {
    sortBy: string;
    sortOrder: 'asc' | 'desc';
    onSort: (key: string, order: 'asc' | 'desc') => void;
  };
  className?: string;
}

export function DataTable<T extends object>({
  data,
  columns,
  keyExtractor,
  onRowClick,
  isLoading = false,
  emptyMessage = 'No data available',
  pagination,
  sorting,
  className,
}: DataTableProps<T>) {
  const [localSortBy, setLocalSortBy] = useState<string>('');
  const [localSortOrder, setLocalSortOrder] = useState<'asc' | 'desc'>('asc');

  const effectiveSortBy = sorting?.sortBy ?? localSortBy;
  const effectiveSortOrder = sorting?.sortOrder ?? localSortOrder;

  const handleSort = (key: string) => {
    const newOrder =
      effectiveSortBy === key && effectiveSortOrder === 'asc' ? 'desc' : 'asc';

    if (sorting) {
      sorting.onSort(key, newOrder);
    } else {
      setLocalSortBy(key);
      setLocalSortOrder(newOrder);
    }
  };

  const sortedData = useMemo(() => {
    if (!effectiveSortBy || sorting) return data;

    return [...data].sort((a, b) => {
      const aVal = (a as Record<string, unknown>)[effectiveSortBy];
      const bVal = (b as Record<string, unknown>)[effectiveSortBy];

      if (aVal === bVal) return 0;
      if (aVal === null || aVal === undefined) return 1;
      if (bVal === null || bVal === undefined) return -1;

      const comparison = String(aVal).localeCompare(String(bVal));
      return effectiveSortOrder === 'asc' ? comparison : -comparison;
    });
  }, [data, effectiveSortBy, effectiveSortOrder, sorting]);

  const getSortIcon = (key: string, sortable?: boolean) => {
    if (!sortable) return null;

    if (effectiveSortBy !== key) {
      return <ChevronsUpDown className="h-4 w-4 text-neutral-400" />;
    }

    return effectiveSortOrder === 'asc' ? (
      <ChevronUp className="h-4 w-4 text-primary-600" />
    ) : (
      <ChevronDown className="h-4 w-4 text-primary-600" />
    );
  };

  const totalPages = pagination
    ? Math.ceil(pagination.total / pagination.pageSize)
    : 1;

  return (
    <div
      className={cn(
        'bg-white rounded-xl border border-neutral-200 overflow-hidden',
        className
      )}
    >
      <div className="overflow-x-auto">
        <table className="w-full">
          <thead className="bg-neutral-50 border-b border-neutral-200">
            <tr>
              {columns.map((column) => (
                <th
                  key={column.key}
                  className={cn(
                    'px-4 py-3 text-left text-sm font-semibold text-neutral-700',
                    column.sortable &&
                      'cursor-pointer select-none hover:bg-neutral-100',
                    column.className
                  )}
                  onClick={
                    column.sortable ? () => handleSort(column.key) : undefined
                  }
                >
                  <div className="flex items-center gap-1">
                    {column.header}
                    {getSortIcon(column.key, column.sortable)}
                  </div>
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-neutral-200">
            {isLoading ? (
              <tr>
                <td colSpan={columns.length} className="px-4 py-12 text-center">
                  <div className="flex items-center justify-center gap-2">
                    <div className="h-5 w-5 animate-spin rounded-full border-2 border-primary-600 border-t-transparent" />
                    <span className="text-neutral-600">Loading...</span>
                  </div>
                </td>
              </tr>
            ) : sortedData.length === 0 ? (
              <tr>
                <td
                  colSpan={columns.length}
                  className="px-4 py-12 text-center text-neutral-500"
                >
                  {emptyMessage}
                </td>
              </tr>
            ) : (
              sortedData.map((row) => (
                <tr
                  key={keyExtractor(row)}
                  className={cn(
                    'hover:bg-neutral-50 transition-colors',
                    onRowClick && 'cursor-pointer'
                  )}
                  onClick={onRowClick ? () => onRowClick(row) : undefined}
                >
                  {columns.map((column) => (
                    <td
                      key={column.key}
                      className={cn('px-4 py-3 text-sm', column.className)}
                    >
                      {column.render
                        ? column.render((row as Record<string, unknown>)[column.key], row)
                        : ((row as Record<string, unknown>)[column.key] as React.ReactNode) ?? '-'}
                    </td>
                  ))}
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {pagination && totalPages > 1 && (
        <div className="flex items-center justify-between px-4 py-3 border-t border-neutral-200 bg-neutral-50">
          <div className="text-sm text-neutral-600">
            Showing {(pagination.page - 1) * pagination.pageSize + 1} to{' '}
            {Math.min(pagination.page * pagination.pageSize, pagination.total)}{' '}
            of {pagination.total} results
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={() => pagination.onPageChange(pagination.page - 1)}
              disabled={pagination.page === 1}
              className="p-2 rounded-lg hover:bg-neutral-200 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <ChevronLeft className="h-4 w-4" />
            </button>
            <span className="text-sm text-neutral-700">
              Page {pagination.page} of {totalPages}
            </span>
            <button
              onClick={() => pagination.onPageChange(pagination.page + 1)}
              disabled={pagination.page >= totalPages}
              className="p-2 rounded-lg hover:bg-neutral-200 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              <ChevronRight className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
