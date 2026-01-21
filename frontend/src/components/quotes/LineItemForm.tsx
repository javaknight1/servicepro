import React, { useState, useCallback, useEffect, useRef } from 'react';
import { LineItemFormData, LineItemValidationErrors } from '../../types/quote';

interface LineItemFormProps {
  onAdd: (item: LineItemFormData) => void;
  onCancel?: () => void;
  initialData?: Partial<LineItemFormData>;
  autoFocus?: boolean;
  showCancel?: boolean;
  buttonText?: string;
}

/**
 * LineItemForm Component
 *
 * Form for adding new line items with validation
 * Features:
 * - Real-time validation
 * - Keyboard shortcuts (Enter to submit, Esc to clear)
 * - Auto-calculate total preview
 * - Pre-populate from templates
 * - Clear form after submit
 */
export const LineItemForm: React.FC<LineItemFormProps> = ({
  onAdd,
  onCancel,
  initialData,
  autoFocus = false,
  showCancel = false,
  buttonText = 'Add Item',
}) => {
  const [formData, setFormData] = useState<LineItemFormData>({
    description: initialData?.description || '',
    quantity: initialData?.quantity || '',
    unit_price: initialData?.unit_price || '',
  });

  const [errors, setErrors] = useState<LineItemValidationErrors>({});
  const [touched, setTouched] = useState<Record<string, boolean>>({});
  const descriptionRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (autoFocus && descriptionRef.current) {
      descriptionRef.current.focus();
    }
  }, [autoFocus]);

  const validateField = (
    field: string,
    value: string | number
  ): string | undefined => {
    const strValue = String(value);

    switch (field) {
      case 'description':
        if (!strValue.trim()) return 'Description is required';
        if (strValue.length > 500)
          return 'Description too long (max 500 characters)';
        return undefined;

      case 'quantity':
        if (strValue === '') return 'Quantity is required';
        const qty = parseFloat(strValue);
        if (isNaN(qty)) return 'Must be a valid number';
        if (qty <= 0) return 'Must be greater than 0';
        if (qty > 999999) return 'Quantity too large';
        return undefined;

      case 'unit_price':
        if (strValue === '') return 'Unit price is required';
        const price = parseFloat(strValue);
        if (isNaN(price)) return 'Must be a valid number';
        if (price < 0) return 'Cannot be negative';
        if (price > 999999) return 'Price too large';
        return undefined;

      default:
        return undefined;
    }
  };

  const handleFieldChange = (field: keyof LineItemFormData, value: string) => {
    setFormData((prev) => ({ ...prev, [field]: value }));

    // Only show errors for touched fields
    if (touched[field]) {
      const error = validateField(field, value);
      setErrors((prev) => ({ ...prev, [field]: error }));
    }
  };

  const handleBlur = (field: keyof LineItemFormData) => {
    setTouched((prev) => ({ ...prev, [field]: true }));
    const error = validateField(field, formData[field]);
    setErrors((prev) => ({ ...prev, [field]: error }));
  };

  const validateForm = (): boolean => {
    const newErrors: LineItemValidationErrors = {
      description: validateField('description', formData.description),
      quantity: validateField('quantity', formData.quantity),
      unit_price: validateField('unit_price', formData.unit_price),
    };

    setErrors(newErrors);
    setTouched({
      description: true,
      quantity: true,
      unit_price: true,
    });

    return !Object.values(newErrors).some((error) => error !== undefined);
  };

  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();

      if (!validateForm()) {
        return;
      }

      onAdd(formData);

      // Reset form
      setFormData({
        description: '',
        quantity: '',
        unit_price: '',
      });
      setErrors({});
      setTouched({});

      // Focus back on description for quick entry
      if (descriptionRef.current) {
        descriptionRef.current.focus();
      }
    },
    [formData, onAdd]
  );

  const handleClear = () => {
    setFormData({
      description: '',
      quantity: '',
      unit_price: '',
    });
    setErrors({});
    setTouched({});

    if (descriptionRef.current) {
      descriptionRef.current.focus();
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') {
      if (onCancel) {
        onCancel();
      } else {
        handleClear();
      }
    }
  };

  // Calculate preview total
  const previewTotal = (() => {
    const qty = parseFloat(String(formData.quantity));
    const price = parseFloat(String(formData.unit_price));

    if (!isNaN(qty) && !isNaN(price)) {
      return (qty * price).toFixed(2);
    }

    return '0.00';
  })();

  const hasValidTotal = parseFloat(previewTotal) > 0;

  return (
    <form
      onSubmit={handleSubmit}
      onKeyDown={handleKeyDown}
      className="bg-white rounded-lg border border-gray-200 p-6"
    >
      <div className="grid grid-cols-1 md:grid-cols-12 gap-4">
        {/* Description */}
        <div className="md:col-span-5">
          <label
            htmlFor="description"
            className="block text-sm font-medium text-gray-700 mb-1"
          >
            Description <span className="text-red-500">*</span>
          </label>
          <input
            ref={descriptionRef}
            type="text"
            id="description"
            value={formData.description}
            onChange={(e) => handleFieldChange('description', e.target.value)}
            onBlur={() => handleBlur('description')}
            className={`
              w-full px-3 py-2 border rounded-md shadow-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500
              ${
                errors.description && touched.description
                  ? 'border-red-500'
                  : 'border-gray-300'
              }
            `}
            placeholder="e.g., Professional Services"
          />
          {errors.description && touched.description && (
            <p className="text-xs text-red-600 mt-1">{errors.description}</p>
          )}
        </div>

        {/* Quantity */}
        <div className="md:col-span-2">
          <label
            htmlFor="quantity"
            className="block text-sm font-medium text-gray-700 mb-1"
          >
            Quantity <span className="text-red-500">*</span>
          </label>
          <input
            type="number"
            id="quantity"
            value={formData.quantity}
            onChange={(e) => handleFieldChange('quantity', e.target.value)}
            onBlur={() => handleBlur('quantity')}
            step="0.01"
            min="0"
            className={`
              w-full px-3 py-2 border rounded-md shadow-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500
              ${
                errors.quantity && touched.quantity
                  ? 'border-red-500'
                  : 'border-gray-300'
              }
            `}
            placeholder="0.00"
          />
          {errors.quantity && touched.quantity && (
            <p className="text-xs text-red-600 mt-1">{errors.quantity}</p>
          )}
        </div>

        {/* Unit Price */}
        <div className="md:col-span-2">
          <label
            htmlFor="unit_price"
            className="block text-sm font-medium text-gray-700 mb-1"
          >
            Unit Price <span className="text-red-500">*</span>
          </label>
          <div className="relative">
            <span className="absolute left-3 top-2 text-gray-500">$</span>
            <input
              type="number"
              id="unit_price"
              value={formData.unit_price}
              onChange={(e) => handleFieldChange('unit_price', e.target.value)}
              onBlur={() => handleBlur('unit_price')}
              step="0.01"
              min="0"
              className={`
                w-full pl-8 pr-3 py-2 border rounded-md shadow-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500
                ${
                  errors.unit_price && touched.unit_price
                    ? 'border-red-500'
                    : 'border-gray-300'
                }
              `}
              placeholder="0.00"
            />
          </div>
          {errors.unit_price && touched.unit_price && (
            <p className="text-xs text-red-600 mt-1">{errors.unit_price}</p>
          )}
        </div>

        {/* Total Preview */}
        <div className="md:col-span-2">
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Total
          </label>
          <div
            className={`
            px-3 py-2 border rounded-md bg-gray-50
            ${hasValidTotal ? 'text-gray-900 font-semibold' : 'text-gray-400'}
          `}
          >
            ${previewTotal}
          </div>
        </div>

        {/* Actions */}
        <div className="md:col-span-1 flex items-end gap-2">
          <button
            type="submit"
            className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 transition-colors shadow-sm font-medium"
            title="Add item (Enter)"
          >
            <svg
              className="w-5 h-5 mx-auto"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 4v16m8-8H4"
              />
            </svg>
          </button>

          {showCancel && onCancel && (
            <button
              type="button"
              onClick={onCancel}
              className="px-4 py-2 border border-gray-300 text-gray-700 rounded-md hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-gray-500 transition-colors"
              title="Cancel (Esc)"
            >
              Cancel
            </button>
          )}
        </div>
      </div>

      {/* Helper text */}
      <div className="mt-3 flex items-center justify-between text-xs text-gray-500">
        <div className="flex items-center gap-4">
          <span>
            Press{' '}
            <kbd className="px-1.5 py-0.5 bg-gray-100 border border-gray-300 rounded">
              Enter
            </kbd>{' '}
            to add
          </span>
          <span>
            Press{' '}
            <kbd className="px-1.5 py-0.5 bg-gray-100 border border-gray-300 rounded">
              Esc
            </kbd>{' '}
            to {onCancel ? 'cancel' : 'clear'}
          </span>
        </div>
        <button
          type="button"
          onClick={handleClear}
          className="text-blue-600 hover:text-blue-700 hover:underline"
        >
          Clear form
        </button>
      </div>
    </form>
  );
};

export default LineItemForm;
