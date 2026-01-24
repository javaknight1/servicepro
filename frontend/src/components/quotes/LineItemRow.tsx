import React, { useState, useCallback, useRef, useEffect } from 'react';
import { LineItem, LineItemValidationErrors } from '../../types/quote';

interface LineItemRowProps {
  item: LineItem;
  index: number;
  onUpdate: (id: string, updates: Partial<LineItem>) => void;
  onDelete: (id: string) => void;
  onDuplicate?: (id: string) => void;
  isSelected?: boolean;
  onSelect?: (id: string, selected: boolean) => void;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  dragHandleProps?: any;
  isDragging?: boolean;
  isEditable?: boolean;
}

/**
 * LineItemRow Component
 *
 * Displays a single line item with inline editing capabilities
 * Features:
 * - Inline edit mode
 * - Real-time validation
 * - Delete confirmation
 * - Drag handle for reordering
 * - Checkbox for bulk selection
 * - Quick duplicate action
 */
export const LineItemRow: React.FC<LineItemRowProps> = ({
  item,
  index,
  onUpdate,
  onDelete,
  onDuplicate,
  isSelected = false,
  onSelect,
  dragHandleProps,
  isDragging = false,
  isEditable = true,
}) => {
  const [isEditing, setIsEditing] = useState(false);
  const [editValues, setEditValues] = useState({
    description: item.description,
    quantity: item.quantity.toString(),
    unit_price: item.unit_price.toString(),
  });
  const [errors, setErrors] = useState<LineItemValidationErrors>({});
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const descriptionRef = useRef<HTMLInputElement>(null);

  // Focus description field when entering edit mode
  useEffect(() => {
    if (isEditing && descriptionRef.current) {
      descriptionRef.current.focus();
    }
  }, [isEditing]);

  const validateField = (field: string, value: string): string | undefined => {
    switch (field) {
      case 'description':
        if (!value.trim()) return 'Description is required';
        if (value.length > 500)
          return 'Description too long (max 500 characters)';
        return undefined;
      case 'quantity':
        const qty = parseFloat(value);
        if (isNaN(qty)) return 'Must be a valid number';
        if (qty <= 0) return 'Must be greater than 0';
        if (qty > 999999) return 'Quantity too large';
        return undefined;
      case 'unit_price':
        const price = parseFloat(value);
        if (isNaN(price)) return 'Must be a valid number';
        if (price < 0) return 'Cannot be negative';
        if (price > 999999) return 'Price too large';
        return undefined;
      default:
        return undefined;
    }
  };

  const handleFieldChange = (field: string, value: string) => {
    setEditValues((prev) => ({ ...prev, [field]: value }));
    const error = validateField(field, value);
    setErrors((prev) => ({ ...prev, [field]: error }));
  };

  const handleSave = useCallback(() => {
    // Validate all fields
    const newErrors: LineItemValidationErrors = {
      description: validateField('description', editValues.description),
      quantity: validateField('quantity', editValues.quantity),
      unit_price: validateField('unit_price', editValues.unit_price),
    };

    setErrors(newErrors);

    // Check if there are any errors
    if (Object.values(newErrors).some((error) => error !== undefined)) {
      return;
    }

    // Update item
    onUpdate(item.id, {
      description: editValues.description.trim(),
      quantity: parseFloat(editValues.quantity),
      unit_price: parseFloat(editValues.unit_price),
      total:
        parseFloat(editValues.quantity) * parseFloat(editValues.unit_price),
    });

    setIsEditing(false);
  }, [editValues, item.id, onUpdate]);

  const handleCancel = () => {
    setEditValues({
      description: item.description,
      quantity: item.quantity.toString(),
      unit_price: item.unit_price.toString(),
    });
    setErrors({});
    setIsEditing(false);
  };

  const handleDelete = () => {
    if (showDeleteConfirm) {
      onDelete(item.id);
      setShowDeleteConfirm(false);
    } else {
      setShowDeleteConfirm(true);
      // Auto-hide confirmation after 3 seconds
      setTimeout(() => setShowDeleteConfirm(false), 3000);
    }
  };

  const formatCurrency = (amount: number): string => {
    return `$${amount.toFixed(2)}`;
  };

  const itemTotal = Number(item.quantity) * Number(item.unit_price);

  return (
    <tr
      className={`
        group hover:bg-gray-50 transition-colors
        ${isDragging ? 'opacity-50' : 'opacity-100'}
        ${isSelected ? 'bg-blue-50' : ''}
      `}
    >
      {/* Drag Handle */}
      <td className="px-2 py-3 w-8">
        <div
          {...dragHandleProps}
          className="cursor-grab active:cursor-grabbing text-gray-400 hover:text-gray-600"
          title="Drag to reorder"
        >
          <svg
            className="w-5 h-5"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M4 8h16M4 16h16"
            />
          </svg>
        </div>
      </td>

      {/* Checkbox */}
      {onSelect && (
        <td className="px-2 py-3 w-12">
          <input
            type="checkbox"
            checked={isSelected}
            onChange={(e) => onSelect(item.id, e.target.checked)}
            className="w-4 h-4 text-blue-600 rounded border-gray-300 focus:ring-blue-500"
          />
        </td>
      )}

      {/* Index */}
      <td className="px-4 py-3 w-12 text-sm text-gray-500 font-medium">
        {index + 1}
      </td>

      {/* Description */}
      <td className="px-4 py-3 flex-1 min-w-[200px]">
        {isEditing ? (
          <div>
            <input
              ref={descriptionRef}
              type="text"
              value={editValues.description}
              onChange={(e) => handleFieldChange('description', e.target.value)}
              className={`
                w-full px-3 py-2 border rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500
                ${errors.description ? 'border-red-500' : 'border-gray-300'}
              `}
              placeholder="Item description"
            />
            {errors.description && (
              <p className="text-xs text-red-600 mt-1">{errors.description}</p>
            )}
          </div>
        ) : (
          <span className="text-sm text-gray-900">{item.description}</span>
        )}
      </td>

      {/* Quantity */}
      <td className="px-4 py-3 w-32">
        {isEditing ? (
          <div>
            <input
              type="number"
              value={editValues.quantity}
              onChange={(e) => handleFieldChange('quantity', e.target.value)}
              step="0.01"
              min="0"
              className={`
                w-full px-3 py-2 border rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500
                ${errors.quantity ? 'border-red-500' : 'border-gray-300'}
              `}
              placeholder="0.00"
            />
            {errors.quantity && (
              <p className="text-xs text-red-600 mt-1">{errors.quantity}</p>
            )}
          </div>
        ) : (
          <span className="text-sm text-gray-900">{item.quantity}</span>
        )}
      </td>

      {/* Unit Price */}
      <td className="px-4 py-3 w-32">
        {isEditing ? (
          <div>
            <input
              type="number"
              value={editValues.unit_price}
              onChange={(e) => handleFieldChange('unit_price', e.target.value)}
              step="0.01"
              min="0"
              className={`
                w-full px-3 py-2 border rounded-md focus:ring-2 focus:ring-blue-500 focus:border-blue-500
                ${errors.unit_price ? 'border-red-500' : 'border-gray-300'}
              `}
              placeholder="0.00"
            />
            {errors.unit_price && (
              <p className="text-xs text-red-600 mt-1">{errors.unit_price}</p>
            )}
          </div>
        ) : (
          <span className="text-sm text-gray-900">
            {formatCurrency(item.unit_price)}
          </span>
        )}
      </td>

      {/* Total */}
      <td className="px-4 py-3 w-32">
        <span className="text-sm font-semibold text-gray-900">
          {formatCurrency(itemTotal)}
        </span>
      </td>

      {/* Actions */}
      <td className="px-4 py-3 w-48">
        <div className="flex items-center gap-2 justify-end">
          {isEditing ? (
            <>
              <button
                onClick={handleSave}
                className="px-3 py-1.5 text-xs font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                Save
              </button>
              <button
                onClick={handleCancel}
                className="px-3 py-1.5 text-xs font-medium text-gray-700 bg-gray-100 rounded-md hover:bg-gray-200 focus:outline-none focus:ring-2 focus:ring-gray-500"
              >
                Cancel
              </button>
            </>
          ) : (
            <>
              {isEditable && (
                <button
                  onClick={() => setIsEditing(true)}
                  className="p-1.5 text-gray-600 hover:text-blue-600 hover:bg-blue-50 rounded-md transition-colors"
                  title="Edit"
                >
                  <svg
                    className="w-4 h-4"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                    />
                  </svg>
                </button>
              )}

              {onDuplicate && (
                <button
                  onClick={() => onDuplicate(item.id)}
                  className="p-1.5 text-gray-600 hover:text-green-600 hover:bg-green-50 rounded-md transition-colors"
                  title="Duplicate"
                >
                  <svg
                    className="w-4 h-4"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
                    />
                  </svg>
                </button>
              )}

              {isEditable && (
                <button
                  onClick={handleDelete}
                  className={`
                    p-1.5 rounded-md transition-colors
                    ${
                      showDeleteConfirm
                        ? 'text-white bg-red-600 hover:bg-red-700'
                        : 'text-gray-600 hover:text-red-600 hover:bg-red-50'
                    }
                  `}
                  title={
                    showDeleteConfirm ? 'Click again to confirm' : 'Delete'
                  }
                >
                  <svg
                    className="w-4 h-4"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                    />
                  </svg>
                </button>
              )}
            </>
          )}
        </div>
      </td>
    </tr>
  );
};

export default LineItemRow;
