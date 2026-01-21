import React, {
  createContext,
  useContext,
  useReducer,
  useCallback,
  useEffect,
  useRef,
  useMemo,
  ReactNode,
} from 'react';
import {
  FilterState,
  FilterAction,
  FilterConfig,
  FilterContextValue,
  ActiveFilter,
  SortConfig,
  FilterValue,
  serializeFilterValue,
  parseFilterValue,
} from '../../types/filter';

// ============================================
// Initial State
// ============================================

const initialState: FilterState = {
  filters: {},
  searchQuery: '',
  sort: null,
  pagination: {
    page: 1,
    pageSize: 20,
  },
  loading: false,
  lastUpdated: Date.now(),
};

// ============================================
// Reducer
// ============================================

function filterReducer(state: FilterState, action: FilterAction): FilterState {
  switch (action.type) {
    case 'SET_FILTER':
      return {
        ...state,
        filters: {
          ...state.filters,
          [action.payload.id]: action.payload,
        },
        pagination: { ...state.pagination, page: 1 }, // Reset to first page
        lastUpdated: Date.now(),
      };

    case 'REMOVE_FILTER':
      const { [action.payload]: removed, ...remainingFilters } = state.filters;
      return {
        ...state,
        filters: remainingFilters,
        pagination: { ...state.pagination, page: 1 },
        lastUpdated: Date.now(),
      };

    case 'CLEAR_FILTERS':
      return {
        ...state,
        filters: {},
        searchQuery: '',
        pagination: { ...state.pagination, page: 1 },
        lastUpdated: Date.now(),
      };

    case 'SET_SEARCH':
      return {
        ...state,
        searchQuery: action.payload,
        pagination: { ...state.pagination, page: 1 },
        lastUpdated: Date.now(),
      };

    case 'SET_SORT':
      return {
        ...state,
        sort: action.payload,
        lastUpdated: Date.now(),
      };

    case 'SET_PAGE':
      return {
        ...state,
        pagination: { ...state.pagination, page: action.payload },
        lastUpdated: Date.now(),
      };

    case 'SET_PAGE_SIZE':
      return {
        ...state,
        pagination: { ...state.pagination, pageSize: action.payload, page: 1 },
        lastUpdated: Date.now(),
      };

    case 'SET_LOADING':
      return {
        ...state,
        loading: action.payload,
      };

    case 'RESTORE_STATE':
      return {
        ...state,
        ...action.payload,
        lastUpdated: Date.now(),
      };

    default:
      return state;
  }
}

// ============================================
// Context
// ============================================

const FilterContext = createContext<FilterContextValue | undefined>(undefined);

// ============================================
// Provider Props
// ============================================

interface FilterProviderProps {
  children: ReactNode;
  /** Filter configurations */
  config: FilterConfig[];
  /** Enable URL synchronization */
  syncUrl?: boolean;
  /** Enable localStorage persistence */
  persist?: boolean;
  /** Storage key for persistence */
  storageKey?: string;
  /** Initial state override */
  initialState?: Partial<FilterState>;
  /** Callback when filters change */
  onChange?: (state: FilterState) => void;
  /** Debounce delay for onChange callback (ms) */
  debounceMs?: number;
}

// ============================================
// URL Utilities
// ============================================

const getUrlParams = (): URLSearchParams => {
  if (typeof window === 'undefined') return new URLSearchParams();
  return new URLSearchParams(window.location.search);
};

const updateUrl = (params: Record<string, string>) => {
  if (typeof window === 'undefined') return;

  const url = new URL(window.location.href);

  // Clear existing filter params
  Array.from(url.searchParams.keys())
    .filter(
      (key) =>
        key.startsWith('filter_') ||
        ['search', 'sort', 'page', 'pageSize'].includes(key)
    )
    .forEach((key) => url.searchParams.delete(key));

  // Set new params
  Object.entries(params).forEach(([key, value]) => {
    if (value) {
      url.searchParams.set(key, value);
    }
  });

  window.history.replaceState({}, '', url.toString());
};

// ============================================
// Storage Utilities
// ============================================

const saveToLocalStorage = (key: string, state: FilterState) => {
  if (typeof window === 'undefined') return;

  try {
    const serialized = JSON.stringify({
      filters: state.filters,
      searchQuery: state.searchQuery,
      sort: state.sort,
      pagination: {
        page: state.pagination.page,
        pageSize: state.pagination.pageSize,
      },
    });
    localStorage.setItem(key, serialized);
  } catch (err) {
    console.error('Failed to save filter state:', err);
  }
};

const loadFromLocalStorage = (key: string): Partial<FilterState> | null => {
  if (typeof window === 'undefined') return null;

  try {
    const serialized = localStorage.getItem(key);
    if (!serialized) return null;
    return JSON.parse(serialized);
  } catch (err) {
    console.error('Failed to load filter state:', err);
    return null;
  }
};

const clearLocalStorage = (key: string) => {
  if (typeof window === 'undefined') return;
  localStorage.removeItem(key);
};

// ============================================
// Provider Component
// ============================================

export const FilterProvider: React.FC<FilterProviderProps> = ({
  children,
  config,
  syncUrl = false,
  persist = false,
  storageKey = 'filter_state',
  initialState: customInitialState,
  onChange,
  debounceMs = 300,
}) => {
  const [state, dispatch] = useReducer(filterReducer, {
    ...initialState,
    ...customInitialState,
  });

  const configRef = useRef(config);
  const onChangeRef = useRef(onChange);
  const debounceRef = useRef<NodeJS.Timeout | null>(null);
  const isInitializedRef = useRef(false);

  // Update refs
  useEffect(() => {
    configRef.current = config;
    onChangeRef.current = onChange;
  }, [config, onChange]);

  // Initialize from URL or storage
  useEffect(() => {
    if (isInitializedRef.current) return;
    isInitializedRef.current = true;

    if (syncUrl) {
      const params = getUrlParams();
      const restoredState: Partial<FilterState> = {};
      const filters: Record<string, ActiveFilter> = {};

      // Parse filters from URL
      params.forEach((value, key) => {
        if (key.startsWith('filter_')) {
          const filterId = key.replace('filter_', '');
          const filterConfig = config.find((f) => f.id === filterId);
          if (filterConfig) {
            const parsedValue = parseFilterValue(value, filterConfig.type);
            if (parsedValue !== null) {
              filters[filterId] = {
                id: filterId,
                field: filterConfig.field,
                operator: filterConfig.operator || 'equals',
                value: parsedValue,
              };
            }
          }
        } else if (key === 'search') {
          restoredState.searchQuery = value;
        } else if (key === 'sort') {
          const [field, direction] = value.split(':');
          if (field && (direction === 'asc' || direction === 'desc')) {
            restoredState.sort = { field, direction };
          }
        } else if (key === 'page') {
          const page = parseInt(value, 10);
          if (!isNaN(page) && page > 0) {
            restoredState.pagination = {
              ...initialState.pagination,
              page,
            };
          }
        } else if (key === 'pageSize') {
          const pageSize = parseInt(value, 10);
          if (!isNaN(pageSize) && pageSize > 0) {
            restoredState.pagination = {
              ...restoredState.pagination,
              ...initialState.pagination,
              pageSize,
            };
          }
        }
      });

      if (Object.keys(filters).length > 0) {
        restoredState.filters = filters;
      }

      if (Object.keys(restoredState).length > 0) {
        dispatch({ type: 'RESTORE_STATE', payload: restoredState });
        return;
      }
    }

    if (persist) {
      const savedState = loadFromLocalStorage(storageKey);
      if (savedState) {
        dispatch({ type: 'RESTORE_STATE', payload: savedState });
      }
    }
  }, [syncUrl, persist, storageKey, config]);

  // Sync to URL when state changes
  useEffect(() => {
    if (!syncUrl || !isInitializedRef.current) return;

    const params: Record<string, string> = {};

    // Add filters
    Object.values(state.filters).forEach((filter) => {
      const filterConfig = config.find((f) => f.id === filter.id);
      if (filterConfig) {
        const serialized = serializeFilterValue(
          filter.value,
          filterConfig.type
        );
        if (serialized) {
          params[`filter_${filter.id}`] = serialized;
        }
      }
    });

    // Add search
    if (state.searchQuery) {
      params.search = state.searchQuery;
    }

    // Add sort
    if (state.sort) {
      params.sort = `${state.sort.field}:${state.sort.direction}`;
    }

    // Add pagination
    if (state.pagination.page > 1) {
      params.page = String(state.pagination.page);
    }
    if (state.pagination.pageSize !== 20) {
      params.pageSize = String(state.pagination.pageSize);
    }

    updateUrl(params);
  }, [state, syncUrl, config]);

  // Persist to storage when state changes
  useEffect(() => {
    if (!persist || !isInitializedRef.current) return;
    saveToLocalStorage(storageKey, state);
  }, [state, persist, storageKey]);

  // Call onChange with debounce
  useEffect(() => {
    if (!onChange) return;

    if (debounceRef.current) {
      clearTimeout(debounceRef.current);
    }

    debounceRef.current = setTimeout(() => {
      onChangeRef.current?.(state);
    }, debounceMs);

    return () => {
      if (debounceRef.current) {
        clearTimeout(debounceRef.current);
      }
    };
  }, [state, debounceMs, onChange]);

  // Actions
  const setFilter = useCallback((filter: ActiveFilter) => {
    dispatch({ type: 'SET_FILTER', payload: filter });
  }, []);

  const removeFilter = useCallback((id: string) => {
    dispatch({ type: 'REMOVE_FILTER', payload: id });
  }, []);

  const clearFilters = useCallback(() => {
    dispatch({ type: 'CLEAR_FILTERS' });
  }, []);

  const getFilterValue = useCallback(
    (id: string): FilterValue | undefined => {
      return state.filters[id]?.value;
    },
    [state.filters]
  );

  const hasActiveFilters = useCallback(() => {
    return Object.keys(state.filters).length > 0 || state.searchQuery !== '';
  }, [state.filters, state.searchQuery]);

  const getActiveFilterCount = useCallback(() => {
    return Object.keys(state.filters).length + (state.searchQuery ? 1 : 0);
  }, [state.filters, state.searchQuery]);

  const setSearch = useCallback((query: string) => {
    dispatch({ type: 'SET_SEARCH', payload: query });
  }, []);

  const setSort = useCallback((sort: SortConfig | null) => {
    dispatch({ type: 'SET_SORT', payload: sort });
  }, []);

  const toggleSort = useCallback(
    (field: string) => {
      if (state.sort?.field === field) {
        if (state.sort.direction === 'asc') {
          dispatch({ type: 'SET_SORT', payload: { field, direction: 'desc' } });
        } else {
          dispatch({ type: 'SET_SORT', payload: null });
        }
      } else {
        dispatch({ type: 'SET_SORT', payload: { field, direction: 'asc' } });
      }
    },
    [state.sort]
  );

  const setPage = useCallback((page: number) => {
    dispatch({ type: 'SET_PAGE', payload: page });
  }, []);

  const setPageSize = useCallback((size: number) => {
    dispatch({ type: 'SET_PAGE_SIZE', payload: size });
  }, []);

  const syncToUrl = useCallback(() => {
    if (!syncUrl) return;
    // URL is already synced via useEffect
  }, [syncUrl]);

  const syncFromUrl = useCallback(() => {
    if (!syncUrl) return;
    // Re-initialize from URL
    isInitializedRef.current = false;
  }, [syncUrl]);

  const saveToStorage = useCallback(() => {
    if (!persist) return;
    saveToLocalStorage(storageKey, state);
  }, [persist, storageKey, state]);

  const loadFromStorage = useCallback(() => {
    if (!persist) return;
    const savedState = loadFromLocalStorage(storageKey);
    if (savedState) {
      dispatch({ type: 'RESTORE_STATE', payload: savedState });
    }
  }, [persist, storageKey]);

  const clearStorage = useCallback(() => {
    clearLocalStorage(storageKey);
  }, [storageKey]);

  const buildQueryParams = useCallback((): Record<string, string> => {
    const params: Record<string, string> = {};

    Object.values(state.filters).forEach((filter) => {
      const filterConfig = config.find((f) => f.id === filter.id);
      if (filterConfig) {
        params[filter.field] = serializeFilterValue(
          filter.value,
          filterConfig.type
        );
      }
    });

    if (state.searchQuery) {
      params.q = state.searchQuery;
    }

    if (state.sort) {
      params.sortBy = state.sort.field;
      params.sortOrder = state.sort.direction;
    }

    params.page = String(state.pagination.page);
    params.pageSize = String(state.pagination.pageSize);

    return params;
  }, [state, config]);

  const buildApiQuery = useCallback((): Record<string, unknown> => {
    const query: Record<string, unknown> = {
      page: state.pagination.page,
      pageSize: state.pagination.pageSize,
    };

    if (state.searchQuery) {
      query.search = state.searchQuery;
    }

    if (state.sort) {
      query.sort = {
        field: state.sort.field,
        direction: state.sort.direction,
      };
    }

    const filters: Record<string, unknown> = {};
    Object.values(state.filters).forEach((filter) => {
      filters[filter.field] = {
        operator: filter.operator,
        value: filter.value,
      };
    });

    if (Object.keys(filters).length > 0) {
      query.filters = filters;
    }

    return query;
  }, [state]);

  const contextValue: FilterContextValue = useMemo(
    () => ({
      state,
      config,
      setFilter,
      removeFilter,
      clearFilters,
      getFilterValue,
      hasActiveFilters,
      getActiveFilterCount,
      setSearch,
      setSort,
      toggleSort,
      setPage,
      setPageSize,
      syncToUrl,
      syncFromUrl,
      saveToStorage,
      loadFromStorage,
      clearStorage,
      buildQueryParams,
      buildApiQuery,
    }),
    [
      state,
      config,
      setFilter,
      removeFilter,
      clearFilters,
      getFilterValue,
      hasActiveFilters,
      getActiveFilterCount,
      setSearch,
      setSort,
      toggleSort,
      setPage,
      setPageSize,
      syncToUrl,
      syncFromUrl,
      saveToStorage,
      loadFromStorage,
      clearStorage,
      buildQueryParams,
      buildApiQuery,
    ]
  );

  return (
    <FilterContext.Provider value={contextValue}>
      {children}
    </FilterContext.Provider>
  );
};

// ============================================
// Hook
// ============================================

export const useFilters = (): FilterContextValue => {
  const context = useContext(FilterContext);
  if (!context) {
    throw new Error('useFilters must be used within a FilterProvider');
  }
  return context;
};

// ============================================
// Utility Hook for specific filter
// ============================================

export const useFilter = (id: string) => {
  const { state, setFilter, removeFilter, config } = useFilters();

  const filterConfig = config.find((f) => f.id === id);
  const activeFilter = state.filters[id];

  const setValue = useCallback(
    (value: FilterValue) => {
      if (!filterConfig) return;

      if (value === null || value === undefined || value === '') {
        removeFilter(id);
      } else {
        setFilter({
          id,
          field: filterConfig.field,
          operator: filterConfig.operator || 'equals',
          value,
        });
      }
    },
    [id, filterConfig, setFilter, removeFilter]
  );

  const clear = useCallback(() => {
    removeFilter(id);
  }, [id, removeFilter]);

  return {
    value: activeFilter?.value ?? filterConfig?.defaultValue ?? null,
    config: filterConfig,
    isActive: !!activeFilter,
    setValue,
    clear,
  };
};

export default FilterContext;
