import * as Sentry from '@sentry/react';

// Error tracking configuration
export interface ErrorTrackingConfig {
  dsn: string;
  environment: string;
  release?: string;
  sampleRate?: number;
  tracesSampleRate?: number;
  debug?: boolean;
  allowUrls?: (string | RegExp)[];
  denyUrls?: (string | RegExp)[];
  beforeSend?: (
    event: Sentry.Event,
    hint: Sentry.EventHint
  ) => Sentry.Event | null;
}

// Default configuration
const defaultConfig: Partial<ErrorTrackingConfig> = {
  environment: import.meta.env.MODE || 'development',
  release: import.meta.env.VITE_APP_VERSION || 'unknown',
  sampleRate: 1.0,
  tracesSampleRate: 0.1,
  debug: import.meta.env.DEV,
};

// Sensitive data patterns to filter
const sensitivePatterns = [
  /password/i,
  /secret/i,
  /token/i,
  /api_key/i,
  /apikey/i,
  /authorization/i,
  /credit_card/i,
  /creditcard/i,
  /ssn/i,
  /social_security/i,
];

// URLs to ignore (usually third-party scripts)
const defaultDenyUrls = [
  /extensions\//i,
  /^chrome:\/\//i,
  /^chrome-extension:\/\//i,
  /^moz-extension:\/\//i,
];

// Error messages to filter out
const ignoredErrors = [
  'ResizeObserver loop limit exceeded',
  'ResizeObserver loop completed with undelivered notifications',
  'Non-Error promise rejection captured',
  'Network Error',
  'Load failed',
  'Failed to fetch',
  'ChunkLoadError',
  'Loading chunk',
  'cancelled',
  'AbortError',
];

/**
 * Initialize Sentry error tracking
 */
export function initErrorTracking(config: ErrorTrackingConfig): void {
  if (!config.dsn) {
    console.warn('Sentry DSN not configured. Error tracking disabled.');
    return;
  }

  const finalConfig = { ...defaultConfig, ...config };

  Sentry.init({
    dsn: finalConfig.dsn,
    environment: finalConfig.environment,
    release: finalConfig.release,
    sampleRate: finalConfig.sampleRate,
    tracesSampleRate: finalConfig.tracesSampleRate,
    debug: finalConfig.debug,

    integrations: [
      Sentry.browserTracingIntegration({
        enableInp: true,
      }),
      Sentry.replayIntegration({
        maskAllText: true,
        blockAllMedia: true,
      }),
    ],

    tracePropagationTargets: [
      'localhost',
      /^https:\/\/api\.servicepro\.com/,
      /^https:\/\/.*\.servicepro\.com/,
    ],

    replaysSessionSampleRate: 0.1,
    replaysOnErrorSampleRate: 1.0,

    allowUrls: finalConfig.allowUrls,
    denyUrls: [...defaultDenyUrls, ...(finalConfig.denyUrls || [])],

    beforeSend: (event, hint) => {
      // Apply custom filter first
      if (finalConfig.beforeSend) {
        const result = finalConfig.beforeSend(event, hint);
        if (!result) return null;
        // Use the filtered result
        return sanitizeEvent(result) as Sentry.ErrorEvent;
      }

      // Filter ignored errors
      if (shouldIgnoreError(event, hint)) {
        return null;
      }

      // Sanitize sensitive data
      return sanitizeEvent(event) as Sentry.ErrorEvent;
    },

    beforeBreadcrumb: (breadcrumb) => {
      // Filter out noisy breadcrumbs
      if (breadcrumb.category === 'console' && breadcrumb.level === 'debug') {
        return null;
      }

      // Sanitize breadcrumb data
      if (breadcrumb.data) {
        breadcrumb.data = sanitizeData(breadcrumb.data);
      }

      return breadcrumb;
    },
  });
}

/**
 * Check if an error should be ignored
 */
function shouldIgnoreError(
  event: Sentry.Event,
  hint: Sentry.EventHint
): boolean {
  const error = hint.originalException;

  // Check error message
  if (error instanceof Error) {
    for (const ignored of ignoredErrors) {
      if (error.message.includes(ignored)) {
        return true;
      }
    }
  }

  // Check event message
  if (event.message) {
    for (const ignored of ignoredErrors) {
      if (event.message.includes(ignored)) {
        return true;
      }
    }
  }

  return false;
}

/**
 * Sanitize event data to remove sensitive information
 */
function sanitizeEvent(event: Sentry.Event): Sentry.Event {
  // Sanitize request headers
  if (event.request?.headers) {
    const sensitiveHeaders = [
      'Authorization',
      'Cookie',
      'Set-Cookie',
      'X-API-Key',
    ];
    for (const header of sensitiveHeaders) {
      if (event.request.headers[header]) {
        event.request.headers[header] = '[FILTERED]';
      }
    }
  }

  // Sanitize extra data
  if (event.extra) {
    event.extra = sanitizeData(event.extra);
  }

  // Sanitize tags
  if (event.tags) {
    event.tags = sanitizeData(event.tags) as Record<string, string>;
  }

  return event;
}

/**
 * Sanitize object data recursively
 */
function sanitizeData(data: Record<string, unknown>): Record<string, unknown> {
  const sanitized: Record<string, unknown> = {};

  for (const [key, value] of Object.entries(data)) {
    // Check if key matches sensitive pattern
    const isSensitive = sensitivePatterns.some((pattern) => pattern.test(key));

    if (isSensitive) {
      sanitized[key] = '[REDACTED]';
    } else if (typeof value === 'object' && value !== null) {
      sanitized[key] = sanitizeData(value as Record<string, unknown>);
    } else {
      sanitized[key] = value;
    }
  }

  return sanitized;
}

/**
 * Capture an exception with additional context
 */
export function captureException(
  error: Error | unknown,
  context?: {
    tags?: Record<string, string>;
    extra?: Record<string, unknown>;
    user?: { id: string; email?: string; username?: string };
    level?: Sentry.SeverityLevel;
    fingerprint?: string[];
  }
): string | undefined {
  return Sentry.withScope((scope) => {
    if (context?.tags) {
      for (const [key, value] of Object.entries(context.tags)) {
        scope.setTag(key, value);
      }
    }

    if (context?.extra) {
      for (const [key, value] of Object.entries(context.extra)) {
        scope.setExtra(key, value);
      }
    }

    if (context?.user) {
      scope.setUser(context.user);
    }

    if (context?.level) {
      scope.setLevel(context.level);
    }

    if (context?.fingerprint) {
      scope.setFingerprint(context.fingerprint);
    }

    return Sentry.captureException(error);
  });
}

/**
 * Capture a message
 */
export function captureMessage(
  message: string,
  level: Sentry.SeverityLevel = 'info',
  context?: Record<string, unknown>
): string | undefined {
  return Sentry.withScope((scope) => {
    scope.setLevel(level);

    if (context) {
      for (const [key, value] of Object.entries(context)) {
        scope.setExtra(key, value);
      }
    }

    return Sentry.captureMessage(message);
  });
}

/**
 * Set user context for error tracking
 */
export function setUser(
  user: { id: string; email?: string; username?: string } | null
): void {
  Sentry.setUser(user);
}

/**
 * Set a tag on all future events
 */
export function setTag(key: string, value: string): void {
  Sentry.setTag(key, value);
}

/**
 * Set extra context on all future events
 */
export function setExtra(key: string, value: unknown): void {
  Sentry.setExtra(key, value);
}

/**
 * Add a breadcrumb
 */
export function addBreadcrumb(breadcrumb: Sentry.Breadcrumb): void {
  Sentry.addBreadcrumb(breadcrumb);
}

/**
 * Start a performance span (replaces deprecated startTransaction)
 */
export function startPerformanceSpan<T>(
  name: string,
  op: string,
  callback: () => T
): T {
  return Sentry.startSpan({ name, op }, callback);
}

/**
 * Get the active span
 */
export function getActiveSpan(): Sentry.Span | undefined {
  return Sentry.getActiveSpan();
}

/**
 * Flush pending events
 */
export async function flush(timeout = 2000): Promise<boolean> {
  return Sentry.flush(timeout);
}

/**
 * Close the Sentry client
 */
export async function close(timeout = 2000): Promise<boolean> {
  return Sentry.close(timeout);
}

// Export Sentry for direct access when needed
export { Sentry };
