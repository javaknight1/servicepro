import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { templateService } from '../../services/templateService';
import type { QuoteTemplate, TemplateListFilter } from '../../types/template';

interface TemplateListProps {
  onSelect?: (template: QuoteTemplate) => void;
  onEdit?: (template: QuoteTemplate) => void;
  filter?: TemplateListFilter;
  selectable?: boolean;
  multiSelect?: boolean;
  selectedIds?: string[];
  onSelectionChange?: (ids: string[]) => void;
}

export const TemplateList: React.FC<TemplateListProps> = ({
  onSelect,
  onEdit,
  filter: initialFilter,
  selectable = false,
  multiSelect = false,
  selectedIds = [],
  onSelectionChange,
}) => {
  const queryClient = useQueryClient();
  const [filter, setFilter] = useState<TemplateListFilter>(
    initialFilter || {
      page: 1,
      page_size: 20,
      sort_by: 'created_at',
      sort_order: 'desc',
    }
  );
  const [searchTerm, setSearchTerm] = useState('');

  // Fetch templates
  const { data, isLoading, error } = useQuery({
    queryKey: ['templates', filter],
    queryFn: () => templateService.listTemplates(filter),
  });

  // Fetch categories
  const { data: categories } = useQuery({
    queryKey: ['template-categories'],
    queryFn: () => templateService.listCategories(),
  });

  // Delete mutation
  const deleteMutation = useMutation({
    mutationFn: (id: string) => templateService.deleteTemplate(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['templates'] });
    },
  });

  // Duplicate mutation
  const duplicateMutation = useMutation({
    mutationFn: (id: string) => templateService.duplicateTemplate(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['templates'] });
    },
  });

  const handleSearch = () => {
    setFilter({ ...filter, search: searchTerm, page: 1 });
  };

  const handleCategoryFilter = (category: string) => {
    setFilter({
      ...filter,
      category: category === filter.category ? undefined : category,
      page: 1,
    });
  };

  const handleDelete = (template: QuoteTemplate) => {
    if (window.confirm(`Are you sure you want to delete "${template.name}"?`)) {
      deleteMutation.mutate(template.id);
    }
  };

  const handleDuplicate = (template: QuoteTemplate) => {
    duplicateMutation.mutate(template.id);
  };

  const handleSelectionToggle = (id: string) => {
    if (!selectable) return;

    if (multiSelect) {
      const newSelection = selectedIds.includes(id)
        ? selectedIds.filter((selectedId) => selectedId !== id)
        : [...selectedIds, id];
      onSelectionChange?.(newSelection);
    } else {
      onSelectionChange?.(selectedIds.includes(id) ? [] : [id]);
    }
  };

  const handlePageChange = (newPage: number) => {
    setFilter({ ...filter, page: newPage });
  };

  if (error) {
    return (
      <div className="rounded-md bg-red-50 p-4">
        <p className="text-sm text-red-800">
          Error loading templates:{' '}
          {error instanceof Error ? error.message : 'Unknown error'}
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Search and Filters */}
      <div className="rounded-lg border bg-white p-4">
        <div className="flex gap-4">
          {/* Search */}
          <div className="flex-1">
            <div className="flex gap-2">
              <input
                type="text"
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                onKeyPress={(e) => e.key === 'Enter' && handleSearch()}
                placeholder="Search templates..."
                className="block w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              />
              <button
                type="button"
                onClick={handleSearch}
                className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
              >
                Search
              </button>
            </div>
          </div>

          {/* Sort */}
          <div>
            <select
              value={`${filter.sort_by}_${filter.sort_order}`}
              onChange={(e) => {
                const [sort_by, sort_order] = e.target.value.split('_');
                setFilter({
                  ...filter,
                  sort_by,
                  sort_order: sort_order as 'asc' | 'desc',
                });
              }}
              className="block w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            >
              <option value="created_at_desc">Newest First</option>
              <option value="created_at_asc">Oldest First</option>
              <option value="name_asc">Name A-Z</option>
              <option value="name_desc">Name Z-A</option>
              <option value="usage_count_desc">Most Used</option>
            </select>
          </div>
        </div>

        {/* Category Filters */}
        {categories && categories.length > 0 && (
          <div className="mt-4 flex flex-wrap gap-2">
            <button
              type="button"
              onClick={() =>
                setFilter({ ...filter, category: undefined, page: 1 })
              }
              className={`rounded-full px-3 py-1 text-sm font-medium ${
                !filter.category
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
              }`}
            >
              All Categories
            </button>
            {categories.map((category) => (
              <button
                key={category.id}
                type="button"
                onClick={() => handleCategoryFilter(category.name)}
                className={`rounded-full px-3 py-1 text-sm font-medium ${
                  filter.category === category.name
                    ? 'text-white'
                    : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                }`}
                style={{
                  backgroundColor:
                    filter.category === category.name
                      ? category.color
                      : undefined,
                }}
              >
                {category.icon} {category.name}
              </button>
            ))}
          </div>
        )}

        {/* Active Filters Display */}
        {(filter.search || filter.category) && (
          <div className="mt-3 flex items-center gap-2 text-sm">
            <span className="text-gray-500">Active filters:</span>
            {filter.search && (
              <span className="inline-flex items-center rounded-full bg-blue-100 px-2 py-1 text-xs font-medium text-blue-800">
                Search: {filter.search}
                <button
                  type="button"
                  onClick={() => setFilter({ ...filter, search: undefined })}
                  className="ml-1 text-blue-600 hover:text-blue-800"
                >
                  ×
                </button>
              </span>
            )}
            {filter.category && (
              <span className="inline-flex items-center rounded-full bg-blue-100 px-2 py-1 text-xs font-medium text-blue-800">
                Category: {filter.category}
                <button
                  type="button"
                  onClick={() => setFilter({ ...filter, category: undefined })}
                  className="ml-1 text-blue-600 hover:text-blue-800"
                >
                  ×
                </button>
              </span>
            )}
          </div>
        )}
      </div>

      {/* Templates List */}
      {isLoading ? (
        <div className="flex items-center justify-center p-8">
          <div className="text-gray-600">Loading templates...</div>
        </div>
      ) : !data || data.templates.length === 0 ? (
        <div className="rounded-lg border bg-white p-8 text-center">
          <svg
            className="mx-auto h-12 w-12 text-gray-400"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
            />
          </svg>
          <h3 className="mt-2 text-sm font-medium text-gray-900">
            No templates found
          </h3>
          <p className="mt-1 text-sm text-gray-500">
            {filter.search || filter.category
              ? 'Try adjusting your search or filters'
              : 'Get started by creating a new template'}
          </p>
        </div>
      ) : (
        <>
          <div className="space-y-3">
            {data.templates.map((template) => {
              const isSelected = selectedIds.includes(template.id);

              return (
                <div
                  key={template.id}
                  className={`rounded-lg border bg-white p-4 transition-all hover:shadow-md ${
                    isSelected ? 'ring-2 ring-blue-500' : ''
                  }`}
                >
                  <div className="flex items-start gap-4">
                    {/* Selection Checkbox */}
                    {selectable && (
                      <div className="flex items-center pt-1">
                        <input
                          type="checkbox"
                          checked={isSelected}
                          onChange={() => handleSelectionToggle(template.id)}
                          className="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                        />
                      </div>
                    )}

                    {/* Template Info */}
                    <div className="flex-1">
                      <div className="flex items-start justify-between">
                        <div className="flex-1">
                          <h3 className="text-lg font-semibold text-gray-900">
                            {template.name}
                          </h3>
                          {template.description && (
                            <p className="mt-1 text-sm text-gray-500">
                              {template.description}
                            </p>
                          )}
                        </div>
                        <div className="ml-4 flex gap-2">
                          {!template.is_active && (
                            <span className="inline-flex items-center rounded-full bg-gray-100 px-2 py-1 text-xs font-medium text-gray-800">
                              Inactive
                            </span>
                          )}
                          {template.is_public && (
                            <span className="inline-flex items-center rounded-full bg-blue-100 px-2 py-1 text-xs font-medium text-blue-800">
                              Public
                            </span>
                          )}
                        </div>
                      </div>

                      {/* Metadata */}
                      <div className="mt-3 flex flex-wrap items-center gap-3 text-sm text-gray-500">
                        <span className="inline-flex items-center">
                          <svg
                            className="mr-1 h-4 w-4"
                            fill="none"
                            viewBox="0 0 24 24"
                            stroke="currentColor"
                          >
                            <path
                              strokeLinecap="round"
                              strokeLinejoin="round"
                              strokeWidth={2}
                              d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z"
                            />
                          </svg>
                          {template.category}
                        </span>
                        <span className="inline-flex items-center">
                          <svg
                            className="mr-1 h-4 w-4"
                            fill="none"
                            viewBox="0 0 24 24"
                            stroke="currentColor"
                          >
                            <path
                              strokeLinecap="round"
                              strokeLinejoin="round"
                              strokeWidth={2}
                              d="M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z"
                            />
                          </svg>
                          {template.variables.length} variable
                          {template.variables.length !== 1 ? 's' : ''}
                        </span>
                        <span className="inline-flex items-center">
                          <svg
                            className="mr-1 h-4 w-4"
                            fill="none"
                            viewBox="0 0 24 24"
                            stroke="currentColor"
                          >
                            <path
                              strokeLinecap="round"
                              strokeLinejoin="round"
                              strokeWidth={2}
                              d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2"
                            />
                          </svg>
                          {template.line_items.length} line item
                          {template.line_items.length !== 1 ? 's' : ''}
                        </span>
                        <span className="inline-flex items-center">
                          <svg
                            className="mr-1 h-4 w-4"
                            fill="none"
                            viewBox="0 0 24 24"
                            stroke="currentColor"
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
                          Used {template.usage_count} time
                          {template.usage_count !== 1 ? 's' : ''}
                        </span>
                      </div>

                      {/* Tags */}
                      {template.tags && template.tags.length > 0 && (
                        <div className="mt-2 flex flex-wrap gap-1">
                          {template.tags.map((tag, index) => (
                            <span
                              key={index}
                              className="inline-flex items-center rounded-full bg-gray-100 px-2 py-0.5 text-xs text-gray-700"
                            >
                              {tag}
                            </span>
                          ))}
                        </div>
                      )}

                      {/* Actions */}
                      <div className="mt-4 flex gap-2">
                        {onSelect && (
                          <button
                            type="button"
                            onClick={() => onSelect(template)}
                            className="rounded-md bg-blue-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-blue-700"
                          >
                            Use Template
                          </button>
                        )}
                        {onEdit && (
                          <button
                            type="button"
                            onClick={() => onEdit(template)}
                            className="rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50"
                          >
                            Edit
                          </button>
                        )}
                        <button
                          type="button"
                          onClick={() => handleDuplicate(template)}
                          disabled={duplicateMutation.isPending}
                          className="rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50"
                        >
                          Duplicate
                        </button>
                        <button
                          type="button"
                          onClick={() => handleDelete(template)}
                          disabled={deleteMutation.isPending}
                          className="rounded-md border border-red-300 bg-white px-3 py-1.5 text-sm font-medium text-red-700 hover:bg-red-50 disabled:opacity-50"
                        >
                          Delete
                        </button>
                      </div>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>

          {/* Pagination */}
          {data.total_pages > 1 && (
            <div className="flex items-center justify-between rounded-lg border bg-white p-4">
              <div className="text-sm text-gray-700">
                Showing{' '}
                <span className="font-medium">
                  {(data.page - 1) * data.page_size + 1}
                </span>{' '}
                to{' '}
                <span className="font-medium">
                  {Math.min(data.page * data.page_size, data.total)}
                </span>{' '}
                of <span className="font-medium">{data.total}</span> results
              </div>
              <div className="flex gap-2">
                <button
                  type="button"
                  onClick={() => handlePageChange(data.page - 1)}
                  disabled={data.page === 1}
                  className="rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50"
                >
                  Previous
                </button>
                <button
                  type="button"
                  onClick={() => handlePageChange(data.page + 1)}
                  disabled={data.page >= data.total_pages}
                  className="rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50"
                >
                  Next
                </button>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
};
