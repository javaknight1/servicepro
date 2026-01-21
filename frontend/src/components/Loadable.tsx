/**
 * =============================================================================
 * Loadable Component
 * =============================================================================
 * Wrapper for lazy-loaded components with loading states and error boundaries
 */

import React, {
  Suspense,
  ComponentType,
  ReactNode,
  Component,
  ErrorInfo,
  lazy,
  useState,
  useEffect,
} from 'react';
import { trackChunkLoad } from '@config/splitting';

// =============================================================================
// Types
// =============================================================================

interface LoadableOptions {
  fallback?: ReactNode;
  errorFallback?: ComponentType<ErrorFallbackProps>;
  delay?: number;
  timeout?: number;
  chunkName?: string;
}

interface ErrorFallbackProps {
  error: Error;
  resetError: () => void;
}

interface ErrorBoundaryProps {
  children: ReactNode;
  fallback?: ComponentType<ErrorFallbackProps>;
  onError?: (error: Error, errorInfo: ErrorInfo) => void;
}

interface ErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

// =============================================================================
// Default Loading Component
// =============================================================================

function DefaultLoadingSpinner() {
  return (
    <div className="flex items-center justify-center min-h-[200px] w-full">
      <div className="relative">
        <div className="w-12 h-12 rounded-full border-4 border-gray-200" />
        <div className="absolute top-0 left-0 w-12 h-12 rounded-full border-4 border-blue-600 border-t-transparent animate-spin" />
      </div>
    </div>
  );
}

// =============================================================================
// Delayed Loading Component
// =============================================================================

interface DelayedFallbackProps {
  delay: number;
  children: ReactNode;
}

function DelayedFallback({ delay, children }: DelayedFallbackProps) {
  const [showFallback, setShowFallback] = useState(false);

  useEffect(() => {
    const timer = setTimeout(() => setShowFallback(true), delay);
    return () => clearTimeout(timer);
  }, [delay]);

  if (!showFallback) {
    return null;
  }

  return <>{children}</>;
}

// =============================================================================
// Default Error Fallback
// =============================================================================

function DefaultErrorFallback({ error, resetError }: ErrorFallbackProps) {
  return (
    <div className="flex flex-col items-center justify-center min-h-[200px] w-full p-6">
      <div className="text-red-500 mb-4">
        <svg
          className="w-16 h-16"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
          />
        </svg>
      </div>
      <h3 className="text-lg font-semibold text-gray-900 mb-2">
        Failed to load component
      </h3>
      <p className="text-sm text-gray-600 mb-4 text-center max-w-md">
        {error.message ||
          'An unexpected error occurred while loading this component.'}
      </p>
      <button
        onClick={resetError}
        className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
      >
        Try Again
      </button>
    </div>
  );
}

// =============================================================================
// Error Boundary Component
// =============================================================================

class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): ErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo): void {
    // Log error to console in development
    if (process.env.NODE_ENV === 'development') {
      console.error('Loadable Error:', error);
      console.error('Component Stack:', errorInfo.componentStack);
    }

    // Call optional error handler
    this.props.onError?.(error, errorInfo);

    // Report to error tracking service
    if (typeof window !== 'undefined' && 'reportError' in window) {
      (window as Window & { reportError?: (e: Error) => void }).reportError?.(
        error
      );
    }
  }

  resetError = (): void => {
    this.setState({ hasError: false, error: null });
  };

  render(): ReactNode {
    if (this.state.hasError && this.state.error) {
      const FallbackComponent = this.props.fallback || DefaultErrorFallback;
      return (
        <FallbackComponent
          error={this.state.error}
          resetError={this.resetError}
        />
      );
    }

    return this.props.children;
  }
}

// =============================================================================
// Loadable Factory
// =============================================================================

/**
 * Creates a loadable component with Suspense and Error Boundary
 */
export function loadable<P extends object>(
  importFn: () => Promise<{ default: ComponentType<P> }>,
  options: LoadableOptions = {}
): ComponentType<P> {
  const {
    fallback,
    errorFallback,
    delay = 200,
    timeout,
    chunkName = 'unknown',
  } = options;

  // Track load time
  const startTime = performance.now();

  // Create lazy component with timeout
  const LazyComponent = lazy(async () => {
    try {
      // Create timeout promise if specified
      const loadPromise = importFn();

      if (timeout) {
        const timeoutPromise = new Promise<never>((_, reject) => {
          setTimeout(
            () =>
              reject(
                new Error(
                  `Chunk "${chunkName}" load timeout after ${timeout}ms`
                )
              ),
            timeout
          );
        });

        const module = await Promise.race([loadPromise, timeoutPromise]);
        trackChunkLoad(chunkName, startTime);
        return module;
      }

      const module = await loadPromise;
      trackChunkLoad(chunkName, startTime);
      return module;
    } catch (error) {
      // Re-throw for error boundary to catch
      throw error;
    }
  });

  // Create wrapper component
  function LoadableComponent(props: P): JSX.Element {
    const loadingFallback = fallback || <DefaultLoadingSpinner />;
    const delayedFallback =
      delay > 0 ? (
        <DelayedFallback delay={delay}>{loadingFallback}</DelayedFallback>
      ) : (
        loadingFallback
      );

    return (
      <ErrorBoundary fallback={errorFallback}>
        <Suspense fallback={delayedFallback}>
          <LazyComponent {...props} />
        </Suspense>
      </ErrorBoundary>
    );
  }

  // Set display name for debugging
  LoadableComponent.displayName = `Loadable(${chunkName})`;

  return LoadableComponent;
}

// =============================================================================
// Page Loading Component
// =============================================================================

export function PageLoader() {
  return (
    <div className="flex flex-col items-center justify-center min-h-screen bg-gray-50">
      <div className="relative mb-4">
        <div className="w-16 h-16 rounded-full border-4 border-gray-200" />
        <div className="absolute top-0 left-0 w-16 h-16 rounded-full border-4 border-blue-600 border-t-transparent animate-spin" />
      </div>
      <p className="text-gray-600 text-sm">Loading...</p>
    </div>
  );
}

// =============================================================================
// Skeleton Components
// =============================================================================

interface SkeletonProps {
  className?: string;
  animate?: boolean;
}

export function Skeleton({ className = '', animate = true }: SkeletonProps) {
  return (
    <div
      className={`bg-gray-200 rounded ${
        animate ? 'animate-pulse' : ''
      } ${className}`}
    />
  );
}

export function CardSkeleton() {
  return (
    <div className="bg-white rounded-lg shadow p-6">
      <Skeleton className="h-6 w-1/3 mb-4" />
      <Skeleton className="h-4 w-full mb-2" />
      <Skeleton className="h-4 w-2/3 mb-4" />
      <div className="flex gap-2">
        <Skeleton className="h-8 w-20" />
        <Skeleton className="h-8 w-20" />
      </div>
    </div>
  );
}

export function TableSkeleton({ rows = 5 }: { rows?: number }) {
  return (
    <div className="bg-white rounded-lg shadow overflow-hidden">
      <div className="px-6 py-4 border-b">
        <Skeleton className="h-6 w-1/4" />
      </div>
      <div className="divide-y">
        {Array.from({ length: rows }).map((_, i) => (
          <div key={i} className="px-6 py-4 flex gap-4">
            <Skeleton className="h-4 w-1/4" />
            <Skeleton className="h-4 w-1/3" />
            <Skeleton className="h-4 w-1/6" />
            <Skeleton className="h-4 w-1/6" />
          </div>
        ))}
      </div>
    </div>
  );
}

export function DashboardSkeleton() {
  return (
    <div className="space-y-6">
      {/* Stats row */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="bg-white rounded-lg shadow p-6">
            <Skeleton className="h-4 w-20 mb-2" />
            <Skeleton className="h-8 w-24" />
          </div>
        ))}
      </div>

      {/* Main content */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <CardSkeleton />
        <CardSkeleton />
      </div>

      {/* Table */}
      <TableSkeleton />
    </div>
  );
}

// =============================================================================
// Retry Logic
// =============================================================================

interface RetryOptions {
  retries?: number;
  delay?: number;
  onRetry?: (attempt: number, error: Error) => void;
}

/**
 * Creates a loadable component with retry logic
 */
export function loadableWithRetry<P extends object>(
  importFn: () => Promise<{ default: ComponentType<P> }>,
  loadableOptions: LoadableOptions = {},
  retryOptions: RetryOptions = {}
): ComponentType<P> {
  const { retries = 3, delay = 1000, onRetry } = retryOptions;

  const retryImport = async (): Promise<{ default: ComponentType<P> }> => {
    let lastError: Error | null = null;

    for (let attempt = 1; attempt <= retries; attempt++) {
      try {
        return await importFn();
      } catch (error) {
        lastError = error as Error;

        if (attempt < retries) {
          onRetry?.(attempt, lastError);

          // Wait before retry with exponential backoff
          await new Promise((resolve) => setTimeout(resolve, delay * attempt));
        }
      }
    }

    throw lastError;
  };

  return loadable(retryImport, loadableOptions);
}

// =============================================================================
// Preload Component
// =============================================================================

interface PreloadLinkProps
  extends React.AnchorHTMLAttributes<HTMLAnchorElement> {
  onPreload?: () => void;
  preloadDelay?: number;
  children: ReactNode;
}

export function PreloadLink({
  onPreload,
  preloadDelay = 100,
  children,
  onMouseEnter,
  onFocus,
  ...props
}: PreloadLinkProps) {
  const [preloaded, setPreloaded] = useState(false);

  const handlePreload = () => {
    if (!preloaded && onPreload) {
      setTimeout(() => {
        onPreload();
        setPreloaded(true);
      }, preloadDelay);
    }
  };

  return (
    <a
      {...props}
      onMouseEnter={(e) => {
        handlePreload();
        onMouseEnter?.(e);
      }}
      onFocus={(e) => {
        handlePreload();
        onFocus?.(e);
      }}
    >
      {children}
    </a>
  );
}

// =============================================================================
// Exports
// =============================================================================

export { ErrorBoundary, DefaultLoadingSpinner, DefaultErrorFallback };
export type { LoadableOptions, ErrorFallbackProps, ErrorBoundaryProps };
