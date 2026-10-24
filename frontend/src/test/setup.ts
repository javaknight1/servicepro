import '@testing-library/jest-dom';
import { cleanup } from '@testing-library/react';
import { afterEach, vi } from 'vitest';

/**
 * In-memory Web Storage implementation.
 *
 * Node >= 25 ships an experimental global `localStorage`/`sessionStorage` that
 * shadows jsdom's implementation (and throws/returns undefined without
 * `--localstorage-file`). Install a spec-compliant in-memory Storage on both
 * globalThis and window so tests behave identically on Node 20/22 and 25+.
 */
class MemoryStorage implements Storage {
  private store = new Map<string, string>();

  get length(): number {
    return this.store.size;
  }

  clear(): void {
    this.store.clear();
  }

  getItem(key: string): string | null {
    return this.store.has(key) ? (this.store.get(key) as string) : null;
  }

  key(index: number): string | null {
    return Array.from(this.store.keys())[index] ?? null;
  }

  removeItem(key: string): void {
    this.store.delete(key);
  }

  setItem(key: string, value: string): void {
    this.store.set(key, String(value));
  }
}

function installStorage(name: 'localStorage' | 'sessionStorage'): void {
  const storage = new MemoryStorage();
  const descriptor: PropertyDescriptor = {
    configurable: true,
    enumerable: true,
    get: () => storage,
  };
  Object.defineProperty(globalThis, name, descriptor);
  if (typeof window !== 'undefined' && window !== globalThis) {
    Object.defineProperty(window, name, descriptor);
  }
}

installStorage('localStorage');
installStorage('sessionStorage');

// jsdom does not implement elementFromPoint; react-big-calendar's drag-and-drop
// Selection handler calls it on mousedown.
if (typeof document !== 'undefined' && !document.elementFromPoint) {
  document.elementFromPoint = () => null;
}

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

// Mock @sentry/tracing globally (not installed as a dependency)
vi.mock('@sentry/tracing', () => ({
  BrowserTracing: vi.fn().mockImplementation(() => ({})),
}));

// Mock the sentry service which depends on @sentry/tracing
vi.mock('../services/errorTracking/sentry', () => ({
  captureException: vi.fn(() => 'mock-event-id'),
  captureMessage: vi.fn(() => 'mock-event-id'),
  setUser: vi.fn(),
  setTag: vi.fn(),
  setExtra: vi.fn(),
  addBreadcrumb: vi.fn(),
  initSentry: vi.fn(),
}));

// The footer build stamp's GET /version query: stub it out so layout and
// page tests need no QueryClientProvider and fire no extra request (its
// "unknown" result hides the stamp). BuildStamp.test.tsx mocks it itself.
vi.mock('../hooks/useVersion', () => ({
  useVersion: vi.fn(() => ({ data: undefined })),
}));

// Cleanup after each test
afterEach(() => {
  cleanup();
  localStorage.clear();
  sessionStorage.clear();
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
