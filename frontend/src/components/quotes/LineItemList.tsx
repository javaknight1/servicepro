import React, { useState, useCallback } from 'react';
import { LineItem, LineItemFormData } from '../../types/quote';
import LineItemRow from './LineItemRow';
import LineItemForm from './LineItemForm';
import TotalCalculator from './TotalCalculator';

interface LineItemListProps {
  items: LineItem[];
  taxRate: number;
  onChange: (items: LineItem[]) => void;
  isEditable?: boolean;
  showForm?: boolean;
  showTotals?: boolean;
  showBulkActions?: boolean;
}

/**
 * LineItemList Component
 *
 * Main component for managing quote line items
 * Features:
 * - Add/edit/delete items
 * - Drag-and-drop reordering
 * - Bulk selection and actions
 * - Real-time total calculation
 * - Keyboard shortcuts
 */
export const LineItemList: React.FC<LineItemListProps> = ({
  items,
  taxRate,
  onChange,
  isEditable = true,
  showForm = true,
  showTotals = true,
  showBulkActions = true,
}) => {
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [draggedIndex, setDraggedIndex] = useState<number | null>(null);
  const [dragOverIndex, setDragOverIndex] = useState<number | null>(null);

  // Generate unique ID for new items
  const generateId = (): string => {
    return `temp-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
  };

  // Add new item
  const handleAddItem = useCallback(
    (formData: LineItemFormData) => {
      const newItem: LineItem = {
        id: generateId(),
        description: formData.description,
        quantity: Number(formData.quantity),
        unit_price: Number(formData.unit_price),
        total: Number(formData.quantity) * Number(formData.unit_price),
        sort_order: items.length,
      };

      onChange([...items, newItem]);
    },
    [items, onChange]
  );

  // Update existing item
  const handleUpdateItem = useCallback(
    (id: string, updates: Partial<LineItem>) => {
      const updatedItems = items.map((item) =>
        item.id === id ? { ...item, ...updates } : item
      );
      onChange(updatedItems);
    },
    [items, onChange]
  );

  // Delete item
  const handleDeleteItem = useCallback(
    (id: string) => {
      const updatedItems = items.filter((item) => item.id !== id);
      // Update sort_order
      const reorderedItems = updatedItems.map((item, index) => ({
        ...item,
        sort_order: index,
      }));
      onChange(reorderedItems);
      setSelectedIds((prev) => {
        const newSet = new Set(prev);
        newSet.delete(id);
        return newSet;
      });
    },
    [items, onChange]
  );

  // Duplicate item
  const handleDuplicateItem = useCallback(
    (id: string) => {
      const itemToDuplicate = items.find((item) => item.id === id);
      if (!itemToDuplicate) return;

      const duplicatedItem: LineItem = {
        ...itemToDuplicate,
        id: generateId(),
        description: `${itemToDuplicate.description} (Copy)`,
        sort_order: items.length,
      };

      onChange([...items, duplicatedItem]);
    },
    [items, onChange]
  );

  // Selection handlers
  const handleSelectItem = useCallback((id: string, selected: boolean) => {
    setSelectedIds((prev) => {
      const newSet = new Set(prev);
      if (selected) {
        newSet.add(id);
      } else {
        newSet.delete(id);
      }
      return newSet;
    });
  }, []);

  const handleSelectAll = useCallback(
    (selected: boolean) => {
      if (selected) {
        setSelectedIds(new Set(items.map((item) => item.id)));
      } else {
        setSelectedIds(new Set());
      }
    },
    [items]
  );

  // Bulk actions
  const handleBulkDelete = useCallback(() => {
    if (selectedIds.size === 0) return;

    if (!confirm(`Delete ${selectedIds.size} selected item(s)?`)) return;

    const updatedItems = items
      .filter((item) => !selectedIds.has(item.id))
      .map((item, index) => ({ ...item, sort_order: index }));

    onChange(updatedItems);
    setSelectedIds(new Set());
  }, [items, selectedIds, onChange]);

  const handleBulkDuplicate = useCallback(() => {
    if (selectedIds.size === 0) return;

    const itemsToDuplicate = items.filter((item) => selectedIds.has(item.id));
    const duplicatedItems = itemsToDuplicate.map((item, index) => ({
      ...item,
      id: generateId(),
      description: `${item.description} (Copy)`,
      sort_order: items.length + index,
    }));

    onChange([...items, ...duplicatedItems]);
    setSelectedIds(new Set());
  }, [items, selectedIds, onChange]);

  // Drag and drop handlers
  const handleDragStart = (e: React.DragEvent, index: number) => {
    setDraggedIndex(index);
    e.dataTransfer.effectAllowed = 'move';
  };

  const handleDragOver = (e: React.DragEvent, index: number) => {
    e.preventDefault();
    if (draggedIndex === null || draggedIndex === index) return;
    setDragOverIndex(index);
  };

  const handleDragEnd = () => {
    if (draggedIndex === null || dragOverIndex === null) {
      setDraggedIndex(null);
      setDragOverIndex(null);
      return;
    }

    const newItems = [...items];
    const [removed] = newItems.splice(draggedIndex, 1);
    newItems.splice(dragOverIndex, 0, removed);

    // Update sort_order
    const reorderedItems = newItems.map((item, index) => ({
      ...item,
      sort_order: index,
    }));

    onChange(reorderedItems);
    setDraggedIndex(null);
    setDragOverIndex(null);
  };

  const allSelected = items.length > 0 && selectedIds.size === items.length;
  const someSelected = selectedIds.size > 0 && selectedIds.size < items.length;

  return (
    <div className="space-y-6">
      {/* Add Item Form */}
      {showForm && isEditable && (
        <div>
          <h3 className="text-lg font-semibold text-gray-900 mb-4">
            Add Line Item
          </h3>
          <LineItemForm onAdd={handleAddItem} autoFocus />
        </div>
      )}

      {/* Items Table */}
      {items.length > 0 ? (
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <h3 className="text-lg font-semibold text-gray-900">
              Line Items ({items.length})
            </h3>

            {/* Bulk Actions */}
            {showBulkActions && isEditable && selectedIds.size > 0 && (
              <div className="flex items-center gap-3">
                <span className="text-sm text-gray-600">
                  {selectedIds.size} selected
                </span>
                <button
                  onClick={handleBulkDuplicate}
                  className="px-3 py-1.5 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                  Duplicate
                </button>
                <button
                  onClick={handleBulkDelete}
                  className="px-3 py-1.5 text-sm font-medium text-white bg-red-600 rounded-md hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-red-500"
                >
                  Delete
                </button>
              </div>
            )}
          </div>

          <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-2 py-3 w-8"></th>
                    {showBulkActions && (
                      <th className="px-2 py-3 w-12">
                        <input
                          type="checkbox"
                          checked={allSelected}
                          ref={(input) => {
                            if (input) input.indeterminate = someSelected;
                          }}
                          onChange={(e) => handleSelectAll(e.target.checked)}
                          className="w-4 h-4 text-blue-600 rounded border-gray-300 focus:ring-blue-500"
                        />
                      </th>
                    )}
                    <th className="px-4 py-3 text-left text-xs font-semibold text-gray-700 uppercase tracking-wider w-12">
                      #
                    </th>
                    <th className="px-4 py-3 text-left text-xs font-semibold text-gray-700 uppercase tracking-wider min-w-[200px]">
                      Description
                    </th>
                    <th className="px-4 py-3 text-left text-xs font-semibold text-gray-700 uppercase tracking-wider w-32">
                      Quantity
                    </th>
                    <th className="px-4 py-3 text-left text-xs font-semibold text-gray-700 uppercase tracking-wider w-32">
                      Unit Price
                    </th>
                    <th className="px-4 py-3 text-left text-xs font-semibold text-gray-700 uppercase tracking-wider w-32">
                      Total
                    </th>
                    <th className="px-4 py-3 text-right text-xs font-semibold text-gray-700 uppercase tracking-wider w-48">
                      Actions
                    </th>
                  </tr>
                </thead>
                <tbody className="bg-white divide-y divide-gray-200">
                  {items.map((item, index) => (
                    <LineItemRow
                      key={item.id}
                      item={item}
                      index={index}
                      onUpdate={handleUpdateItem}
                      onDelete={handleDeleteItem}
                      onDuplicate={handleDuplicateItem}
                      isSelected={selectedIds.has(item.id)}
                      onSelect={showBulkActions ? handleSelectItem : undefined}
                      isDragging={draggedIndex === index}
                      isEditable={isEditable}
                      dragHandleProps={{
                        draggable: true,
                        onDragStart: (e: React.DragEvent) =>
                          handleDragStart(e, index),
                        onDragOver: (e: React.DragEvent) =>
                          handleDragOver(e, index),
                        onDragEnd: handleDragEnd,
                      }}
                    />
                  ))}
                </tbody>
              </table>
            </div>
          </div>

          {/* Keyboard Shortcuts Help */}
          <div className="text-xs text-gray-500 flex items-center gap-4">
            <span>💡 Tip: Drag rows to reorder</span>
            {showBulkActions && (
              <span>• Select multiple items for bulk actions</span>
            )}
          </div>
        </div>
      ) : (
        !showForm && (
          <div className="bg-white rounded-lg border-2 border-dashed border-gray-300 p-12 text-center">
            <svg
              className="mx-auto h-12 w-12 text-gray-400"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              aria-hidden="true"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M9 13h6m-3-3v6m5 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
              />
            </svg>
            <h3 className="mt-2 text-sm font-medium text-gray-900">
              No line items
            </h3>
            <p className="mt-1 text-sm text-gray-500">
              Get started by adding a line item.
            </p>
          </div>
        )
      )}

      {/* Total Calculator */}
      {showTotals && (
        <div className="lg:w-96 ml-auto">
          <TotalCalculator
            items={items}
            taxRate={taxRate}
            showBreakdown={items.length > 0 && items.length <= 10}
          />
        </div>
      )}
    </div>
  );
};

export default LineItemList;
