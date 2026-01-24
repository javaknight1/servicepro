import React from 'react';
import {
  render,
  screen,
  fireEvent,
  waitFor,
  within,
} from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import FilterBar from '../FilterBar';
import DateRangePicker from '../DateRangePicker';
import SegmentFilter, {
  PillSegmentFilter,
  TabSegmentFilter,
} from '../SegmentFilter';
import NumberRangeFilter, { CurrencyRangeFilter } from '../NumberRangeFilter';
import { FilterProvider, useFilters, useFilter } from '../FilterState';
import {
  FilterConfig,
  FilterState,
  DateRange,
  NumberRange,
  dateRangePresets,
  serializeFilterValue,
  parseFilterValue,
  formatFilterValue,
} from '../../../types/filter';

// ============================================
// Test Utilities
// ============================================

const defaultFilters: FilterConfig[] = [
  {
    id: 'status',
    label: 'Status',
    type: 'select',
    field: 'status',
    options: [
      { value: 'active', label: 'Active' },
      { value: 'pending', label: 'Pending' },
      { value: 'completed', label: 'Completed' },
    ],
    clearable: true,
  },
  {
    id: 'priority',
    label: 'Priority',
    type: 'multiselect',
    field: 'priority',
    options: [
      { value: 'high', label: 'High', color: '#EF4444' },
      { value: 'medium', label: 'Medium', color: '#F59E0B' },
      { value: 'low', label: 'Low', color: '#10B981' },
    ],
  },
  {
    id: 'dateRange',
    label: 'Date Range',
    type: 'daterange',
    field: 'createdAt',
  },
  {
    id: 'amount',
    label: 'Amount',
    type: 'number',
    field: 'amount',
    min: 0,
    max: 10000,
  },
  {
    id: 'active',
    label: 'Active Only',
    type: 'boolean',
    field: 'isActive',
  },
];

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: vi.fn((key: string) => store[key] || null),
    setItem: vi.fn((key: string, value: string) => {
      store[key] = value;
    }),
    removeItem: vi.fn((key: string) => {
      delete store[key];
    }),
    clear: vi.fn(() => {
      store = {};
    }),
  };
})();

Object.defineProperty(window, 'localStorage', { value: localStorageMock });

// Mock URL and history
const mockHistoryReplaceState = vi.fn();
Object.defineProperty(window, 'history', {
  value: {
    replaceState: mockHistoryReplaceState,
  },
  writable: true,
});

// ============================================
// FilterBar Tests
// ============================================

describe('FilterBar', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorageMock.clear();
  });

  it('renders search input by default', () => {
    render(<FilterBar filters={defaultFilters} />);
    expect(screen.getByTestId('filter-search-input')).toBeInTheDocument();
  });

  it('renders filter controls for each filter config', () => {
    render(<FilterBar filters={defaultFilters} />);
    expect(screen.getByTestId('filter-item-status')).toBeInTheDocument();
    expect(screen.getByTestId('filter-item-priority')).toBeInTheDocument();
  });

  it('hides search input when showSearch is false', () => {
    render(<FilterBar filters={defaultFilters} showSearch={false} />);
    expect(screen.queryByTestId('filter-search-input')).not.toBeInTheDocument();
  });

  it('shows compact mode toggle when compact is true', () => {
    render(<FilterBar filters={defaultFilters} compact />);
    expect(screen.getByTestId('filter-toggle')).toBeInTheDocument();
  });

  it('calls onChange when filters change', async () => {
    const onChange = vi.fn();
    render(<FilterBar filters={defaultFilters} onChange={onChange} />);

    const statusFilter = screen.getByTestId('filter-select-status');
    fireEvent.click(statusFilter.querySelector('button')!);

    await waitFor(() => {
      const option = screen.getByText('Active');
      fireEvent.click(option);
    });

    await waitFor(
      () => {
        expect(onChange).toHaveBeenCalled();
      },
      { timeout: 500 }
    );
  });

  it('debounces search input changes', async () => {
    const onChange = vi.fn();
    render(<FilterBar filters={defaultFilters} onChange={onChange} />);

    const searchInput = screen.getByTestId('filter-search-input');
    await userEvent.type(searchInput, 'test');

    // Should debounce
    expect(onChange).not.toHaveBeenCalled();

    // Wait for debounce
    await waitFor(
      () => {
        expect(onChange).toHaveBeenCalled();
      },
      { timeout: 500 }
    );
  });

  it('clears search input when clear button is clicked', async () => {
    render(<FilterBar filters={defaultFilters} />);

    const searchInput = screen.getByTestId('filter-search-input');
    await userEvent.type(searchInput, 'test');

    await waitFor(() => {
      expect(screen.getByTestId('filter-search-clear')).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId('filter-search-clear'));
    expect(searchInput).toHaveValue('');
  });
});

// ============================================
// DateRangePicker Tests
// ============================================

describe('DateRangePicker', () => {
  it('renders with placeholder when no value', () => {
    const onChange = vi.fn();
    render(
      <DateRangePicker
        value={null}
        onChange={onChange}
        placeholder="Select dates"
      />
    );
    expect(screen.getByText('Select dates')).toBeInTheDocument();
  });

  it('opens calendar dropdown when clicked', async () => {
    const onChange = vi.fn();
    render(<DateRangePicker value={null} onChange={onChange} />);

    fireEvent.click(screen.getByTestId('date-range-trigger'));
    expect(screen.getByTestId('date-range-dropdown')).toBeInTheDocument();
  });

  it('displays preset options', async () => {
    const onChange = vi.fn();
    render(<DateRangePicker value={null} onChange={onChange} />);

    fireEvent.click(screen.getByTestId('date-range-trigger'));

    expect(screen.getByTestId('preset-today')).toBeInTheDocument();
    expect(screen.getByTestId('preset-yesterday')).toBeInTheDocument();
    expect(screen.getByTestId('preset-last-7-days')).toBeInTheDocument();
  });

  it('applies preset when clicked', async () => {
    const onChange = vi.fn();
    render(<DateRangePicker value={null} onChange={onChange} />);

    fireEvent.click(screen.getByTestId('date-range-trigger'));
    fireEvent.click(screen.getByTestId('preset-today'));

    expect(onChange).toHaveBeenCalled();
    const calledWith = onChange.mock.calls[0][0] as DateRange;
    expect(calledWith.start).toBeInstanceOf(Date);
    expect(calledWith.end).toBeInstanceOf(Date);
  });

  it('displays selected date range', () => {
    const onChange = vi.fn();
    const value: DateRange = {
      start: new Date(2024, 0, 1),
      end: new Date(2024, 0, 31),
    };
    render(<DateRangePicker value={value} onChange={onChange} />);

    expect(screen.getByText(/Jan 1, 2024/)).toBeInTheDocument();
    expect(screen.getByText(/Jan 31, 2024/)).toBeInTheDocument();
  });

  it('allows clearing selected dates', () => {
    const onChange = vi.fn();
    const value: DateRange = {
      start: new Date(2024, 0, 1),
      end: new Date(2024, 0, 31),
    };
    render(<DateRangePicker value={value} onChange={onChange} />);

    fireEvent.click(screen.getByTestId('clear-button'));
    expect(onChange).toHaveBeenCalledWith(null);
  });

  it('navigates months correctly', async () => {
    const onChange = vi.fn();
    render(<DateRangePicker value={null} onChange={onChange} />);

    fireEvent.click(screen.getByTestId('date-range-trigger'));

    const prevButtons = screen.getAllByTestId('prev-month');
    const nextButtons = screen.getAllByTestId('next-month');

    fireEvent.click(nextButtons[0]);
    fireEvent.click(prevButtons[0]);

    // Calendar should still be visible
    expect(screen.getByTestId('date-range-dropdown')).toBeInTheDocument();
  });
});

// ============================================
// SegmentFilter Tests
// ============================================

describe('SegmentFilter', () => {
  const segments = [
    { value: 'all', label: 'All', count: 100 },
    { value: 'active', label: 'Active', count: 50 },
    { value: 'inactive', label: 'Inactive', count: 50 },
  ];

  it('renders all segments', () => {
    const onChange = vi.fn();
    render(
      <SegmentFilter
        id="test"
        label="Test Filter"
        segments={segments}
        value={null}
        onChange={onChange}
      />
    );

    expect(screen.getByText('All')).toBeInTheDocument();
    expect(screen.getByText('Active')).toBeInTheDocument();
    expect(screen.getByText('Inactive')).toBeInTheDocument();
  });

  it('shows counts when showCounts is true', () => {
    const onChange = vi.fn();
    render(
      <SegmentFilter
        id="test"
        label="Test Filter"
        segments={segments}
        value={null}
        onChange={onChange}
        showCounts
      />
    );

    expect(screen.getByText('100')).toBeInTheDocument();
    // There are two segments with count 50 (Active and Inactive)
    expect(screen.getAllByText('50')).toHaveLength(2);
  });

  it('selects segment on click', () => {
    const onChange = vi.fn();
    render(
      <SegmentFilter
        id="test"
        label="Test Filter"
        segments={segments}
        value={null}
        onChange={onChange}
      />
    );

    fireEvent.click(screen.getByTestId('segment-active'));
    expect(onChange).toHaveBeenCalledWith('active');
  });

  it('allows multiple selection when multiple is true', () => {
    const onChange = vi.fn();
    render(
      <SegmentFilter
        id="test"
        label="Test Filter"
        segments={segments}
        value={['active']}
        onChange={onChange}
        multiple
      />
    );

    fireEvent.click(screen.getByTestId('segment-inactive'));
    expect(onChange).toHaveBeenCalledWith(['active', 'inactive']);
  });

  it('deselects when clicking selected segment', () => {
    const onChange = vi.fn();
    render(
      <SegmentFilter
        id="test"
        label="Test Filter"
        segments={segments}
        value="active"
        onChange={onChange}
      />
    );

    fireEvent.click(screen.getByTestId('segment-active'));
    expect(onChange).toHaveBeenCalledWith(null);
  });
});

describe('PillSegmentFilter', () => {
  const segments = [
    { value: 'new', label: 'New' },
    { value: 'in-progress', label: 'In Progress' },
    { value: 'done', label: 'Done' },
  ];

  it('renders pill-style segments', () => {
    const onChange = vi.fn();
    render(
      <PillSegmentFilter
        id="test"
        label="Status"
        segments={segments}
        value={null}
        onChange={onChange}
      />
    );

    expect(screen.getByTestId('pill-segment-filter-test')).toBeInTheDocument();
    expect(screen.getByTestId('pill-segment-new')).toBeInTheDocument();
  });
});

describe('TabSegmentFilter', () => {
  const segments = [
    { value: 'overview', label: 'Overview' },
    { value: 'details', label: 'Details' },
    { value: 'history', label: 'History' },
  ];

  it('renders tab-style segments', () => {
    const onChange = vi.fn();
    render(
      <TabSegmentFilter
        id="test"
        label="View"
        segments={segments}
        value="overview"
        onChange={onChange}
      />
    );

    expect(screen.getByTestId('tab-segment-filter-test')).toBeInTheDocument();
    expect(screen.getByRole('tablist')).toBeInTheDocument();
  });

  it('marks selected tab with aria-selected', () => {
    const onChange = vi.fn();
    render(
      <TabSegmentFilter
        id="test"
        label="View"
        segments={segments}
        value="overview"
        onChange={onChange}
      />
    );

    expect(screen.getByTestId('tab-segment-overview')).toHaveAttribute(
      'aria-selected',
      'true'
    );
    expect(screen.getByTestId('tab-segment-details')).toHaveAttribute(
      'aria-selected',
      'false'
    );
  });
});

// ============================================
// NumberRangeFilter Tests
// ============================================

describe('NumberRangeFilter', () => {
  it('renders with placeholder when no value', () => {
    const onChange = vi.fn();
    render(<NumberRangeFilter id="test" value={null} onChange={onChange} />);

    expect(screen.getByText('Any')).toBeInTheDocument();
  });

  it('displays range when value is set', () => {
    const onChange = vi.fn();
    render(
      <NumberRangeFilter
        id="test"
        value={{ min: 10, max: 100 }}
        onChange={onChange}
      />
    );

    expect(screen.getByText('10 - 100')).toBeInTheDocument();
  });

  it('opens dropdown when clicked', () => {
    const onChange = vi.fn();
    render(<NumberRangeFilter id="test" value={null} onChange={onChange} />);

    fireEvent.click(screen.getByTestId('number-range-trigger-test'));
    expect(
      screen.getByTestId('number-range-dropdown-test')
    ).toBeInTheDocument();
  });

  it('clears value when clear button is clicked', () => {
    const onChange = vi.fn();
    render(
      <NumberRangeFilter
        id="test"
        value={{ min: 10, max: 100 }}
        onChange={onChange}
      />
    );

    const trigger = screen.getByTestId('number-range-trigger-test');
    const clearButton = within(trigger).getByRole('button', { hidden: true });
    fireEvent.click(clearButton);

    expect(onChange).toHaveBeenCalledWith(null);
  });
});

describe('CurrencyRangeFilter', () => {
  it('formats values as currency', () => {
    const onChange = vi.fn();
    render(
      <CurrencyRangeFilter
        id="test"
        value={{ min: 1000, max: 5000 }}
        onChange={onChange}
        currency="USD"
      />
    );

    expect(screen.getByText('$1,000 - $5,000')).toBeInTheDocument();
  });
});

// ============================================
// FilterState Context Tests
// ============================================

describe('FilterState', () => {
  // Helper component to test hooks
  const TestComponent: React.FC<{
    onStateChange?: (state: FilterState) => void;
  }> = ({ onStateChange }) => {
    const { state, setFilter, removeFilter, clearFilters, setSearch, setSort } =
      useFilters();

    React.useEffect(() => {
      onStateChange?.(state);
    }, [state, onStateChange]);

    return (
      <div>
        <button
          data-testid="set-filter"
          onClick={() =>
            setFilter({
              id: 'status',
              field: 'status',
              operator: 'equals',
              value: 'active',
            })
          }
        >
          Set Filter
        </button>
        <button
          data-testid="remove-filter"
          onClick={() => removeFilter('status')}
        >
          Remove Filter
        </button>
        <button data-testid="clear-filters" onClick={() => clearFilters()}>
          Clear Filters
        </button>
        <button
          data-testid="set-search"
          onClick={() => setSearch('test query')}
        >
          Set Search
        </button>
        <button
          data-testid="set-sort"
          onClick={() => setSort({ field: 'name', direction: 'asc' })}
        >
          Set Sort
        </button>
        <div data-testid="filter-count">
          {Object.keys(state.filters).length}
        </div>
        <div data-testid="search-query">{state.searchQuery}</div>
        <div data-testid="sort">
          {state.sort ? `${state.sort.field}:${state.sort.direction}` : 'none'}
        </div>
      </div>
    );
  };

  it('sets filter correctly', async () => {
    render(
      <FilterProvider config={defaultFilters}>
        <TestComponent />
      </FilterProvider>
    );

    fireEvent.click(screen.getByTestId('set-filter'));

    await waitFor(() => {
      expect(screen.getByTestId('filter-count')).toHaveTextContent('1');
    });
  });

  it('removes filter correctly', async () => {
    render(
      <FilterProvider config={defaultFilters}>
        <TestComponent />
      </FilterProvider>
    );

    fireEvent.click(screen.getByTestId('set-filter'));
    await waitFor(() => {
      expect(screen.getByTestId('filter-count')).toHaveTextContent('1');
    });

    fireEvent.click(screen.getByTestId('remove-filter'));
    await waitFor(() => {
      expect(screen.getByTestId('filter-count')).toHaveTextContent('0');
    });
  });

  it('clears all filters', async () => {
    render(
      <FilterProvider config={defaultFilters}>
        <TestComponent />
      </FilterProvider>
    );

    fireEvent.click(screen.getByTestId('set-filter'));
    fireEvent.click(screen.getByTestId('set-search'));

    await waitFor(() => {
      expect(screen.getByTestId('filter-count')).toHaveTextContent('1');
      expect(screen.getByTestId('search-query')).toHaveTextContent(
        'test query'
      );
    });

    fireEvent.click(screen.getByTestId('clear-filters'));

    await waitFor(() => {
      expect(screen.getByTestId('filter-count')).toHaveTextContent('0');
      expect(screen.getByTestId('search-query')).toHaveTextContent('');
    });
  });

  it('sets search query', async () => {
    render(
      <FilterProvider config={defaultFilters}>
        <TestComponent />
      </FilterProvider>
    );

    fireEvent.click(screen.getByTestId('set-search'));

    await waitFor(() => {
      expect(screen.getByTestId('search-query')).toHaveTextContent(
        'test query'
      );
    });
  });

  it('sets sort configuration', async () => {
    render(
      <FilterProvider config={defaultFilters}>
        <TestComponent />
      </FilterProvider>
    );

    fireEvent.click(screen.getByTestId('set-sort'));

    await waitFor(() => {
      expect(screen.getByTestId('sort')).toHaveTextContent('name:asc');
    });
  });
});

// ============================================
// State Persistence Tests
// ============================================

describe('State Persistence', () => {
  beforeEach(() => {
    localStorageMock.clear();
    vi.clearAllMocks();
  });

  it('saves state to localStorage when persist is enabled', async () => {
    const TestComponent: React.FC = () => {
      const { setFilter } = useFilters();
      return (
        <button
          onClick={() =>
            setFilter({
              id: 'status',
              field: 'status',
              operator: 'equals',
              value: 'active',
            })
          }
        >
          Set Filter
        </button>
      );
    };

    render(
      <FilterProvider config={defaultFilters} persist storageKey="test_filters">
        <TestComponent />
      </FilterProvider>
    );

    fireEvent.click(screen.getByText('Set Filter'));

    await waitFor(() => {
      expect(localStorageMock.setItem).toHaveBeenCalled();
    });
  });

  it('loads state from localStorage on mount', () => {
    const savedState = JSON.stringify({
      filters: {
        status: {
          id: 'status',
          field: 'status',
          operator: 'equals',
          value: 'active',
        },
      },
      searchQuery: 'saved search',
    });

    localStorageMock.getItem.mockReturnValue(savedState);

    const TestComponent: React.FC = () => {
      const { state } = useFilters();
      return (
        <div>
          <div data-testid="search">{state.searchQuery}</div>
          <div data-testid="filters">{Object.keys(state.filters).length}</div>
        </div>
      );
    };

    render(
      <FilterProvider config={defaultFilters} persist storageKey="test_filters">
        <TestComponent />
      </FilterProvider>
    );

    expect(localStorageMock.getItem).toHaveBeenCalledWith('test_filters');
  });
});

// ============================================
// URL Sync Tests
// ============================================

describe('URL Sync', () => {
  beforeEach(() => {
    mockHistoryReplaceState.mockClear();
    // Reset URL
    Object.defineProperty(window, 'location', {
      value: {
        search: '',
        href: 'http://localhost/',
      },
      writable: true,
    });
  });

  it('syncs filters to URL when syncUrl is enabled', async () => {
    const TestComponent: React.FC = () => {
      const { setFilter } = useFilters();
      return (
        <button
          onClick={() =>
            setFilter({
              id: 'status',
              field: 'status',
              operator: 'equals',
              value: 'active',
            })
          }
        >
          Set Filter
        </button>
      );
    };

    render(
      <FilterProvider config={defaultFilters} syncUrl>
        <TestComponent />
      </FilterProvider>
    );

    fireEvent.click(screen.getByText('Set Filter'));

    await waitFor(() => {
      expect(mockHistoryReplaceState).toHaveBeenCalled();
    });
  });
});

// ============================================
// Filter Combination Tests
// ============================================

describe('Filter Combinations', () => {
  it('combines multiple filters correctly', async () => {
    let capturedState: FilterState | null = null;

    const TestComponent: React.FC = () => {
      const { state, setFilter, setSearch, setSort } = useFilters();
      capturedState = state;

      return (
        <div>
          <button
            data-testid="add-status"
            onClick={() =>
              setFilter({
                id: 'status',
                field: 'status',
                operator: 'equals',
                value: 'active',
              })
            }
          >
            Add Status
          </button>
          <button
            data-testid="add-priority"
            onClick={() =>
              setFilter({
                id: 'priority',
                field: 'priority',
                operator: 'in',
                value: ['high', 'medium'],
              })
            }
          >
            Add Priority
          </button>
          <button data-testid="add-search" onClick={() => setSearch('test')}>
            Add Search
          </button>
          <button
            data-testid="add-sort"
            onClick={() => setSort({ field: 'createdAt', direction: 'desc' })}
          >
            Add Sort
          </button>
        </div>
      );
    };

    render(
      <FilterProvider config={defaultFilters}>
        <TestComponent />
      </FilterProvider>
    );

    fireEvent.click(screen.getByTestId('add-status'));
    fireEvent.click(screen.getByTestId('add-priority'));
    fireEvent.click(screen.getByTestId('add-search'));
    fireEvent.click(screen.getByTestId('add-sort'));

    await waitFor(() => {
      expect(capturedState).not.toBeNull();
      expect(Object.keys(capturedState!.filters)).toHaveLength(2);
      expect(capturedState!.searchQuery).toBe('test');
      expect(capturedState!.sort?.field).toBe('createdAt');
    });
  });

  it('builds query params from combined filters', async () => {
    let queryParams: Record<string, string> = {};

    const TestComponent: React.FC = () => {
      const { setFilter, buildQueryParams } = useFilters();

      return (
        <div>
          <button
            data-testid="add-filters"
            onClick={() => {
              setFilter({
                id: 'status',
                field: 'status',
                operator: 'equals',
                value: 'active',
              });
            }}
          >
            Add Filters
          </button>
          <button
            data-testid="build-params"
            onClick={() => {
              queryParams = buildQueryParams();
            }}
          >
            Build Params
          </button>
        </div>
      );
    };

    render(
      <FilterProvider config={defaultFilters}>
        <TestComponent />
      </FilterProvider>
    );

    fireEvent.click(screen.getByTestId('add-filters'));

    await waitFor(() => {
      fireEvent.click(screen.getByTestId('build-params'));
    });

    expect(queryParams.status).toBe('active');
  });
});

// ============================================
// Utility Function Tests
// ============================================

describe('Filter Utility Functions', () => {
  describe('serializeFilterValue', () => {
    it('serializes string values', () => {
      expect(serializeFilterValue('test', 'text')).toBe('test');
    });

    it('serializes number values', () => {
      expect(serializeFilterValue(123, 'number')).toBe('123');
    });

    it('serializes boolean values', () => {
      expect(serializeFilterValue(true, 'boolean')).toBe('true');
      expect(serializeFilterValue(false, 'boolean')).toBe('false');
    });

    it('serializes date values', () => {
      const date = new Date('2024-01-15T00:00:00.000Z');
      const serialized = serializeFilterValue(date, 'date');
      expect(serialized).toContain('2024-01-15');
    });

    it('serializes date range values', () => {
      const range: DateRange = {
        start: new Date('2024-01-01'),
        end: new Date('2024-01-31'),
      };
      const serialized = serializeFilterValue(range, 'daterange');
      expect(serialized).toContain('2024-01-01');
      expect(serialized).toContain('2024-01-31');
    });

    it('serializes multiselect values', () => {
      expect(serializeFilterValue(['a', 'b', 'c'], 'multiselect')).toBe(
        'a,b,c'
      );
    });
  });

  describe('parseFilterValue', () => {
    it('parses string values', () => {
      expect(parseFilterValue('test', 'text')).toBe('test');
    });

    it('parses number values', () => {
      expect(parseFilterValue('123', 'number')).toBe(123);
      expect(parseFilterValue('invalid', 'number')).toBeNull();
    });

    it('parses boolean values', () => {
      expect(parseFilterValue('true', 'boolean')).toBe(true);
      expect(parseFilterValue('false', 'boolean')).toBe(false);
    });

    it('parses multiselect values', () => {
      expect(parseFilterValue('a,b,c', 'multiselect')).toEqual(['a', 'b', 'c']);
    });
  });

  describe('formatFilterValue', () => {
    it('formats string values', () => {
      expect(formatFilterValue('test', 'text')).toBe('test');
    });

    it('formats boolean values', () => {
      expect(formatFilterValue(true, 'boolean')).toBe('Yes');
      expect(formatFilterValue(false, 'boolean')).toBe('No');
    });

    it('formats multiselect values', () => {
      expect(formatFilterValue(['a', 'b'], 'multiselect')).toBe('a, b');
    });

    it('formats number range values', () => {
      const range: NumberRange = { min: 10, max: 100 };
      expect(formatFilterValue(range, 'numberrange')).toBe('10 - 100');
    });
  });
});

// ============================================
// Date Preset Tests
// ============================================

describe('Date Range Presets', () => {
  it('today preset returns current day', () => {
    const todayPreset = dateRangePresets.find((p) => p.label === 'Today');
    expect(todayPreset).toBeDefined();

    const range = todayPreset!.getValue();
    const today = new Date();

    expect(range.start?.getDate()).toBe(today.getDate());
    expect(range.end?.getDate()).toBe(today.getDate());
  });

  it('yesterday preset returns previous day', () => {
    const yesterdayPreset = dateRangePresets.find(
      (p) => p.label === 'Yesterday'
    );
    expect(yesterdayPreset).toBeDefined();

    const range = yesterdayPreset!.getValue();
    const yesterday = new Date();
    yesterday.setDate(yesterday.getDate() - 1);

    expect(range.start?.getDate()).toBe(yesterday.getDate());
    expect(range.end?.getDate()).toBe(yesterday.getDate());
  });

  it('last 7 days preset returns correct range', () => {
    const preset = dateRangePresets.find((p) => p.label === 'Last 7 days');
    expect(preset).toBeDefined();

    const range = preset!.getValue();
    const today = new Date();
    const sevenDaysAgo = new Date();
    sevenDaysAgo.setDate(today.getDate() - 6);

    expect(range.start?.getDate()).toBe(sevenDaysAgo.getDate());
    expect(range.end?.getDate()).toBe(today.getDate());
  });

  it('this month preset starts on first day', () => {
    const preset = dateRangePresets.find((p) => p.label === 'This month');
    expect(preset).toBeDefined();

    const range = preset!.getValue();
    expect(range.start?.getDate()).toBe(1);
  });
});

// ============================================
// useFilter Hook Tests
// ============================================

describe('useFilter Hook', () => {
  const SingleFilterTest: React.FC<{ filterId: string }> = ({ filterId }) => {
    const { value, isActive, setValue, clear, config } = useFilter(filterId);

    return (
      <div>
        <div data-testid="value">{String(value)}</div>
        <div data-testid="is-active">{isActive ? 'active' : 'inactive'}</div>
        <div data-testid="config-label">{config?.label || 'no config'}</div>
        <button data-testid="set-value" onClick={() => setValue('test-value')}>
          Set Value
        </button>
        <button data-testid="clear" onClick={clear}>
          Clear
        </button>
      </div>
    );
  };

  it('returns filter config', () => {
    render(
      <FilterProvider config={defaultFilters}>
        <SingleFilterTest filterId="status" />
      </FilterProvider>
    );

    expect(screen.getByTestId('config-label')).toHaveTextContent('Status');
  });

  it('sets filter value', async () => {
    render(
      <FilterProvider config={defaultFilters}>
        <SingleFilterTest filterId="status" />
      </FilterProvider>
    );

    fireEvent.click(screen.getByTestId('set-value'));

    await waitFor(() => {
      expect(screen.getByTestId('is-active')).toHaveTextContent('active');
    });
  });

  it('clears filter value', async () => {
    render(
      <FilterProvider config={defaultFilters}>
        <SingleFilterTest filterId="status" />
      </FilterProvider>
    );

    fireEvent.click(screen.getByTestId('set-value'));
    await waitFor(() => {
      expect(screen.getByTestId('is-active')).toHaveTextContent('active');
    });

    fireEvent.click(screen.getByTestId('clear'));
    await waitFor(() => {
      expect(screen.getByTestId('is-active')).toHaveTextContent('inactive');
    });
  });
});
