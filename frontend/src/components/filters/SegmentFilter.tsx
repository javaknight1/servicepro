import React, { useCallback, useMemo, memo } from 'react';
import { cn } from '../../utils/cn';
import { SegmentFilterProps, FilterOption } from '../../types/filter';

// ============================================
// Segment Button Component
// ============================================

interface SegmentButtonProps {
  segment: FilterOption;
  isSelected: boolean;
  showCounts?: boolean;
  disabled?: boolean;
  onClick: () => void;
}

const SegmentButton: React.FC<SegmentButtonProps> = memo(
  ({ segment, isSelected, showCounts = false, disabled = false, onClick }) => {
    return (
      <button
        type="button"
        onClick={onClick}
        disabled={disabled || segment.disabled}
        className={cn(
          'flex items-center gap-2 px-4 py-2 text-sm font-medium transition-all border-y border-r first:border-l first:rounded-l-lg last:rounded-r-lg',
          isSelected
            ? 'bg-blue-600 text-white border-blue-600 z-10'
            : 'bg-white text-gray-700 border-gray-300 hover:bg-gray-50',
          (disabled || segment.disabled) && 'opacity-50 cursor-not-allowed'
        )}
        data-testid={`segment-${segment.value}`}
      >
        {segment.icon && (
          <span className="flex-shrink-0">
            {typeof segment.icon === 'string' ? (
              <span>{segment.icon}</span>
            ) : (
              segment.icon
            )}
          </span>
        )}
        <span>{segment.label}</span>
        {showCounts && segment.count !== undefined && (
          <span
            className={cn(
              'px-1.5 py-0.5 text-xs rounded-full',
              isSelected
                ? 'bg-blue-500 text-white'
                : 'bg-gray-100 text-gray-500'
            )}
          >
            {segment.count}
          </span>
        )}
      </button>
    );
  }
);

SegmentButton.displayName = 'SegmentButton';

// ============================================
// Main SegmentFilter Component
// ============================================

const SegmentFilter: React.FC<SegmentFilterProps> = ({
  id,
  label,
  segments,
  multiple = false,
  value,
  onChange,
  showCounts = false,
  disabled = false,
  className,
}) => {
  const selectedValues = useMemo(
    () =>
      multiple
        ? Array.isArray(value)
          ? value
          : []
        : value !== null
          ? [value]
          : [],
    [multiple, value]
  );

  const handleSegmentClick = useCallback(
    (segmentValue: string | number) => {
      const strValue = String(segmentValue);

      if (multiple) {
        const currentValues = selectedValues as string[];
        const newValues = currentValues.includes(strValue)
          ? currentValues.filter((v) => v !== strValue)
          : [...currentValues, strValue];
        onChange(newValues.length > 0 ? newValues : null);
      } else {
        onChange(value === strValue ? null : strValue);
      }
    },
    [multiple, selectedValues, value, onChange]
  );

  return (
    <div
      className={cn('space-y-1', className)}
      data-testid={`segment-filter-${id}`}
    >
      {label && (
        <label className="block text-sm font-medium text-gray-700">
          {label}
        </label>
      )}
      <div className="inline-flex" role="group">
        {segments.map((segment) => (
          <SegmentButton
            key={String(segment.value)}
            segment={segment}
            isSelected={selectedValues.includes(String(segment.value))}
            showCounts={showCounts}
            disabled={disabled}
            onClick={() => handleSegmentClick(segment.value)}
          />
        ))}
      </div>
    </div>
  );
};

export default memo(SegmentFilter);

// ============================================
// Pill Variant Component
// ============================================

interface PillSegmentFilterProps extends Omit<SegmentFilterProps, 'className'> {
  className?: string;
  size?: 'sm' | 'md' | 'lg';
}

export const PillSegmentFilter: React.FC<PillSegmentFilterProps> = memo(
  ({
    id,
    label,
    segments,
    multiple = false,
    value,
    onChange,
    showCounts = false,
    disabled = false,
    className,
    size = 'md',
  }) => {
    const selectedValues = useMemo(
      () =>
        multiple
          ? Array.isArray(value)
            ? value
            : []
          : value !== null
            ? [value]
            : [],
      [multiple, value]
    );

    const handleSegmentClick = useCallback(
      (segmentValue: string | number) => {
        const strValue = String(segmentValue);

        if (multiple) {
          const currentValues = selectedValues as string[];
          const newValues = currentValues.includes(strValue)
            ? currentValues.filter((v) => v !== strValue)
            : [...currentValues, strValue];
          onChange(newValues.length > 0 ? newValues : null);
        } else {
          onChange(value === strValue ? null : strValue);
        }
      },
      [multiple, selectedValues, value, onChange]
    );

    const sizeClasses = {
      sm: 'px-2.5 py-1 text-xs',
      md: 'px-3 py-1.5 text-sm',
      lg: 'px-4 py-2 text-base',
    };

    return (
      <div
        className={cn('space-y-1', className)}
        data-testid={`pill-segment-filter-${id}`}
      >
        {label && (
          <label className="block text-sm font-medium text-gray-700">
            {label}
          </label>
        )}
        <div className="flex flex-wrap gap-2" role="group">
          {segments.map((segment) => {
            const isSelected = selectedValues.includes(String(segment.value));
            return (
              <button
                key={String(segment.value)}
                type="button"
                onClick={() => handleSegmentClick(segment.value)}
                disabled={disabled || segment.disabled}
                className={cn(
                  'flex items-center gap-1.5 rounded-full font-medium transition-all',
                  sizeClasses[size],
                  isSelected
                    ? 'bg-blue-600 text-white'
                    : 'bg-gray-100 text-gray-700 hover:bg-gray-200',
                  (disabled || segment.disabled) &&
                    'opacity-50 cursor-not-allowed'
                )}
                data-testid={`pill-segment-${segment.value}`}
              >
                {segment.icon && (
                  <span className="flex-shrink-0">
                    {typeof segment.icon === 'string'
                      ? segment.icon
                      : segment.icon}
                  </span>
                )}
                <span>{segment.label}</span>
                {showCounts && segment.count !== undefined && (
                  <span
                    className={cn(
                      'px-1.5 py-0.5 text-xs rounded-full',
                      isSelected ? 'bg-blue-500' : 'bg-gray-200'
                    )}
                  >
                    {segment.count}
                  </span>
                )}
              </button>
            );
          })}
        </div>
      </div>
    );
  }
);

PillSegmentFilter.displayName = 'PillSegmentFilter';

// ============================================
// Tab Variant Component
// ============================================

interface TabSegmentFilterProps extends Omit<
  SegmentFilterProps,
  'multiple' | 'className'
> {
  className?: string;
  underline?: boolean;
}

export const TabSegmentFilter: React.FC<TabSegmentFilterProps> = memo(
  ({
    id,
    label,
    segments,
    value,
    onChange,
    showCounts = false,
    disabled = false,
    className,
    underline = false,
  }) => {
    return (
      <div
        className={cn('space-y-1', className)}
        data-testid={`tab-segment-filter-${id}`}
      >
        {label && (
          <label className="block text-sm font-medium text-gray-700 mb-2">
            {label}
          </label>
        )}
        <div
          className={cn(
            'flex',
            underline
              ? 'border-b border-gray-200'
              : 'bg-gray-100 rounded-lg p-1'
          )}
          role="tablist"
        >
          {segments.map((segment) => {
            const isSelected = value === String(segment.value);
            return (
              <button
                key={String(segment.value)}
                type="button"
                role="tab"
                aria-selected={isSelected}
                onClick={() => onChange(String(segment.value))}
                disabled={disabled || segment.disabled}
                className={cn(
                  'flex items-center justify-center gap-2 px-4 py-2 text-sm font-medium transition-all',
                  underline
                    ? cn(
                        'relative pb-3',
                        isSelected
                          ? 'text-blue-600'
                          : 'text-gray-500 hover:text-gray-700'
                      )
                    : cn(
                        'rounded-md flex-1',
                        isSelected
                          ? 'bg-white text-gray-900 shadow-sm'
                          : 'text-gray-500 hover:text-gray-700'
                      ),
                  (disabled || segment.disabled) &&
                    'opacity-50 cursor-not-allowed'
                )}
                data-testid={`tab-segment-${segment.value}`}
              >
                {segment.icon && (
                  <span className="flex-shrink-0">
                    {typeof segment.icon === 'string'
                      ? segment.icon
                      : segment.icon}
                  </span>
                )}
                <span>{segment.label}</span>
                {showCounts && segment.count !== undefined && (
                  <span
                    className={cn(
                      'px-1.5 py-0.5 text-xs rounded-full',
                      isSelected
                        ? 'bg-blue-100 text-blue-600'
                        : 'bg-gray-200 text-gray-500'
                    )}
                  >
                    {segment.count}
                  </span>
                )}
                {underline && isSelected && (
                  <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-600" />
                )}
              </button>
            );
          })}
        </div>
      </div>
    );
  }
);

TabSegmentFilter.displayName = 'TabSegmentFilter';

// ============================================
// Color Segment Variant
// ============================================

interface ColorSegmentFilterProps extends Omit<
  SegmentFilterProps,
  'className'
> {
  className?: string;
}

export const ColorSegmentFilter: React.FC<ColorSegmentFilterProps> = memo(
  ({
    id,
    label,
    segments,
    multiple = false,
    value,
    onChange,
    showCounts = false,
    disabled = false,
    className,
  }) => {
    const selectedValues = useMemo(
      () =>
        multiple
          ? Array.isArray(value)
            ? value
            : []
          : value !== null
            ? [value]
            : [],
      [multiple, value]
    );

    const handleSegmentClick = useCallback(
      (segmentValue: string | number) => {
        const strValue = String(segmentValue);

        if (multiple) {
          const currentValues = selectedValues as string[];
          const newValues = currentValues.includes(strValue)
            ? currentValues.filter((v) => v !== strValue)
            : [...currentValues, strValue];
          onChange(newValues.length > 0 ? newValues : null);
        } else {
          onChange(value === strValue ? null : strValue);
        }
      },
      [multiple, selectedValues, value, onChange]
    );

    return (
      <div
        className={cn('space-y-2', className)}
        data-testid={`color-segment-filter-${id}`}
      >
        {label && (
          <label className="block text-sm font-medium text-gray-700">
            {label}
          </label>
        )}
        <div className="flex flex-wrap gap-2" role="group">
          {segments.map((segment) => {
            const isSelected = selectedValues.includes(String(segment.value));
            const color = segment.color || '#3B82F6';

            return (
              <button
                key={String(segment.value)}
                type="button"
                onClick={() => handleSegmentClick(segment.value)}
                disabled={disabled || segment.disabled}
                className={cn(
                  'flex items-center gap-2 px-3 py-1.5 rounded-full text-sm font-medium transition-all border-2',
                  isSelected ? 'text-white' : 'bg-white',
                  (disabled || segment.disabled) &&
                    'opacity-50 cursor-not-allowed'
                )}
                style={{
                  borderColor: color,
                  backgroundColor: isSelected ? color : 'white',
                  color: isSelected ? 'white' : color,
                }}
                data-testid={`color-segment-${segment.value}`}
              >
                {segment.icon && (
                  <span className="flex-shrink-0">
                    {typeof segment.icon === 'string'
                      ? segment.icon
                      : segment.icon}
                  </span>
                )}
                <span>{segment.label}</span>
                {showCounts && segment.count !== undefined && (
                  <span
                    className="px-1.5 py-0.5 text-xs rounded-full"
                    style={{
                      backgroundColor: isSelected
                        ? 'rgba(255,255,255,0.2)'
                        : `${color}20`,
                    }}
                  >
                    {segment.count}
                  </span>
                )}
              </button>
            );
          })}
        </div>
      </div>
    );
  }
);

ColorSegmentFilter.displayName = 'ColorSegmentFilter';
