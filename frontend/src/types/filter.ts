/**
 * Filter System Types and Interfaces
 */

// ============================================
// Core Filter Types
// ============================================

export type FilterType =
  | 'text'
  | 'select'
  | 'multiselect'
  | 'date'
  | 'daterange'
  | 'number'
  | 'numberrange'
  | 'boolean'
  | 'segment';

export type FilterOperator =
  | 'equals'
  | 'notEquals'
  | 'contains'
  | 'startsWith'
  | 'endsWith'
  | 'greaterThan'
  | 'lessThan'
  | 'greaterOrEqual'
  | 'lessOrEqual'
  | 'between'
  | 'in'
  | 'notIn'
  | 'isNull'
  | 'isNotNull';

// ============================================
// Filter Value Types
// ============================================

export type FilterValue =
  | string
  | number
  | boolean
  | Date
  | string[]
  | number[]
  | DateRange
  | NumberRange
  | null;

export interface DateRange {
  start: Date | null;
  end: Date | null;
}

export interface NumberRange {
  min: number | null;
  max: number | null;
}

// ============================================
// Filter Configuration
// ============================================

export interface FilterOption {
  value: string | number;
  label: string;
  count?: number;
  disabled?: boolean;
  icon?: string | React.ReactNode;
  color?: string;
}

export interface FilterConfig {
  /** Unique filter identifier */
  id: string;
  /** Display label */
  label: string;
  /** Filter type */
  type: FilterType;
  /** Field to filter on */
  field: string;
  /** Default operator */
  operator?: FilterOperator;
  /** Available options for select/multiselect */
  options?: FilterOption[];
  /** Load options dynamically */
  loadOptions?: () => Promise<FilterOption[]>;
  /** Default value */
  defaultValue?: FilterValue;
  /** Placeholder text */
  placeholder?: string;
  /** Is filter required */
  required?: boolean;
  /** Is filter clearable */
  clearable?: boolean;
  /** Is filter searchable (for select) */
  searchable?: boolean;
  /** Custom validation */
  validate?: (value: FilterValue) => string | null;
  /** Group for organizing filters */
  group?: string;
  /** Order within group */
  order?: number;
  /** Hide filter in compact mode */
  hideInCompact?: boolean;
  /** Min value for number/date range */
  min?: number | Date;
  /** Max value for number/date range */
  max?: number | Date;
  /** Step for number input */
  step?: number;
}

// ============================================
// Active Filter State
// ============================================

export interface ActiveFilter {
  id: string;
  field: string;
  operator: FilterOperator;
  value: FilterValue;
  label?: string;
}

export interface FilterState {
  /** Currently active filters */
  filters: Record<string, ActiveFilter>;
  /** Search query (global text search) */
  searchQuery: string;
  /** Sort configuration */
  sort: SortConfig | null;
  /** Pagination */
  pagination: PaginationConfig;
  /** Is loading */
  loading: boolean;
  /** Last updated timestamp */
  lastUpdated: number;
}

export interface SortConfig {
  field: string;
  direction: 'asc' | 'desc';
}

export interface PaginationConfig {
  page: number;
  pageSize: number;
  total?: number;
}

// ============================================
// Filter Actions
// ============================================

export type FilterAction =
  | { type: 'SET_FILTER'; payload: ActiveFilter }
  | { type: 'REMOVE_FILTER'; payload: string }
  | { type: 'CLEAR_FILTERS' }
  | { type: 'SET_SEARCH'; payload: string }
  | { type: 'SET_SORT'; payload: SortConfig | null }
  | { type: 'SET_PAGE'; payload: number }
  | { type: 'SET_PAGE_SIZE'; payload: number }
  | { type: 'SET_LOADING'; payload: boolean }
  | { type: 'RESTORE_STATE'; payload: Partial<FilterState> };

// ============================================
// Filter Context Types
// ============================================

export interface FilterContextValue {
  state: FilterState;
  config: FilterConfig[];
  // Filter operations
  setFilter: (filter: ActiveFilter) => void;
  removeFilter: (id: string) => void;
  clearFilters: () => void;
  getFilterValue: (id: string) => FilterValue | undefined;
  hasActiveFilters: () => boolean;
  getActiveFilterCount: () => number;
  // Search
  setSearch: (query: string) => void;
  // Sort
  setSort: (sort: SortConfig | null) => void;
  toggleSort: (field: string) => void;
  // Pagination
  setPage: (page: number) => void;
  setPageSize: (size: number) => void;
  // URL sync
  syncToUrl: () => void;
  syncFromUrl: () => void;
  // Persistence
  saveToStorage: () => void;
  loadFromStorage: () => void;
  clearStorage: () => void;
  // Query building
  buildQueryParams: () => Record<string, string>;
  buildApiQuery: () => Record<string, unknown>;
}

// ============================================
// Component Props
// ============================================

export interface FilterBarProps {
  /** Filter configurations */
  filters: FilterConfig[];
  /** Show search input */
  showSearch?: boolean;
  /** Search placeholder */
  searchPlaceholder?: string;
  /** Show active filter chips */
  showActiveFilters?: boolean;
  /** Compact mode (collapsible) */
  compact?: boolean;
  /** Enable URL sync */
  syncUrl?: boolean;
  /** Enable localStorage persistence */
  persist?: boolean;
  /** Storage key for persistence */
  storageKey?: string;
  /** Callback when filters change */
  onChange?: (state: FilterState) => void;
  /** Custom className */
  className?: string;
}

export interface DateRangePickerProps {
  /** Current value */
  value: DateRange | null;
  /** Change handler */
  onChange: (range: DateRange | null) => void;
  /** Label */
  label?: string;
  /** Placeholder */
  placeholder?: string;
  /** Min date */
  minDate?: Date;
  /** Max date */
  maxDate?: Date;
  /** Preset ranges */
  presets?: DateRangePreset[];
  /** Show time picker */
  showTime?: boolean;
  /** Disabled */
  disabled?: boolean;
  /** Custom className */
  className?: string;
}

export interface DateRangePreset {
  label: string;
  getValue: () => DateRange;
}

export interface SegmentFilterProps {
  /** Filter ID */
  id: string;
  /** Label */
  label: string;
  /** Segments/options */
  segments: FilterOption[];
  /** Allow multiple selection */
  multiple?: boolean;
  /** Current value */
  value: string | string[] | null;
  /** Change handler */
  onChange: (value: string | string[] | null) => void;
  /** Show counts */
  showCounts?: boolean;
  /** Disabled */
  disabled?: boolean;
  /** Custom className */
  className?: string;
}

export interface SelectFilterProps {
  /** Filter ID */
  id: string;
  /** Label */
  label: string;
  /** Options */
  options: FilterOption[];
  /** Allow multiple selection */
  multiple?: boolean;
  /** Current value */
  value: string | string[] | null;
  /** Change handler */
  onChange: (value: string | string[] | null) => void;
  /** Searchable */
  searchable?: boolean;
  /** Clearable */
  clearable?: boolean;
  /** Placeholder */
  placeholder?: string;
  /** Disabled */
  disabled?: boolean;
  /** Loading options */
  loading?: boolean;
  /** Custom className */
  className?: string;
}

export interface NumberRangeFilterProps {
  /** Filter ID */
  id: string;
  /** Label */
  label?: string;
  /** Current value */
  value: NumberRange | null;
  /** Change handler */
  onChange: (range: NumberRange | null) => void;
  /** Min value */
  min?: number;
  /** Max value */
  max?: number;
  /** Step */
  step?: number;
  /** Format display value */
  formatValue?: (value: number) => string;
  /** Disabled */
  disabled?: boolean;
  /** Custom className */
  className?: string;
}

// ============================================
// Utility Types
// ============================================

export interface FilterChip {
  id: string;
  label: string;
  value: string;
  onRemove: () => void;
}

export interface FilterGroup {
  name: string;
  filters: FilterConfig[];
}

// ============================================
// Preset Date Ranges
// ============================================

export const dateRangePresets: DateRangePreset[] = [
  {
    label: 'Today',
    getValue: () => {
      const today = new Date();
      today.setHours(0, 0, 0, 0);
      const end = new Date();
      end.setHours(23, 59, 59, 999);
      return { start: today, end };
    },
  },
  {
    label: 'Yesterday',
    getValue: () => {
      const start = new Date();
      start.setDate(start.getDate() - 1);
      start.setHours(0, 0, 0, 0);
      const end = new Date(start);
      end.setHours(23, 59, 59, 999);
      return { start, end };
    },
  },
  {
    label: 'Last 7 days',
    getValue: () => {
      const end = new Date();
      end.setHours(23, 59, 59, 999);
      const start = new Date();
      start.setDate(start.getDate() - 6);
      start.setHours(0, 0, 0, 0);
      return { start, end };
    },
  },
  {
    label: 'Last 30 days',
    getValue: () => {
      const end = new Date();
      end.setHours(23, 59, 59, 999);
      const start = new Date();
      start.setDate(start.getDate() - 29);
      start.setHours(0, 0, 0, 0);
      return { start, end };
    },
  },
  {
    label: 'This month',
    getValue: () => {
      const start = new Date();
      start.setDate(1);
      start.setHours(0, 0, 0, 0);
      const end = new Date();
      end.setHours(23, 59, 59, 999);
      return { start, end };
    },
  },
  {
    label: 'Last month',
    getValue: () => {
      const start = new Date();
      start.setMonth(start.getMonth() - 1);
      start.setDate(1);
      start.setHours(0, 0, 0, 0);
      const end = new Date(start);
      end.setMonth(end.getMonth() + 1);
      end.setDate(0);
      end.setHours(23, 59, 59, 999);
      return { start, end };
    },
  },
  {
    label: 'This year',
    getValue: () => {
      const start = new Date();
      start.setMonth(0, 1);
      start.setHours(0, 0, 0, 0);
      const end = new Date();
      end.setHours(23, 59, 59, 999);
      return { start, end };
    },
  },
];

// ============================================
// Utility Functions
// ============================================

export const formatFilterValue = (
  value: FilterValue,
  type: FilterType
): string => {
  if (value === null || value === undefined) return '';

  switch (type) {
    case 'date':
      return value instanceof Date ? value.toLocaleDateString() : String(value);

    case 'daterange':
      const dateRange = value as DateRange;
      if (!dateRange.start && !dateRange.end) return '';
      const startStr = dateRange.start?.toLocaleDateString() || '...';
      const endStr = dateRange.end?.toLocaleDateString() || '...';
      return `${startStr} - ${endStr}`;

    case 'numberrange':
      const numRange = value as NumberRange;
      if (numRange.min === null && numRange.max === null) return '';
      const minStr =
        numRange.min !== null ? numRange.min.toLocaleString() : '...';
      const maxStr =
        numRange.max !== null ? numRange.max.toLocaleString() : '...';
      return `${minStr} - ${maxStr}`;

    case 'multiselect':
      return Array.isArray(value) ? value.join(', ') : String(value);

    case 'boolean':
      return value ? 'Yes' : 'No';

    default:
      return String(value);
  }
};

export const parseFilterValue = (
  value: string,
  type: FilterType
): FilterValue => {
  if (!value) return null;

  switch (type) {
    case 'number':
      const num = parseFloat(value);
      return isNaN(num) ? null : num;

    case 'boolean':
      return value === 'true';

    case 'date':
      const date = new Date(value);
      return isNaN(date.getTime()) ? null : date;

    case 'daterange':
      try {
        const [start, end] = value.split(',');
        return {
          start: start ? new Date(start) : null,
          end: end ? new Date(end) : null,
        };
      } catch {
        return null;
      }

    case 'multiselect':
      return value.split(',').filter(Boolean);

    default:
      return value;
  }
};

export const serializeFilterValue = (
  value: FilterValue,
  type: FilterType
): string => {
  if (value === null || value === undefined) return '';

  switch (type) {
    case 'date':
      return value instanceof Date ? value.toISOString() : '';

    case 'daterange':
      const dateRange = value as DateRange;
      return `${dateRange.start?.toISOString() || ''},${
        dateRange.end?.toISOString() || ''
      }`;

    case 'numberrange':
      const numRange = value as NumberRange;
      return `${numRange.min ?? ''},${numRange.max ?? ''}`;

    case 'multiselect':
      return Array.isArray(value) ? value.join(',') : '';

    case 'boolean':
      return String(value);

    default:
      return String(value);
  }
};
