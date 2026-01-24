import '@testing-library/jest-dom';
import { cleanup } from '@testing-library/react';
import { afterEach, vi } from 'vitest';

// Mock @sentry/react globally (not installed as a dependency)
vi.mock('@sentry/react', () => ({
  init: vi.fn(),
  withScope: vi.fn((fn) => fn({ setExtras: vi.fn() })),
  captureException: vi.fn(() => 'mock-event-id'),
  captureMessage: vi.fn(() => 'mock-event-id'),
  showReportDialog: vi.fn(),
  withErrorBoundary: vi.fn((Component) => Component),
  getCurrentHub: vi.fn(() => ({
    getScope: vi.fn(() => ({
      getTransaction: vi.fn(() => ({
        startChild: vi.fn(() => ({
          setStatus: vi.fn(),
          finish: vi.fn(),
        })),
      })),
    })),
  })),
}));

// Mock @tanstack/react-table globally (not installed as a dependency)
vi.mock('@tanstack/react-table', () => ({
  useReactTable: vi.fn(() => ({
    getHeaderGroups: vi.fn(() => []),
    getRowModel: vi.fn(() => ({ rows: [] })),
    getCanNextPage: vi.fn(() => false),
    getCanPreviousPage: vi.fn(() => false),
    nextPage: vi.fn(),
    previousPage: vi.fn(),
    getState: vi.fn(() => ({ pagination: { pageIndex: 0, pageSize: 10 } })),
    setPageSize: vi.fn(),
  })),
  getCoreRowModel: vi.fn(),
  getPaginationRowModel: vi.fn(),
  getSortedRowModel: vi.fn(),
  getFilteredRowModel: vi.fn(),
  flexRender: vi.fn((cell, context) => cell),
  createColumnHelper: vi.fn(() => ({
    accessor: vi.fn(),
    display: vi.fn(),
  })),
}));

// Cleanup after each test
afterEach(() => {
  cleanup();
});

// Mock window.matchMedia
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation((query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
});

// Mock ResizeObserver
class ResizeObserverMock {
  observe = vi.fn();
  unobserve = vi.fn();
  disconnect = vi.fn();
}

window.ResizeObserver = ResizeObserverMock as unknown as typeof ResizeObserver;

// Mock IntersectionObserver
class IntersectionObserverMock {
  observe = vi.fn();
  unobserve = vi.fn();
  disconnect = vi.fn();
}

window.IntersectionObserver =
  IntersectionObserverMock as unknown as typeof IntersectionObserver;

// Mock scrollTo
window.scrollTo = vi.fn();

// Suppress console errors during tests (optional)
// vi.spyOn(console, 'error').mockImplementation(() => {});
