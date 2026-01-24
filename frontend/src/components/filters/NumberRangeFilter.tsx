import React, { useState, useCallback, useRef, useEffect, memo } from 'react';
import { cn } from '../../utils/cn';
import { NumberRangeFilterProps, NumberRange } from '../../types/filter';

// ============================================
// Icons
// ============================================

const ChevronDownIcon: React.FC<{ className?: string }> = ({ className }) => (
  <svg
    className={className}
    fill="none"
    viewBox="0 0 24 24"
    stroke="currentColor"
  >
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M19 9l-7 7-7-7"
    />
  </svg>
);

const XIcon: React.FC<{ className?: string }> = ({ className }) => (
  <svg
    className={className}
    fill="none"
    viewBox="0 0 24 24"
    stroke="currentColor"
  >
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M6 18L18 6M6 6l12 12"
    />
  </svg>
);

// ============================================
// Default Formatter
// ============================================

const defaultFormatValue = (value: number): string => {
  return value.toLocaleString();
};

// ============================================
// Range Input Component
// ============================================

interface RangeInputProps {
  label: string;
  value: number | null;
  onChange: (value: number | null) => void;
  min?: number;
  max?: number;
  step?: number;
  formatValue?: (value: number) => string;
  placeholder?: string;
}

const RangeInput: React.FC<RangeInputProps> = memo(
  ({ label, value, onChange, min, max, step = 1, placeholder }) => {
    const [localValue, setLocalValue] = useState<string>(
      value !== null ? String(value) : ''
    );

    useEffect(() => {
      setLocalValue(value !== null ? String(value) : '');
    }, [value]);

    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      const newValue = e.target.value;
      setLocalValue(newValue);

      if (newValue === '') {
        onChange(null);
      } else {
        const parsed = parseFloat(newValue);
        if (!isNaN(parsed)) {
          onChange(parsed);
        }
      }
    };

    return (
      <div className="flex flex-col gap-1">
        <label className="text-xs font-medium text-gray-500">{label}</label>
        <input
          type="number"
          value={localValue}
          onChange={handleChange}
          min={min}
          max={max}
          step={step}
          placeholder={placeholder}
          className="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
        />
      </div>
    );
  }
);

RangeInput.displayName = 'RangeInput';

// ============================================
// Slider Range Component
// ============================================

interface SliderRangeProps {
  value: NumberRange | null;
  onChange: (range: NumberRange | null) => void;
  min: number;
  max: number;
  step?: number;
  formatValue?: (value: number) => string;
}

const SliderRange: React.FC<SliderRangeProps> = memo(
  ({
    value,
    onChange,
    min,
    max,
    step = 1,
    formatValue = defaultFormatValue,
  }) => {
    const currentMin = value?.min ?? min;
    const currentMax = value?.max ?? max;

    const minPercent = ((currentMin - min) / (max - min)) * 100;
    const maxPercent = ((currentMax - min) / (max - min)) * 100;

    const handleMinChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      const newMin = parseFloat(e.target.value);
      const validMin = Math.min(newMin, currentMax - step);
      onChange({ min: validMin, max: currentMax });
    };

    const handleMaxChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      const newMax = parseFloat(e.target.value);
      const validMax = Math.max(newMax, currentMin + step);
      onChange({ min: currentMin, max: validMax });
    };

    return (
      <div className="space-y-4">
        {/* Display values */}
        <div className="flex justify-between text-sm text-gray-600">
          <span>{formatValue(currentMin)}</span>
          <span>{formatValue(currentMax)}</span>
        </div>

        {/* Slider track */}
        <div className="relative h-2">
          {/* Background track */}
          <div className="absolute inset-0 bg-gray-200 rounded-full" />

          {/* Active range */}
          <div
            className="absolute h-full bg-blue-500 rounded-full"
            style={{
              left: `${minPercent}%`,
              right: `${100 - maxPercent}%`,
            }}
          />

          {/* Min slider */}
          <input
            type="range"
            value={currentMin}
            onChange={handleMinChange}
            min={min}
            max={max}
            step={step}
            className="absolute inset-0 w-full h-full appearance-none bg-transparent pointer-events-auto cursor-pointer [&::-webkit-slider-thumb]:appearance-none [&::-webkit-slider-thumb]:w-4 [&::-webkit-slider-thumb]:h-4 [&::-webkit-slider-thumb]:rounded-full [&::-webkit-slider-thumb]:bg-white [&::-webkit-slider-thumb]:border-2 [&::-webkit-slider-thumb]:border-blue-500 [&::-webkit-slider-thumb]:shadow-md [&::-webkit-slider-thumb]:cursor-pointer [&::-moz-range-thumb]:w-4 [&::-moz-range-thumb]:h-4 [&::-moz-range-thumb]:rounded-full [&::-moz-range-thumb]:bg-white [&::-moz-range-thumb]:border-2 [&::-moz-range-thumb]:border-blue-500 [&::-moz-range-thumb]:shadow-md [&::-moz-range-thumb]:cursor-pointer"
            style={{ zIndex: currentMin > max - 10 ? 5 : 3 }}
          />

          {/* Max slider */}
          <input
            type="range"
            value={currentMax}
            onChange={handleMaxChange}
            min={min}
            max={max}
            step={step}
            className="absolute inset-0 w-full h-full appearance-none bg-transparent pointer-events-auto cursor-pointer [&::-webkit-slider-thumb]:appearance-none [&::-webkit-slider-thumb]:w-4 [&::-webkit-slider-thumb]:h-4 [&::-webkit-slider-thumb]:rounded-full [&::-webkit-slider-thumb]:bg-white [&::-webkit-slider-thumb]:border-2 [&::-webkit-slider-thumb]:border-blue-500 [&::-webkit-slider-thumb]:shadow-md [&::-webkit-slider-thumb]:cursor-pointer [&::-moz-range-thumb]:w-4 [&::-moz-range-thumb]:h-4 [&::-moz-range-thumb]:rounded-full [&::-moz-range-thumb]:bg-white [&::-moz-range-thumb]:border-2 [&::-moz-range-thumb]:border-blue-500 [&::-moz-range-thumb]:shadow-md [&::-moz-range-thumb]:cursor-pointer"
            style={{ zIndex: 4 }}
          />
        </div>

        {/* Min/Max labels */}
        <div className="flex justify-between text-xs text-gray-400">
          <span>{formatValue(min)}</span>
          <span>{formatValue(max)}</span>
        </div>
      </div>
    );
  }
);

SliderRange.displayName = 'SliderRange';

// ============================================
// Main NumberRangeFilter Component
// ============================================

const NumberRangeFilter: React.FC<NumberRangeFilterProps> = ({
  id,
  label,
  value,
  onChange,
  min = 0,
  max = 100,
  step = 1,
  formatValue = defaultFormatValue,
  disabled = false,
  className,
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  // Close on click outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        containerRef.current &&
        !containerRef.current.contains(event.target as Node)
      ) {
        setIsOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleMinChange = useCallback(
    (newMin: number | null) => {
      onChange({
        min: newMin,
        max: value?.max ?? null,
      });
    },
    [value, onChange]
  );

  const handleMaxChange = useCallback(
    (newMax: number | null) => {
      onChange({
        min: value?.min ?? null,
        max: newMax,
      });
    },
    [value, onChange]
  );

  const handleClear = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();
      onChange(null);
    },
    [onChange]
  );

  const hasValue = value !== null && (value.min !== null || value.max !== null);
  const displayValue =
    hasValue && value
      ? `${value.min !== null ? formatValue(value.min) : '...'} - ${
          value.max !== null ? formatValue(value.max) : '...'
        }`
      : 'Any';

  return (
    <div
      ref={containerRef}
      className={cn('relative', className)}
      data-testid={`number-range-filter-${id}`}
    >
      {label && (
        <label className="block text-sm font-medium text-gray-700 mb-1">
          {label}
        </label>
      )}

      {/* Trigger button */}
      <button
        type="button"
        onClick={() => !disabled && setIsOpen(!isOpen)}
        disabled={disabled}
        className={cn(
          'flex items-center justify-between gap-2 w-full px-3 py-2 border rounded-lg text-sm text-left transition-colors',
          disabled
            ? 'bg-gray-100 text-gray-400 cursor-not-allowed'
            : isOpen
              ? 'border-blue-500 ring-2 ring-blue-500'
              : hasValue
                ? 'bg-blue-50 border-blue-300'
                : 'border-gray-300 hover:border-gray-400'
        )}
        data-testid={`number-range-trigger-${id}`}
      >
        <span className={hasValue ? 'text-gray-900' : 'text-gray-400'}>
          {displayValue}
        </span>
        <div className="flex items-center gap-1">
          {hasValue && !disabled && (
            <button
              type="button"
              onClick={handleClear}
              className="p-0.5 hover:bg-gray-200 rounded"
            >
              <XIcon className="w-3 h-3 text-gray-500" />
            </button>
          )}
          <ChevronDownIcon
            className={cn(
              'w-4 h-4 text-gray-400 transition-transform',
              isOpen && 'rotate-180'
            )}
          />
        </div>
      </button>

      {/* Dropdown */}
      {isOpen && (
        <div
          className="absolute z-50 mt-1 w-72 bg-white rounded-lg shadow-lg border border-gray-200 p-4"
          data-testid={`number-range-dropdown-${id}`}
        >
          {/* Input fields */}
          <div className="grid grid-cols-2 gap-4 mb-4">
            <RangeInput
              label="Min"
              value={value?.min ?? null}
              onChange={handleMinChange}
              min={min}
              max={max}
              step={step}
              placeholder={formatValue(min)}
            />
            <RangeInput
              label="Max"
              value={value?.max ?? null}
              onChange={handleMaxChange}
              min={min}
              max={max}
              step={step}
              placeholder={formatValue(max)}
            />
          </div>

          {/* Slider */}
          <SliderRange
            value={value}
            onChange={onChange}
            min={min}
            max={max}
            step={step}
            formatValue={formatValue}
          />

          {/* Quick presets */}
          <div className="flex flex-wrap gap-2 mt-4 pt-4 border-t border-gray-200">
            <button
              type="button"
              onClick={() =>
                onChange({
                  min: min,
                  max: Math.round((max - min) * 0.25) + min,
                })
              }
              className="px-2 py-1 text-xs text-gray-600 hover:bg-gray-100 rounded"
            >
              Low
            </button>
            <button
              type="button"
              onClick={() =>
                onChange({
                  min: Math.round((max - min) * 0.25) + min,
                  max: Math.round((max - min) * 0.75) + min,
                })
              }
              className="px-2 py-1 text-xs text-gray-600 hover:bg-gray-100 rounded"
            >
              Medium
            </button>
            <button
              type="button"
              onClick={() =>
                onChange({
                  min: Math.round((max - min) * 0.75) + min,
                  max: max,
                })
              }
              className="px-2 py-1 text-xs text-gray-600 hover:bg-gray-100 rounded"
            >
              High
            </button>
            <button
              type="button"
              onClick={() => onChange(null)}
              className="px-2 py-1 text-xs text-gray-600 hover:bg-gray-100 rounded ml-auto"
            >
              Clear
            </button>
          </div>
        </div>
      )}
    </div>
  );
};

export default memo(NumberRangeFilter);

// ============================================
// Inline Variant
// ============================================

interface InlineNumberRangeFilterProps extends Omit<
  NumberRangeFilterProps,
  'className'
> {
  className?: string;
}

export const InlineNumberRangeFilter: React.FC<InlineNumberRangeFilterProps> =
  memo(
    ({
      id,
      label,
      value,
      onChange,
      min = 0,
      max = 100,
      step = 1,
      formatValue: _formatValue = defaultFormatValue,
      disabled = false,
      className,
    }) => {
      const handleMinChange = useCallback(
        (newMin: number | null) => {
          onChange({
            min: newMin,
            max: value?.max ?? null,
          });
        },
        [value, onChange]
      );

      const handleMaxChange = useCallback(
        (newMax: number | null) => {
          onChange({
            min: value?.min ?? null,
            max: newMax,
          });
        },
        [value, onChange]
      );

      return (
        <div
          className={cn('space-y-2', className)}
          data-testid={`inline-number-range-${id}`}
        >
          {label && (
            <label className="block text-sm font-medium text-gray-700">
              {label}
            </label>
          )}
          <div className="flex items-center gap-2">
            <input
              type="number"
              value={value?.min ?? ''}
              onChange={(e) =>
                handleMinChange(
                  e.target.value ? parseFloat(e.target.value) : null
                )
              }
              min={min}
              max={max}
              step={step}
              disabled={disabled}
              placeholder="Min"
              className="w-24 px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 disabled:bg-gray-100 disabled:cursor-not-allowed"
            />
            <span className="text-gray-400">to</span>
            <input
              type="number"
              value={value?.max ?? ''}
              onChange={(e) =>
                handleMaxChange(
                  e.target.value ? parseFloat(e.target.value) : null
                )
              }
              min={min}
              max={max}
              step={step}
              disabled={disabled}
              placeholder="Max"
              className="w-24 px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 disabled:bg-gray-100 disabled:cursor-not-allowed"
            />
          </div>
        </div>
      );
    }
  );

InlineNumberRangeFilter.displayName = 'InlineNumberRangeFilter';

// ============================================
// Currency Variant
// ============================================

interface CurrencyRangeFilterProps extends Omit<
  NumberRangeFilterProps,
  'formatValue'
> {
  currency?: string;
  locale?: string;
}

export const CurrencyRangeFilter: React.FC<CurrencyRangeFilterProps> = memo(
  ({ currency = 'USD', locale = 'en-US', ...props }) => {
    const formatCurrency = useCallback(
      (value: number) => {
        return new Intl.NumberFormat(locale, {
          style: 'currency',
          currency,
          minimumFractionDigits: 0,
          maximumFractionDigits: 0,
        }).format(value);
      },
      [currency, locale]
    );

    return <NumberRangeFilter {...props} formatValue={formatCurrency} />;
  }
);

CurrencyRangeFilter.displayName = 'CurrencyRangeFilter';

// ============================================
// Percentage Variant
// ============================================

interface PercentageRangeFilterProps extends Omit<
  NumberRangeFilterProps,
  'formatValue' | 'min' | 'max'
> {
  min?: number;
  max?: number;
}

export const PercentageRangeFilter: React.FC<PercentageRangeFilterProps> = memo(
  ({ min = 0, max = 100, ...props }) => {
    const formatPercent = useCallback((value: number) => `${value}%`, []);

    return (
      <NumberRangeFilter
        {...props}
        min={min}
        max={max}
        formatValue={formatPercent}
      />
    );
  }
);

PercentageRangeFilter.displayName = 'PercentageRangeFilter';
