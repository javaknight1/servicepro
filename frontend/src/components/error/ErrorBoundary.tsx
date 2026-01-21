import React, { Component, ErrorInfo, ReactNode } from 'react';
import * as Sentry from '@sentry/react';

interface Props {
  children: ReactNode;
  fallback?: ReactNode | ((error: Error, resetError: () => void) => ReactNode);
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
  showDialog?: boolean;
  dialogOptions?: Sentry.ReportDialogOptions;
}

interface State {
  hasError: boolean;
  error: Error | null;
  errorInfo: ErrorInfo | null;
  eventId: string | null;
}

/**
 * Error Boundary component that catches JavaScript errors in child components
 * and reports them to Sentry
 */
export class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = {
      hasError: false,
      error: null,
      errorInfo: null,
      eventId: null,
    };
  }

  static getDerivedStateFromError(error: Error): Partial<State> {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    // Log error info
    console.error('ErrorBoundary caught an error:', error, errorInfo);

    // Capture to Sentry
    const eventId = Sentry.withScope((scope) => {
      scope.setExtras({
        componentStack: errorInfo.componentStack,
      });
      return Sentry.captureException(error);
    });

    this.setState({
      errorInfo,
      eventId: eventId || null,
    });

    // Call custom error handler
    this.props.onError?.(error, errorInfo);
  }

  resetError = (): void => {
    this.setState({
      hasError: false,
      error: null,
      errorInfo: null,
      eventId: null,
    });
  };

  showReportDialog = (): void => {
    if (this.state.eventId) {
      Sentry.showReportDialog({
        eventId: this.state.eventId,
        ...this.props.dialogOptions,
      });
    }
  };

  render(): ReactNode {
    const { hasError, error } = this.state;
    const { children, fallback, showDialog } = this.props;

    if (hasError && error) {
      // Show custom fallback if provided
      if (fallback) {
        if (typeof fallback === 'function') {
          return fallback(error, this.resetError);
        }
        return fallback;
      }

      // Default error UI
      return (
        <DefaultErrorFallback
          error={error}
          resetError={this.resetError}
          showReportDialog={showDialog ? this.showReportDialog : undefined}
        />
      );
    }

    return children;
  }
}

// Default error fallback component
interface DefaultErrorFallbackProps {
  error: Error;
  resetError: () => void;
  showReportDialog?: () => void;
}

function DefaultErrorFallback({
  error,
  resetError,
  showReportDialog,
}: DefaultErrorFallbackProps): JSX.Element {
  return (
    <div className="error-boundary-fallback">
      <div className="error-boundary-content">
        <div className="error-icon">
          <svg
            width="64"
            height="64"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <circle cx="12" cy="12" r="10" />
            <line x1="12" y1="8" x2="12" y2="12" />
            <line x1="12" y1="16" x2="12.01" y2="16" />
          </svg>
        </div>

        <h1>Something went wrong</h1>

        <p className="error-message">
          We're sorry, but something unexpected happened. Our team has been
          notified and is working to fix the issue.
        </p>

        {process.env.NODE_ENV === 'development' && (
          <details className="error-details">
            <summary>Error Details</summary>
            <pre>{error.message}</pre>
            <pre>{error.stack}</pre>
          </details>
        )}

        <div className="error-actions">
          <button onClick={resetError} className="btn-primary">
            Try Again
          </button>

          <button
            onClick={() => window.location.reload()}
            className="btn-secondary"
          >
            Reload Page
          </button>

          {showReportDialog && (
            <button onClick={showReportDialog} className="btn-outline">
              Report Issue
            </button>
          )}
        </div>
      </div>

      <style>{`
        .error-boundary-fallback {
          display: flex;
          align-items: center;
          justify-content: center;
          min-height: 400px;
          padding: 2rem;
          background-color: #f8fafc;
        }

        .error-boundary-content {
          max-width: 500px;
          text-align: center;
        }

        .error-icon {
          color: #ef4444;
          margin-bottom: 1.5rem;
        }

        .error-boundary-content h1 {
          font-size: 1.5rem;
          font-weight: 600;
          color: #1e293b;
          margin-bottom: 0.75rem;
        }

        .error-message {
          color: #64748b;
          margin-bottom: 1.5rem;
          line-height: 1.6;
        }

        .error-details {
          text-align: left;
          background-color: #fee2e2;
          border-radius: 0.5rem;
          padding: 1rem;
          margin-bottom: 1.5rem;
        }

        .error-details summary {
          cursor: pointer;
          font-weight: 500;
          color: #991b1b;
        }

        .error-details pre {
          margin-top: 0.5rem;
          font-size: 0.75rem;
          overflow-x: auto;
          white-space: pre-wrap;
          word-break: break-word;
        }

        .error-actions {
          display: flex;
          gap: 0.75rem;
          justify-content: center;
          flex-wrap: wrap;
        }

        .error-actions button {
          padding: 0.625rem 1.25rem;
          border-radius: 0.375rem;
          font-weight: 500;
          cursor: pointer;
          transition: all 0.2s;
        }

        .btn-primary {
          background-color: #2563eb;
          color: white;
          border: none;
        }

        .btn-primary:hover {
          background-color: #1d4ed8;
        }

        .btn-secondary {
          background-color: #e2e8f0;
          color: #475569;
          border: none;
        }

        .btn-secondary:hover {
          background-color: #cbd5e1;
        }

        .btn-outline {
          background-color: transparent;
          color: #2563eb;
          border: 1px solid #2563eb;
        }

        .btn-outline:hover {
          background-color: #eff6ff;
        }
      `}</style>
    </div>
  );
}

// Higher-order component version
export function withErrorBoundary<P extends object>(
  WrappedComponent: React.ComponentType<P>,
  errorBoundaryProps?: Omit<Props, 'children'>
): React.FC<P> {
  const displayName =
    WrappedComponent.displayName || WrappedComponent.name || 'Component';

  const ComponentWithErrorBoundary: React.FC<P> = (props) => (
    <ErrorBoundary {...errorBoundaryProps}>
      <WrappedComponent {...props} />
    </ErrorBoundary>
  );

  ComponentWithErrorBoundary.displayName = `withErrorBoundary(${displayName})`;

  return ComponentWithErrorBoundary;
}

// Sentry's error boundary wrapper for additional features
export const SentryErrorBoundary = Sentry.withErrorBoundary(ErrorBoundary, {
  showDialog: true,
});

export default ErrorBoundary;
