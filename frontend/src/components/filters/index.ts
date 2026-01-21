// Main components
export { default as FilterBar } from './FilterBar';
export { default as DateRangePicker } from './DateRangePicker';
export { default as SegmentFilter } from './SegmentFilter';
export { default as NumberRangeFilter } from './NumberRangeFilter';

// State management
export { FilterProvider, useFilters, useFilter } from './FilterState';

// FilterBar sub-components
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
} from './FilterBar';

// Segment filter variants
export {
  PillSegmentFilter,
  TabSegmentFilter,
  ColorSegmentFilter,
} from './SegmentFilter';

// Number range filter variants
export {
  InlineNumberRangeFilter,
  CurrencyRangeFilter,
  PercentageRangeFilter,
} from './NumberRangeFilter';

// Re-export types for convenience
export type {
  // Core types
  FilterType,
  FilterOperator,
  FilterValue,
  DateRange,
  NumberRange,
  // Config types
  FilterOption,
  FilterConfig,
  ActiveFilter,
  FilterState,
  SortConfig,
  PaginationConfig,
  // Action types
  FilterAction,
  // Context types
  FilterContextValue,
  // Component prop types
  FilterBarProps,
  DateRangePickerProps,
  DateRangePreset,
  SegmentFilterProps,
  SelectFilterProps,
  NumberRangeFilterProps,
  // Utility types
  FilterChip as FilterChipType,
  FilterGroup,
} from '../../types/filter';

// Re-export utility functions
export {
  dateRangePresets,
  formatFilterValue,
  parseFilterValue,
  serializeFilterValue,
} from '../../types/filter';
