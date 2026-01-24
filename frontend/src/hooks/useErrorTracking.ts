import { useCallback, useEffect } from 'react';
import * as Sentry from '@sentry/react';
import {
  captureException,
  captureMessage,
  setUser,
  setTag,
  setExtra,
  addBreadcrumb,
} from '../services/errorTracking/sentry';

interface ErrorContext {
  tags?: Record<string, string>;
  extra?: Record<string, unknown>;
  fingerprint?: string[];
}

interface UseErrorTrackingOptions {
  componentName?: string;
  tags?: Record<string, string>;
}

/**
 * Hook for error tracking functionality in components
 */
export function useErrorTracking(options: UseErrorTrackingOptions = {}) {
  const { componentName, tags: initialTags } = options;

  // Set component context on mount
  useEffect(() => {
    if (componentName) {
      setTag('component', componentName);
    }

    if (initialTags) {
      for (const [key, value] of Object.entries(initialTags)) {
        setTag(key, value);
      }
    }
  }, [componentName, initialTags]);

  /**
   * Capture an error with optional context
   */
  const trackError = useCallback(
    (error: Error | unknown, context?: ErrorContext) => {
      const tags = {
        ...initialTags,
        ...context?.tags,
      };

      if (componentName) {
        tags.component = componentName;
      }

      return captureException(error, {
        tags,
        extra: context?.extra,
        fingerprint: context?.fingerprint,
      });
    },
    [componentName, initialTags]
  );

  /**
   * Capture a message with optional context
   */
  const trackMessage = useCallback(
    (
      message: string,
      level: Sentry.SeverityLevel = 'info',
      context?: Record<string, unknown>
    ) => {
      if (componentName) {
        setTag('component', componentName);
      }
      return captureMessage(message, level, context);
    },
    [componentName]
  );

  /**
   * Track a user action as a breadcrumb
   */
  const trackAction = useCallback(
    (action: string, data?: Record<string, unknown>) => {
      addBreadcrumb({
        category: 'user-action',
        message: action,
        data: {
          component: componentName,
          ...data,
        },
        level: 'info',
      });
    },
    [componentName]
  );

  /**
   * Track navigation as a breadcrumb
   */
  const trackNavigation = useCallback((from: string, to: string) => {
    addBreadcrumb({
      category: 'navigation',
      message: `Navigate from ${from} to ${to}`,
      data: { from, to },
      level: 'info',
    });
  }, []);

  /**
   * Track an API call as a breadcrumb
   */
  const trackApiCall = useCallback(
    (
      method: string,
      url: string,
      status?: number,
      data?: Record<string, unknown>
    ) => {
      addBreadcrumb({
        category: 'http',
        message: `${method} ${url}`,
        data: {
          method,
          url,
          status,
          ...data,
        },
        level: status && status >= 400 ? 'error' : 'info',
      });
    },
    []
  );

  /**
   * Create a performance span for tracking
   */
  const withSpan = useCallback(
    async <T>(name: string, fn: () => Promise<T>): Promise<T> => {
      return Sentry.startSpan({ name, op: 'function' }, async () => {
        return fn();
      });
    },
    []
  );

  return {
    trackError,
    trackMessage,
    trackAction,
    trackNavigation,
    trackApiCall,
    withSpan,
    setUser,
    setTag,
    setExtra,
    addBreadcrumb,
  };
}

/**
 * Hook for setting user context when authenticated
 */
export function useErrorTrackingUser(
  user: { id: string; email?: string; username?: string } | null
) {
  useEffect(() => {
    setUser(user);

    return () => {
      // Clear user on unmount if they logged out
      if (!user) {
        setUser(null);
      }
    };
  }, [user]);
}

/**
 * Hook for tracking component lifecycle errors
 */
export function useComponentErrorTracking(componentName: string) {
  const { trackError, trackAction } = useErrorTracking({ componentName });

  useEffect(() => {
    trackAction('component_mounted');

    return () => {
      trackAction('component_unmounted');
    };
  }, [trackAction]);

  return { trackError, trackAction };
}

/**
 * Hook for tracking async operations with automatic error capture
 */
export function useTrackedAsync<T extends unknown[], R>(
  operation: (...args: T) => Promise<R>,
  operationName: string
) {
  const { trackError, trackAction, withSpan } = useErrorTracking();

  const trackedOperation = useCallback(
    async (...args: T): Promise<R> => {
      trackAction(`${operationName}_started`);

      try {
        const result = await withSpan(operationName, () => operation(...args));
        trackAction(`${operationName}_completed`);
        return result;
      } catch (error) {
        trackError(error, {
          tags: { operation: operationName },
          extra: { args: JSON.stringify(args) },
        });
        trackAction(`${operationName}_failed`);
        throw error;
      }
    },
    [operation, operationName, trackAction, trackError, withSpan]
  );

  return trackedOperation;
}

/**
 * Hook for creating a try-catch wrapper with error tracking
 */
export function useErrorHandler(componentName?: string) {
  const { trackError } = useErrorTracking({ componentName });

  const handleError = useCallback(
    (error: unknown, context?: ErrorContext) => {
      console.error('Error caught:', error);
      trackError(error, context);
    },
    [trackError]
  );

  const withErrorHandling = useCallback(
    <T extends unknown[], R>(
      fn: (...args: T) => R,
      context?: ErrorContext
    ): ((...args: T) => R | undefined) => {
      return (...args: T): R | undefined => {
        try {
          return fn(...args);
        } catch (error) {
          handleError(error, context);
          return undefined;
        }
      };
    },
    [handleError]
  );

  const withAsyncErrorHandling = useCallback(
    <T extends unknown[], R>(
      fn: (...args: T) => Promise<R>,
      context?: ErrorContext
    ): ((...args: T) => Promise<R | undefined>) => {
      return async (...args: T): Promise<R | undefined> => {
        try {
          return await fn(...args);
        } catch (error) {
          handleError(error, context);
          return undefined;
        }
      };
    },
    [handleError]
  );

  return {
    handleError,
    withErrorHandling,
    withAsyncErrorHandling,
  };
}

export default useErrorTracking;
