import React, { useState, useRef, useEffect, useMemo, memo } from 'react';
import { cn } from '../../utils/cn';
import {
  FilterBarProps,
  FilterConfig,
  FilterValue,
  ActiveFilter,
  DateRange,
  formatFilterValue,
} from '../../types/filter';
import { FilterProvider, useFilters, useFilter } from './FilterState';
import DateRangePicker from './DateRangePicker';

// ============================================
// Icons
// ============================================

const SearchIcon: React.FC<{ className?: string }> = ({ className }) => (
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
      d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
    />
  </svg>
);

const FilterIcon: React.FC<{ className?: string }> = ({ className }) => (
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
      d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z"
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

const CheckIcon: React.FC<{ className?: string }> = ({ className }) => (
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
      d="M5 13l4 4L19 7"
    />
  </svg>
);

// ============================================
// Search Input Component
// ============================================

interface SearchInputProps {
  placeholder?: string;
  className?: string;
}

const SearchInput: React.FC<SearchInputProps> = memo(
  ({ placeholder = 'Search...', className }) => {
    const { state, setSearch } = useFilters();
    const [localValue, setLocalValue] = useState(state.searchQuery);
    const debounceRef = useRef<NodeJS.Timeout | null>(null);

    useEffect(() => {
      setLocalValue(state.searchQuery);
    }, [state.searchQuery]);

    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      const value = e.target.value;
      setLocalValue(value);

      if (debounceRef.current) {
        clearTimeout(debounceRef.current);
      }

      debounceRef.current = setTimeout(() => {
        setSearch(value);
      }, 300);
    };

    const handleClear = () => {
      setLocalValue('');
      setSearch('');
    };

    return (
      <div className={cn('relative', className)}>
        <SearchIcon className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" />
        <input
          type="text"
          value={localValue}
          onChange={handleChange}
          placeholder={placeholder}
          className="w-full pl-10 pr-10 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
          data-testid="filter-search-input"
        />
        {localValue && (
          <button
            type="button"
            onClick={handleClear}
            className="absolute right-3 top-1/2 -translate-y-1/2 p-0.5 hover:bg-gray-100 rounded"
            data-testid="filter-search-clear"
          >
            <XIcon className="w-4 h-4 text-gray-400" />
          </button>
        )}
      </div>
    );
  }
);

SearchInput.displayName = 'SearchInput';

// ============================================
// Select Filter Component
// ============================================

interface SelectFilterComponentProps {
  config: FilterConfig;
  value: FilterValue;
  onChange: (value: FilterValue) => void;
}

const SelectFilterComponent: React.FC<SelectFilterComponentProps> = memo(
  ({ config, value, onChange }) => {
    const [isOpen, setIsOpen] = useState(false);
    const [searchQuery, setSearchQuery] = useState('');
    const containerRef = useRef<HTMLDivElement>(null);
    const inputRef = useRef<HTMLInputElement>(null);

    const options = useMemo(() => config.options || [], [config.options]);
    const isMultiple = config.type === 'multiselect';
    const selectedValues = isMultiple
      ? Array.isArray(value)
        ? value
        : []
      : value !== null
        ? [value]
        : [];

    // Close on click outside
    useEffect(() => {
      const handleClickOutside = (event: MouseEvent) => {
        if (
          containerRef.current &&
          !containerRef.current.contains(event.target as Node)
        ) {
          setIsOpen(false);
          setSearchQuery('');
        }
      };

      document.addEventListener('mousedown', handleClickOutside);
      return () =>
        document.removeEventListener('mousedown', handleClickOutside);
    }, []);

    // Focus input when opening
    useEffect(() => {
      if (isOpen && config.searchable && inputRef.current) {
        inputRef.current.focus();
      }
    }, [isOpen, config.searchable]);

    const filteredOptions = useMemo(() => {
      if (!searchQuery) return options;
      const query = searchQuery.toLowerCase();
      return options.filter((opt) => opt.label.toLowerCase().includes(query));
    }, [options, searchQuery]);

    const handleOptionClick = (optionValue: string | number) => {
      if (isMultiple) {
        const currentValues = selectedValues as (string | number)[];
        const newValues = currentValues.includes(optionValue)
          ? currentValues.filter((v) => v !== optionValue)
          : [...currentValues, optionValue];
        onChange(newValues.length > 0 ? (newValues as FilterValue) : null);
      } else {
        onChange(optionValue === value ? null : optionValue);
        setIsOpen(false);
      }
      setSearchQuery('');
    };

    const getDisplayValue = () => {
      if (selectedValues.length === 0) return config.placeholder || 'Select...';
      if (selectedValues.length === 1) {
        const option = options.find((o) => o.value === selectedValues[0]);
        return option?.label || String(selectedValues[0]);
      }
      return `${selectedValues.length} selected`;
    };

    const handleClear = (e: React.MouseEvent) => {
      e.stopPropagation();
      onChange(null);
    };

    return (
      <div
        ref={containerRef}
        className="relative"
        data-testid={`filter-select-${config.id}`}
      >
        <button
          type="button"
          onClick={() => setIsOpen(!isOpen)}
          className={cn(
            'flex items-center justify-between gap-2 px-3 py-2 border rounded-lg text-sm min-w-[160px] transition-colors',
            isOpen
              ? 'border-blue-500 ring-2 ring-blue-500'
              : 'border-gray-300 hover:border-gray-400',
            selectedValues.length > 0
              ? 'bg-blue-50 border-blue-300'
              : 'bg-white'
          )}
        >
          <span
            className={
              selectedValues.length > 0 ? 'text-gray-900' : 'text-gray-400'
            }
          >
            {getDisplayValue()}
          </span>
          <div className="flex items-center gap-1">
            {selectedValues.length > 0 && config.clearable !== false && (
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

        {isOpen && (
          <div className="absolute z-50 mt-1 w-full min-w-[200px] bg-white rounded-lg shadow-lg border border-gray-200 py-1">
            {config.searchable && (
              <div className="px-2 pb-2 pt-1">
                <input
                  ref={inputRef}
                  type="text"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  placeholder="Search..."
                  className="w-full px-2 py-1.5 text-sm border border-gray-300 rounded focus:outline-none focus:ring-1 focus:ring-blue-500"
                />
              </div>
            )}

            <div className="max-h-60 overflow-y-auto">
              {filteredOptions.length === 0 ? (
                <div className="px-3 py-2 text-sm text-gray-500">
                  No options found
                </div>
              ) : (
                filteredOptions.map((option) => {
                  const isSelected = selectedValues.includes(option.value);
                  return (
                    <button
                      key={String(option.value)}
                      type="button"
                      onClick={() => handleOptionClick(option.value)}
                      disabled={option.disabled}
                      className={cn(
                        'w-full flex items-center gap-2 px-3 py-2 text-sm text-left transition-colors',
                        isSelected
                          ? 'bg-blue-50 text-blue-700'
                          : 'text-gray-700 hover:bg-gray-50',
                        option.disabled && 'opacity-50 cursor-not-allowed'
                      )}
                    >
                      {isMultiple && (
                        <div
                          className={cn(
                            'w-4 h-4 border rounded flex items-center justify-center',
                            isSelected
                              ? 'bg-blue-600 border-blue-600'
                              : 'border-gray-300 bg-white'
                          )}
                        >
                          {isSelected && (
                            <CheckIcon className="w-3 h-3 text-white" />
                          )}
                        </div>
                      )}
                      {option.icon && (
                        <span className="flex-shrink-0">
                          {typeof option.icon === 'string' ? (
                            <span className="text-lg">{option.icon}</span>
                          ) : (
                            option.icon
                          )}
                        </span>
                      )}
                      <span className="flex-1">{option.label}</span>
                      {option.count !== undefined && (
                        <span className="text-xs text-gray-400">
                          {option.count}
                        </span>
                      )}
                      {!isMultiple && isSelected && (
                        <CheckIcon className="w-4 h-4 text-blue-600" />
                      )}
                    </button>
                  );
                })
              )}
            </div>
          </div>
        )}
      </div>
    );
  }
);

SelectFilterComponent.displayName = 'SelectFilterComponent';

// ============================================
// Text Filter Component
// ============================================

interface TextFilterComponentProps {
  config: FilterConfig;
  value: FilterValue;
  onChange: (value: FilterValue) => void;
}

const TextFilterComponent: React.FC<TextFilterComponentProps> = memo(
  ({ config, value, onChange }) => {
    const [localValue, setLocalValue] = useState(String(value || ''));
    const debounceRef = useRef<NodeJS.Timeout | null>(null);

    useEffect(() => {
      setLocalValue(String(value || ''));
    }, [value]);

    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      const newValue = e.target.value;
      setLocalValue(newValue);

      if (debounceRef.current) {
        clearTimeout(debounceRef.current);
      }

      debounceRef.current = setTimeout(() => {
        onChange(newValue || null);
      }, 300);
    };

    return (
      <input
        type="text"
        value={localValue}
        onChange={handleChange}
        placeholder={config.placeholder}
        className="px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 min-w-[150px]"
        data-testid={`filter-text-${config.id}`}
      />
    );
  }
);

TextFilterComponent.displayName = 'TextFilterComponent';

// ============================================
// Number Filter Component
// ============================================

interface NumberFilterComponentProps {
  config: FilterConfig;
  value: FilterValue;
  onChange: (value: FilterValue) => void;
}

const NumberFilterComponent: React.FC<NumberFilterComponentProps> = memo(
  ({ config, value, onChange }) => {
    const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      const newValue = e.target.value ? parseFloat(e.target.value) : null;
      onChange(newValue);
    };

    return (
      <input
        type="number"
        value={value !== null ? String(value) : ''}
        onChange={handleChange}
        placeholder={config.placeholder}
        min={config.min as number}
        max={config.max as number}
        step={config.step}
        className="px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 w-32"
        data-testid={`filter-number-${config.id}`}
      />
    );
  }
);

NumberFilterComponent.displayName = 'NumberFilterComponent';

// ============================================
// Boolean Filter Component
// ============================================

interface BooleanFilterComponentProps {
  config: FilterConfig;
  value: FilterValue;
  onChange: (value: FilterValue) => void;
}

const BooleanFilterComponent: React.FC<BooleanFilterComponentProps> = memo(
  ({ config, value, onChange }) => {
    const handleChange = () => {
      if (value === null) onChange(true);
      else if (value === true) onChange(false);
      else onChange(null);
    };

    return (
      <button
        type="button"
        onClick={handleChange}
        className={cn(
          'flex items-center gap-2 px-3 py-2 border rounded-lg text-sm transition-colors',
          value === true
            ? 'bg-green-50 border-green-300 text-green-700'
            : value === false
              ? 'bg-red-50 border-red-300 text-red-700'
              : 'border-gray-300 text-gray-500 hover:border-gray-400'
        )}
        data-testid={`filter-boolean-${config.id}`}
      >
        <div
          className={cn(
            'w-4 h-4 rounded flex items-center justify-center',
            value === true
              ? 'bg-green-600'
              : value === false
                ? 'bg-red-600'
                : 'border border-gray-300'
          )}
        >
          {value === true && <CheckIcon className="w-3 h-3 text-white" />}
          {value === false && <XIcon className="w-3 h-3 text-white" />}
        </div>
        <span>
          {config.label}:{' '}
          {value === null ? 'Any' : value === true ? 'Yes' : 'No'}
        </span>
      </button>
    );
  }
);

BooleanFilterComponent.displayName = 'BooleanFilterComponent';

// ============================================
// Filter Item Component
// ============================================

interface FilterItemProps {
  config: FilterConfig;
}

const FilterItem: React.FC<FilterItemProps> = memo(({ config }) => {
  const { value, setValue } = useFilter(config.id);

  const renderFilter = () => {
    switch (config.type) {
      case 'select':
      case 'multiselect':
        return (
          <SelectFilterComponent
            config={config}
            value={value}
            onChange={setValue}
          />
        );

      case 'text':
        return (
          <TextFilterComponent
            config={config}
            value={value}
            onChange={setValue}
          />
        );

      case 'number':
        return (
          <NumberFilterComponent
            config={config}
            value={value}
            onChange={setValue}
          />
        );

      case 'boolean':
        return (
          <BooleanFilterComponent
            config={config}
            value={value}
            onChange={setValue}
          />
        );

      case 'date':
        return (
          <DateRangePicker
            value={value as DateRange | null}
            onChange={setValue as (value: DateRange | null) => void}
            placeholder={config.placeholder}
          />
        );

      case 'daterange':
        return (
          <DateRangePicker
            value={value as DateRange | null}
            onChange={setValue as (value: DateRange | null) => void}
            placeholder={config.placeholder}
            minDate={config.min as Date}
            maxDate={config.max as Date}
          />
        );

      default:
        return null;
    }
  };

  return (
    <div
      className="flex flex-col gap-1"
      data-testid={`filter-item-${config.id}`}
    >
      {config.type !== 'boolean' && (
        <label className="text-xs font-medium text-gray-500">
          {config.label}
        </label>
      )}
      {renderFilter()}
    </div>
  );
});

FilterItem.displayName = 'FilterItem';

// ============================================
// Active Filter Chip Component
// ============================================

interface FilterChipProps {
  filter: ActiveFilter;
  config: FilterConfig;
  onRemove: () => void;
}

const FilterChip: React.FC<FilterChipProps> = memo(
  ({ filter, config, onRemove }) => {
    const displayValue = formatFilterValue(filter.value, config.type);

    return (
      <div
        className="flex items-center gap-1.5 px-2 py-1 bg-blue-50 border border-blue-200 rounded-full text-sm"
        data-testid={`filter-chip-${filter.id}`}
      >
        <span className="text-blue-700 font-medium">{config.label}:</span>
        <span className="text-blue-600 max-w-[150px] truncate">
          {displayValue}
        </span>
        <button
          type="button"
          onClick={onRemove}
          className="p-0.5 hover:bg-blue-100 rounded-full"
          data-testid={`filter-chip-remove-${filter.id}`}
        >
          <XIcon className="w-3 h-3 text-blue-500" />
        </button>
      </div>
    );
  }
);

FilterChip.displayName = 'FilterChip';

// ============================================
// Active Filters Display
// ============================================

const ActiveFiltersDisplay: React.FC = memo(() => {
  const { state, config, removeFilter, clearFilters, hasActiveFilters } =
    useFilters();

  if (!hasActiveFilters()) return null;

  const activeFilters = Object.values(state.filters);

  return (
    <div
      className="flex flex-wrap items-center gap-2"
      data-testid="active-filters"
    >
      {activeFilters.map((filter) => {
        const filterConfig = config.find((c) => c.id === filter.id);
        if (!filterConfig) return null;
        return (
          <FilterChip
            key={filter.id}
            filter={filter}
            config={filterConfig}
            onRemove={() => removeFilter(filter.id)}
          />
        );
      })}
      {activeFilters.length > 1 && (
        <button
          type="button"
          onClick={clearFilters}
          className="text-sm text-gray-500 hover:text-gray-700 px-2 py-1"
          data-testid="clear-all-filters"
        >
          Clear all
        </button>
      )}
    </div>
  );
});

ActiveFiltersDisplay.displayName = 'ActiveFiltersDisplay';

// ============================================
// Main FilterBar Content Component
// ============================================

interface FilterBarContentProps {
  showSearch?: boolean;
  searchPlaceholder?: string;
  showActiveFilters?: boolean;
  compact?: boolean;
  className?: string;
}

const FilterBarContent: React.FC<FilterBarContentProps> = ({
  showSearch = true,
  searchPlaceholder,
  showActiveFilters = true,
  compact = false,
  className,
}) => {
  const [isExpanded, setIsExpanded] = useState(!compact);
  const { config, getActiveFilterCount } = useFilters();

  const visibleFilters =
    compact && !isExpanded ? config.filter((f) => !f.hideInCompact) : config;

  const filterCount = getActiveFilterCount();

  return (
    <div className={cn('space-y-3', className)} data-testid="filter-bar">
      {/* Main controls row */}
      <div className="flex flex-wrap items-end gap-4">
        {showSearch && (
          <SearchInput placeholder={searchPlaceholder} className="w-64" />
        )}

        {compact && (
          <button
            type="button"
            onClick={() => setIsExpanded(!isExpanded)}
            className={cn(
              'flex items-center gap-2 px-3 py-2 border rounded-lg text-sm transition-colors',
              isExpanded
                ? 'bg-gray-100 border-gray-300'
                : filterCount > 0
                  ? 'bg-blue-50 border-blue-300 text-blue-700'
                  : 'border-gray-300 hover:border-gray-400'
            )}
            data-testid="filter-toggle"
          >
            <FilterIcon className="w-4 h-4" />
            <span>Filters</span>
            {filterCount > 0 && (
              <span className="px-1.5 py-0.5 bg-blue-600 text-white text-xs rounded-full">
                {filterCount}
              </span>
            )}
            <ChevronDownIcon
              className={cn(
                'w-4 h-4 transition-transform',
                isExpanded && 'rotate-180'
              )}
            />
          </button>
        )}

        {(!compact || isExpanded) && (
          <div className="flex flex-wrap items-end gap-3">
            {visibleFilters.map((filterConfig) => (
              <FilterItem key={filterConfig.id} config={filterConfig} />
            ))}
          </div>
        )}
      </div>

      {/* Active filters row */}
      {showActiveFilters && <ActiveFiltersDisplay />}
    </div>
  );
};

// ============================================
// Main FilterBar Component (with Provider)
// ============================================

const FilterBar: React.FC<FilterBarProps> = ({
  filters,
  showSearch = true,
  searchPlaceholder = 'Search...',
  showActiveFilters = true,
  compact = false,
  syncUrl = false,
  persist = false,
  storageKey = 'filter_state',
  onChange,
  className,
}) => {
  return (
    <FilterProvider
      config={filters}
      syncUrl={syncUrl}
      persist={persist}
      storageKey={storageKey}
      onChange={onChange}
    >
      <FilterBarContent
        showSearch={showSearch}
        searchPlaceholder={searchPlaceholder}
        showActiveFilters={showActiveFilters}
        compact={compact}
        className={className}
      />
    </FilterProvider>
  );
};

export default memo(FilterBar);

// Export sub-components for flexibility
export {
  SearchInput,
  SelectFilterComponent,
  TextFilterComponent,
  NumberFilterComponent,
  BooleanFilterComponent,
  FilterItem,
  FilterChip,
  ActiveFiltersDisplay,
  FilterBarContent,
};
