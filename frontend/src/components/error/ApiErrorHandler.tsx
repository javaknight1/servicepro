/* eslint-disable react-refresh/only-export-components */
import React, {
  createContext,
  useContext,
  useCallback,
  useState,
  ReactNode,
} from 'react';
import {
  captureException,
  addBreadcrumb,
} from '../../services/errorTracking/sentry';

// API Error types
export interface ApiError {
  code: string;
  message: string;
  status: number;
  details?: string;
  retryable?: boolean;
}

export interface ApiErrorContextValue {
  error: ApiError | null;
  setError: (error: ApiError | null) => void;
  clearError: () => void;
  handleApiError: (
    error: unknown,
    context?: Record<string, unknown>
  ) => ApiError;
}

const ApiErrorContext = createContext<ApiErrorContextValue | null>(null);

interface ApiErrorProviderProps {
  children: ReactNode;
  onError?: (error: ApiError) => void;
}

/**
 * Provider component for API error handling
 */
export function ApiErrorProvider({
  children,
  onError,
}: ApiErrorProviderProps): JSX.Element {
  const [error, setErrorState] = useState<ApiError | null>(null);

  const setError = useCallback(
    (err: ApiError | null) => {
      setErrorState(err);
      if (err && onError) {
        onError(err);
      }
    },
    [onError]
  );

  const clearError = useCallback(() => {
    setErrorState(null);
  }, []);

  const handleApiError = useCallback(
    (err: unknown, context?: Record<string, unknown>): ApiError => {
      // Parse the error
      const apiError = parseApiError(err);

      // Add breadcrumb
      addBreadcrumb({
        category: 'api-error',
        message: apiError.message,
        data: {
          code: apiError.code,
          status: apiError.status,
          ...context,
        },
        level: 'error',
      });

      // Capture to Sentry for server errors
      if (apiError.status >= 500) {
        captureException(
          err instanceof Error ? err : new Error(apiError.message),
          {
            tags: {
              error_code: apiError.code,
              http_status: String(apiError.status),
            },
            extra: context,
          }
        );
      }

      // Set error state
      setError(apiError);

      return apiError;
    },
    [setError]
  );

  return (
    <ApiErrorContext.Provider
      value={{ error, setError, clearError, handleApiError }}
    >
      {children}
    </ApiErrorContext.Provider>
  );
}

/**
 * Hook to access API error context
 */
export function useApiError(): ApiErrorContextValue {
  const context = useContext(ApiErrorContext);
  if (!context) {
    throw new Error('useApiError must be used within an ApiErrorProvider');
  }
  return context;
}

/**
 * Parse various error formats into a standard ApiError
 */
function parseApiError(error: unknown): ApiError {
  // Already an ApiError
  if (isApiError(error)) {
    return error;
  }

  // Axios-like error response
  if (isAxiosError(error)) {
    const response = error.response;
    if (response?.data) {
      return {
        code: response.data.code || 'API_ERROR',
        message:
          response.data.message || response.statusText || 'An error occurred',
        status: response.status,
        details: response.data.details,
        retryable: response.status >= 500 || response.status === 429,
      };
    }

    // Network error
    if (error.code === 'ERR_NETWORK') {
      return {
        code: 'NETWORK_ERROR',
        message:
          'Unable to connect to the server. Please check your internet connection.',
        status: 0,
        retryable: true,
      };
    }

    // Timeout
    if (error.code === 'ECONNABORTED') {
      return {
        code: 'TIMEOUT',
        message: 'The request timed out. Please try again.',
        status: 0,
        retryable: true,
      };
    }
  }

  // Fetch API error
  if (error instanceof Response) {
    return {
      code: 'HTTP_ERROR',
      message: error.statusText || 'An error occurred',
      status: error.status,
      retryable: error.status >= 500,
    };
  }

  // Generic Error
  if (error instanceof Error) {
    return {
      code: 'UNKNOWN_ERROR',
      message: error.message,
      status: 0,
      retryable: false,
    };
  }

  // Unknown error type
  return {
    code: 'UNKNOWN_ERROR',
    message: String(error),
    status: 0,
    retryable: false,
  };
}

// Type guards
function isApiError(error: unknown): error is ApiError {
  return (
    typeof error === 'object' &&
    error !== null &&
    'code' in error &&
    'message' in error &&
    'status' in error
  );
}

interface AxiosLikeError {
  response?: {
    data?: {
      code?: string;
      message?: string;
      details?: string;
    };
    status: number;
    statusText: string;
  };
  code?: string;
}

function isAxiosError(error: unknown): error is AxiosLikeError {
  return (
    typeof error === 'object' &&
    error !== null &&
    ('response' in error || 'code' in error)
  );
}

/**
 * Component to display API errors
 */
interface ApiErrorDisplayProps {
  error: ApiError;
  onDismiss?: () => void;
  onRetry?: () => void;
}

export function ApiErrorDisplay({
  error,
  onDismiss,
  onRetry,
}: ApiErrorDisplayProps): JSX.Element {
  const isServerError = error.status >= 500;
  const isClientError = error.status >= 400 && error.status < 500;

  return (
    <div
      className={`api-error-display ${isServerError ? 'server-error' : ''} ${
        isClientError ? 'client-error' : ''
      }`}
    >
      <div className="error-icon">
        {isServerError ? (
          <svg
            width="24"
            height="24"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
          >
            <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
            <line x1="12" y1="9" x2="12" y2="13" />
            <line x1="12" y1="17" x2="12.01" y2="17" />
          </svg>
        ) : (
          <svg
            width="24"
            height="24"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
          >
            <circle cx="12" cy="12" r="10" />
            <line x1="15" y1="9" x2="9" y2="15" />
            <line x1="9" y1="9" x2="15" y2="15" />
          </svg>
        )}
      </div>

      <div className="error-content">
        <h4 className="error-title">{getErrorTitle(error)}</h4>
        <p className="error-message">{error.message}</p>
        {error.details && <p className="error-details">{error.details}</p>}
      </div>

      <div className="error-actions">
        {error.retryable && onRetry && (
          <button onClick={onRetry} className="btn-retry">
            Retry
          </button>
        )}
        {onDismiss && (
          <button onClick={onDismiss} className="btn-dismiss">
            Dismiss
          </button>
        )}
      </div>

      <style>{`
        .api-error-display {
          display: flex;
          align-items: flex-start;
          gap: 1rem;
          padding: 1rem;
          border-radius: 0.5rem;
          background-color: #fef2f2;
          border: 1px solid #fecaca;
        }

        .api-error-display.server-error {
          background-color: #fef2f2;
          border-color: #fecaca;
        }

        .api-error-display.client-error {
          background-color: #fffbeb;
          border-color: #fde68a;
        }

        .error-icon {
          flex-shrink: 0;
          color: #dc2626;
        }

        .client-error .error-icon {
          color: #d97706;
        }

        .error-content {
          flex: 1;
        }

        .error-title {
          font-weight: 600;
          color: #991b1b;
          margin: 0 0 0.25rem 0;
        }

        .client-error .error-title {
          color: #92400e;
        }

        .error-message {
          color: #b91c1c;
          margin: 0;
          font-size: 0.875rem;
        }

        .client-error .error-message {
          color: #b45309;
        }

        .error-details {
          color: #6b7280;
          font-size: 0.75rem;
          margin: 0.5rem 0 0 0;
        }

        .error-actions {
          display: flex;
          gap: 0.5rem;
        }

        .btn-retry, .btn-dismiss {
          padding: 0.375rem 0.75rem;
          border-radius: 0.25rem;
          font-size: 0.875rem;
          font-weight: 500;
          cursor: pointer;
        }

        .btn-retry {
          background-color: #dc2626;
          color: white;
          border: none;
        }

        .btn-retry:hover {
          background-color: #b91c1c;
        }

        .btn-dismiss {
          background-color: transparent;
          color: #6b7280;
          border: 1px solid #d1d5db;
        }

        .btn-dismiss:hover {
          background-color: #f3f4f6;
        }
      `}</style>
    </div>
  );
}

function getErrorTitle(error: ApiError): string {
  switch (error.code) {
    case 'NETWORK_ERROR':
      return 'Connection Error';
    case 'TIMEOUT':
      return 'Request Timeout';
    case 'UNAUTHORIZED':
      return 'Authentication Required';
    case 'FORBIDDEN':
      return 'Access Denied';
    case 'NOT_FOUND':
      return 'Not Found';
    case 'VALIDATION_ERROR':
      return 'Validation Error';
    case 'RATE_LIMIT':
      return 'Too Many Requests';
    default:
      if (error.status >= 500) {
        return 'Server Error';
      }
      return 'Error';
  }
}

export default ApiErrorProvider;
