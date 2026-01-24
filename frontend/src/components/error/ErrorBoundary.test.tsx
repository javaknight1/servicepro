import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { vi } from 'vitest';
import { ErrorBoundary, withErrorBoundary } from './ErrorBoundary';

// Mock Sentry
vi.mock('@sentry/react', () => ({
  withScope: vi.fn((fn) => fn({ setExtras: vi.fn() })),
  captureException: vi.fn(() => 'mock-event-id'),
  showReportDialog: vi.fn(),
  withErrorBoundary: vi.fn((Component) => Component),
}));

// Component that throws an error
const ThrowError: React.FC<{ shouldThrow?: boolean }> = ({
  shouldThrow = true,
}) => {
  if (shouldThrow) {
    throw new Error('Test error');
  }
  return <div>No error</div>;
};

// Suppress console.error for expected errors
const originalError = console.error;
beforeAll(() => {
  console.error = vi.fn();
});

afterAll(() => {
  console.error = originalError;
});

describe('ErrorBoundary', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders children when there is no error', () => {
    render(
      <ErrorBoundary>
        <div>Test content</div>
      </ErrorBoundary>
    );

    expect(screen.getByText('Test content')).toBeInTheDocument();
  });

  it('renders default fallback when there is an error', () => {
    render(
      <ErrorBoundary>
        <ThrowError />
      </ErrorBoundary>
    );

    expect(screen.getByText('Something went wrong')).toBeInTheDocument();
  });

  it('renders custom fallback component', () => {
    render(
      <ErrorBoundary fallback={<div>Custom error message</div>}>
        <ThrowError />
      </ErrorBoundary>
    );

    expect(screen.getByText('Custom error message')).toBeInTheDocument();
  });

  it('renders custom fallback function with error and reset', () => {
    const fallbackFn = vi.fn((error: Error, resetError: () => void) => (
      <div>
        <span>Error: {error.message}</span>
        <button onClick={resetError}>Reset</button>
      </div>
    ));

    render(
      <ErrorBoundary fallback={fallbackFn}>
        <ThrowError />
      </ErrorBoundary>
    );

    expect(screen.getByText('Error: Test error')).toBeInTheDocument();
    expect(fallbackFn).toHaveBeenCalled();
  });

  it('calls onError callback when error occurs', () => {
    const onError = vi.fn();

    render(
      <ErrorBoundary onError={onError}>
        <ThrowError />
      </ErrorBoundary>
    );

    expect(onError).toHaveBeenCalledWith(
      expect.any(Error),
      expect.objectContaining({
        componentStack: expect.any(String),
      })
    );
  });

  it('resets error state when resetError is called', () => {
    const TestComponent: React.FC = () => {
      const [shouldThrow, setShouldThrow] = React.useState(true);

      return (
        <ErrorBoundary
          fallback={(error, resetError) => (
            <button
              onClick={() => {
                setShouldThrow(false);
                resetError();
              }}
            >
              Reset
            </button>
          )}
        >
          <ThrowError shouldThrow={shouldThrow} />
        </ErrorBoundary>
      );
    };

    render(<TestComponent />);

    // Error state
    expect(screen.getByText('Reset')).toBeInTheDocument();

    // Click reset
    fireEvent.click(screen.getByText('Reset'));

    // Should render normally now
    expect(screen.getByText('No error')).toBeInTheDocument();
  });

  it('shows Try Again button in default fallback', () => {
    render(
      <ErrorBoundary>
        <ThrowError />
      </ErrorBoundary>
    );

    expect(screen.getByText('Try Again')).toBeInTheDocument();
  });

  it('shows Reload Page button in default fallback', () => {
    render(
      <ErrorBoundary>
        <ThrowError />
      </ErrorBoundary>
    );

    expect(screen.getByText('Reload Page')).toBeInTheDocument();
  });

  it('shows error details in development mode', () => {
    const originalEnv = process.env.NODE_ENV;
    process.env.NODE_ENV = 'development';

    render(
      <ErrorBoundary>
        <ThrowError />
      </ErrorBoundary>
    );

    expect(screen.getByText('Error Details')).toBeInTheDocument();

    process.env.NODE_ENV = originalEnv;
  });

  it('captures error to Sentry', () => {
    const Sentry = require('@sentry/react');

    render(
      <ErrorBoundary>
        <ThrowError />
      </ErrorBoundary>
    );

    expect(Sentry.withScope).toHaveBeenCalled();
    expect(Sentry.captureException).toHaveBeenCalled();
  });
});

describe('withErrorBoundary HOC', () => {
  it('wraps component with error boundary', () => {
    const TestComponent: React.FC = () => <div>Test</div>;
    const WrappedComponent = withErrorBoundary(TestComponent);

    render(<WrappedComponent />);

    expect(screen.getByText('Test')).toBeInTheDocument();
  });

  it('catches errors in wrapped component', () => {
    const WrappedComponent = withErrorBoundary(ThrowError);

    render(<WrappedComponent />);

    expect(screen.getByText('Something went wrong')).toBeInTheDocument();
  });

  it('passes error boundary props to wrapper', () => {
    const onError = vi.fn();
    const WrappedComponent = withErrorBoundary(ThrowError, { onError });

    render(<WrappedComponent />);

    expect(onError).toHaveBeenCalled();
  });

  it('has correct display name', () => {
    const TestComponent: React.FC = () => <div>Test</div>;
    TestComponent.displayName = 'TestComponent';

    const WrappedComponent = withErrorBoundary(TestComponent);

    expect(WrappedComponent.displayName).toBe(
      'withErrorBoundary(TestComponent)'
    );
  });
});

describe('ErrorBoundary with nested errors', () => {
  it('catches deeply nested errors', () => {
    const DeepChild: React.FC = () => {
      throw new Error('Deep error');
    };

    const Parent: React.FC = () => (
      <div>
        <DeepChild />
      </div>
    );

    render(
      <ErrorBoundary>
        <Parent />
      </ErrorBoundary>
    );

    expect(screen.getByText('Something went wrong')).toBeInTheDocument();
  });

  it('allows nested error boundaries', () => {
    const InnerThrow: React.FC = () => {
      throw new Error('Inner error');
    };

    render(
      <ErrorBoundary fallback={<div>Outer fallback</div>}>
        <div>
          <ErrorBoundary fallback={<div>Inner fallback</div>}>
            <InnerThrow />
          </ErrorBoundary>
        </div>
      </ErrorBoundary>
    );

    // Inner boundary should catch the error
    expect(screen.getByText('Inner fallback')).toBeInTheDocument();
    expect(screen.queryByText('Outer fallback')).not.toBeInTheDocument();
  });
});
