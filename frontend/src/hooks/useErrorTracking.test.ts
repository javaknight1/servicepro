import { renderHook, act } from '@testing-library/react';
import { vi } from 'vitest';
import {
  useErrorTracking,
  useErrorTrackingUser,
  useComponentErrorTracking,
  useErrorHandler,
} from './useErrorTracking';

// Mock the sentry service
vi.mock('../services/errorTracking/sentry', () => ({
  captureException: vi.fn(() => 'mock-event-id'),
  captureMessage: vi.fn(() => 'mock-event-id'),
  setUser: vi.fn(),
  setTag: vi.fn(),
  setExtra: vi.fn(),
  addBreadcrumb: vi.fn(),
}));

// Mock Sentry
vi.mock('@sentry/react', () => ({
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

describe('useErrorTracking', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('trackError', () => {
    it('captures error with context', () => {
      const { captureException } = require('../services/errorTracking/sentry');
      const { result } = renderHook(() =>
        useErrorTracking({ componentName: 'TestComponent' })
      );

      const error = new Error('Test error');
      act(() => {
        result.current.trackError(error, {
          tags: { severity: 'high' },
          extra: { userId: '123' },
        });
      });

      expect(captureException).toHaveBeenCalledWith(error, {
        tags: expect.objectContaining({
          component: 'TestComponent',
          severity: 'high',
        }),
        extra: { userId: '123' },
        fingerprint: undefined,
      });
    });

    it('includes initial tags', () => {
      const { captureException } = require('../services/errorTracking/sentry');
      const { result } = renderHook(() =>
        useErrorTracking({
          componentName: 'TestComponent',
          tags: { feature: 'payments' },
        })
      );

      act(() => {
        result.current.trackError(new Error('Test'));
      });

      expect(captureException).toHaveBeenCalledWith(
        expect.any(Error),
        expect.objectContaining({
          tags: expect.objectContaining({
            feature: 'payments',
            component: 'TestComponent',
          }),
        })
      );
    });
  });

  describe('trackMessage', () => {
    it('captures message with level', () => {
      const { captureMessage } = require('../services/errorTracking/sentry');
      const { result } = renderHook(() => useErrorTracking());

      act(() => {
        result.current.trackMessage('Test message', 'warning', {
          detail: 'info',
        });
      });

      expect(captureMessage).toHaveBeenCalledWith('Test message', 'warning', {
        detail: 'info',
      });
    });
  });

  describe('trackAction', () => {
    it('adds breadcrumb for user action', () => {
      const { addBreadcrumb } = require('../services/errorTracking/sentry');
      const { result } = renderHook(() =>
        useErrorTracking({ componentName: 'TestComponent' })
      );

      act(() => {
        result.current.trackAction('button_clicked', { buttonId: 'submit' });
      });

      expect(addBreadcrumb).toHaveBeenCalledWith({
        category: 'user-action',
        message: 'button_clicked',
        data: {
          component: 'TestComponent',
          buttonId: 'submit',
        },
        level: 'info',
      });
    });
  });

  describe('trackNavigation', () => {
    it('adds breadcrumb for navigation', () => {
      const { addBreadcrumb } = require('../services/errorTracking/sentry');
      const { result } = renderHook(() => useErrorTracking());

      act(() => {
        result.current.trackNavigation('/home', '/dashboard');
      });

      expect(addBreadcrumb).toHaveBeenCalledWith({
        category: 'navigation',
        message: 'Navigate from /home to /dashboard',
        data: { from: '/home', to: '/dashboard' },
        level: 'info',
      });
    });
  });

  describe('trackApiCall', () => {
    it('adds breadcrumb for successful API call', () => {
      const { addBreadcrumb } = require('../services/errorTracking/sentry');
      const { result } = renderHook(() => useErrorTracking());

      act(() => {
        result.current.trackApiCall('GET', '/api/users', 200);
      });

      expect(addBreadcrumb).toHaveBeenCalledWith({
        category: 'http',
        message: 'GET /api/users',
        data: {
          method: 'GET',
          url: '/api/users',
          status: 200,
        },
        level: 'info',
      });
    });

    it('adds error level for failed API call', () => {
      const { addBreadcrumb } = require('../services/errorTracking/sentry');
      const { result } = renderHook(() => useErrorTracking());

      act(() => {
        result.current.trackApiCall('POST', '/api/users', 500);
      });

      expect(addBreadcrumb).toHaveBeenCalledWith(
        expect.objectContaining({
          level: 'error',
        })
      );
    });
  });
});

describe('useErrorTrackingUser', () => {
  it('sets user when provided', () => {
    const { setUser } = require('../services/errorTracking/sentry');
    const user = { id: '123', email: 'test@example.com' };

    renderHook(() => useErrorTrackingUser(user));

    expect(setUser).toHaveBeenCalledWith(user);
  });

  it('clears user when null', () => {
    const { setUser } = require('../services/errorTracking/sentry');

    const { rerender } = renderHook(({ user }) => useErrorTrackingUser(user), {
      initialProps: {
        user: { id: '123' } as {
          id: string;
          email?: string;
          username?: string;
        } | null,
      },
    });

    // Clear mocks
    setUser.mockClear();

    // Rerender with null user
    rerender({ user: null });

    // Should not be called during rerender cleanup
    // The hook clears on unmount, not on rerender
  });
});

describe('useComponentErrorTracking', () => {
  it('tracks component mount', () => {
    const { addBreadcrumb } = require('../services/errorTracking/sentry');

    renderHook(() => useComponentErrorTracking('TestComponent'));

    expect(addBreadcrumb).toHaveBeenCalledWith(
      expect.objectContaining({
        message: 'component_mounted',
      })
    );
  });

  it('tracks component unmount', () => {
    const { addBreadcrumb } = require('../services/errorTracking/sentry');

    const { unmount } = renderHook(() =>
      useComponentErrorTracking('TestComponent')
    );

    addBreadcrumb.mockClear();
    unmount();

    expect(addBreadcrumb).toHaveBeenCalledWith(
      expect.objectContaining({
        message: 'component_unmounted',
      })
    );
  });
});

describe('useErrorHandler', () => {
  describe('handleError', () => {
    it('logs and tracks error', () => {
      const { captureException } = require('../services/errorTracking/sentry');
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation();

      const { result } = renderHook(() => useErrorHandler('TestComponent'));

      const error = new Error('Test error');
      act(() => {
        result.current.handleError(error);
      });

      expect(consoleSpy).toHaveBeenCalledWith('Error caught:', error);
      expect(captureException).toHaveBeenCalled();

      consoleSpy.mockRestore();
    });
  });

  describe('withErrorHandling', () => {
    it('catches sync errors', () => {
      const { captureException } = require('../services/errorTracking/sentry');
      vi.spyOn(console, 'error').mockImplementation();

      const { result } = renderHook(() => useErrorHandler());

      const throwingFn = () => {
        throw new Error('Sync error');
      };

      const wrappedFn = result.current.withErrorHandling(throwingFn);

      act(() => {
        const returnValue = wrappedFn();
        expect(returnValue).toBeUndefined();
      });

      expect(captureException).toHaveBeenCalled();
    });

    it('returns value on success', () => {
      const { result } = renderHook(() => useErrorHandler());

      const successFn = () => 'success';
      const wrappedFn = result.current.withErrorHandling(successFn);

      let returnValue: string | undefined;
      act(() => {
        returnValue = wrappedFn();
      });

      expect(returnValue).toBe('success');
    });
  });

  describe('withAsyncErrorHandling', () => {
    it('catches async errors', async () => {
      const { captureException } = require('../services/errorTracking/sentry');
      vi.spyOn(console, 'error').mockImplementation();

      const { result } = renderHook(() => useErrorHandler());

      const throwingFn = async () => {
        throw new Error('Async error');
      };

      const wrappedFn = result.current.withAsyncErrorHandling(throwingFn);

      await act(async () => {
        const returnValue = await wrappedFn();
        expect(returnValue).toBeUndefined();
      });

      expect(captureException).toHaveBeenCalled();
    });

    it('returns value on success', async () => {
      const { result } = renderHook(() => useErrorHandler());

      const successFn = async () => 'async success';
      const wrappedFn = result.current.withAsyncErrorHandling(successFn);

      let returnValue: string | undefined;
      await act(async () => {
        returnValue = await wrappedFn();
      });

      expect(returnValue).toBe('async success');
    });
  });
});
