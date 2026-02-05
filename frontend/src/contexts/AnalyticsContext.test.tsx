import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { vi } from 'vitest';
import {
  AnalyticsProvider,
  useAnalytics,
  usePageTracking,
  useComponentTracking,
  useInteractionTracking,
  useBusinessTracking,
  useApiTracking,
  useSearchTracking,
} from './AnalyticsContext';
import { analytics } from '../services/analytics';

// Mock the analytics service
vi.mock('../services/analytics', () => ({
  analytics: {
    init: vi.fn(),
    setUserId: vi.fn(),
    setTenantId: vi.fn(),
    setConsent: vi.fn(),
    track: vi.fn(),
    pageView: vi.fn(),
    trackClick: vi.fn(),
    trackFormSubmit: vi.fn(),
    trackSearch: vi.fn(),
    trackFeature: vi.fn(),
    trackBusiness: vi.fn(),
    trackTiming: vi.fn(),
    trackError: vi.fn(),
    trackLogin: vi.fn(),
    trackSignup: vi.fn(),
    trackLogout: vi.fn(),
    getSessionId: vi.fn(() => 'mock-session-id'),
    flush: vi.fn(() => Promise.resolve()),
    reset: vi.fn(),
  },
}));

// Mock react-router-dom
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useLocation: vi.fn(() => ({
      pathname: '/test-path',
      search: '?query=value',
    })),
  };
});

// Helper component to test hooks
function TestComponent() {
  const {
    track,
    trackClick,
    trackFormSubmit,
    trackSearch,
    trackFeature,
    trackError,
    trackLogin,
    trackLogout,
    getSessionId,
  } = useAnalytics();

  return (
    <div>
      <button onClick={() => track('test_event', 'feature', { key: 'value' })}>
        Track Event
      </button>
      <button onClick={() => trackClick('btn-1', 'Click Me')}>
        Track Click
      </button>
      <button onClick={() => trackFormSubmit('form-1', 'Contact Form', true)}>
        Track Form
      </button>
      <button onClick={() => trackSearch('test query', 10)}>
        Track Search
      </button>
      <button onClick={() => trackFeature('dashboard', 'viewed')}>
        Track Feature
      </button>
      <button onClick={() => trackError('TestError', 'Something went wrong')}>
        Track Error
      </button>
      <button onClick={() => trackLogin('email', 'user-123')}>
        Track Login
      </button>
      <button onClick={() => trackLogout()}>Track Logout</button>
      <span data-testid="session-id">{getSessionId()}</span>
    </div>
  );
}

// Helper wrapper
const renderWithProvider = (
  ui: React.ReactElement,
  { providerProps = {}, ...options } = {}
) => {
  return render(
    <MemoryRouter>
      <AnalyticsProvider {...providerProps}>{ui}</AnalyticsProvider>
    </MemoryRouter>,
    options
  );
};

describe('AnalyticsProvider', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('initializes with config', () => {
    const config = {
      enabled: true,
      debug: true,
      apiEndpoint: '/api/analytics',
    };

    renderWithProvider(<div>Test</div>, {
      providerProps: { config },
    });

    expect(analytics.init).toHaveBeenCalledWith(config);
  });

  it('sets user ID when provided', () => {
    renderWithProvider(<div>Test</div>, {
      providerProps: { userId: 'user-123' },
    });

    expect(analytics.setUserId).toHaveBeenCalledWith('user-123');
  });

  it('sets tenant ID when provided', () => {
    renderWithProvider(<div>Test</div>, {
      providerProps: { tenantId: 'tenant-456' },
    });

    expect(analytics.setTenantId).toHaveBeenCalledWith('tenant-456');
  });

  it('sets consent status', () => {
    renderWithProvider(<div>Test</div>, {
      providerProps: { hasConsent: true },
    });

    expect(analytics.setConsent).toHaveBeenCalledWith(true);
  });

  it('tracks page views on route changes by default', () => {
    renderWithProvider(<div>Test</div>);

    expect(analytics.pageView).toHaveBeenCalledWith('/test-path?query=value');
  });

  it('does not track page views when disabled', () => {
    renderWithProvider(<div>Test</div>, {
      providerProps: { trackPageViews: false },
    });

    expect(analytics.pageView).not.toHaveBeenCalled();
  });
});

describe('useAnalytics', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('throws error when used outside provider', () => {
    // Suppress console.error for expected error
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation();

    // In React 18, errors thrown during render are handled by React's error boundaries
    // So we need to catch the error differently
    let thrownError: Error | null = null;
    try {
      render(
        <MemoryRouter>
          <TestComponent />
        </MemoryRouter>
      );
    } catch (error) {
      thrownError = error as Error;
    }

    expect(thrownError).not.toBeNull();
    expect(thrownError?.message).toBe(
      'useAnalytics must be used within an AnalyticsProvider'
    );

    consoleSpy.mockRestore();
  });

  it('provides track function', () => {
    renderWithProvider(<TestComponent />);

    fireEvent.click(screen.getByText('Track Event'));

    expect(analytics.track).toHaveBeenCalledWith('test_event', 'feature', {
      key: 'value',
    });
  });

  it('provides trackClick function', () => {
    renderWithProvider(<TestComponent />);

    fireEvent.click(screen.getByText('Track Click'));

    expect(analytics.trackClick).toHaveBeenCalledWith('btn-1', 'Click Me');
  });

  it('provides trackFormSubmit function', () => {
    renderWithProvider(<TestComponent />);

    fireEvent.click(screen.getByText('Track Form'));

    expect(analytics.trackFormSubmit).toHaveBeenCalledWith(
      'form-1',
      'Contact Form',
      true
    );
  });

  it('provides trackSearch function', () => {
    renderWithProvider(<TestComponent />);

    fireEvent.click(screen.getByText('Track Search'));

    expect(analytics.trackSearch).toHaveBeenCalledWith('test query', 10);
  });

  it('provides trackFeature function', () => {
    renderWithProvider(<TestComponent />);

    fireEvent.click(screen.getByText('Track Feature'));

    expect(analytics.trackFeature).toHaveBeenCalledWith('dashboard', 'viewed');
  });

  it('provides trackError function', () => {
    renderWithProvider(<TestComponent />);

    fireEvent.click(screen.getByText('Track Error'));

    expect(analytics.trackError).toHaveBeenCalledWith(
      'TestError',
      'Something went wrong'
    );
  });

  it('provides trackLogin function', () => {
    renderWithProvider(<TestComponent />);

    fireEvent.click(screen.getByText('Track Login'));

    expect(analytics.trackLogin).toHaveBeenCalledWith('email', 'user-123');
  });

  it('provides trackLogout function', () => {
    renderWithProvider(<TestComponent />);

    fireEvent.click(screen.getByText('Track Logout'));

    expect(analytics.trackLogout).toHaveBeenCalled();
  });

  it('provides getSessionId function', () => {
    renderWithProvider(<TestComponent />);

    expect(screen.getByTestId('session-id')).toHaveTextContent(
      'mock-session-id'
    );
  });
});

describe('usePageTracking', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('tracks page view on mount', () => {
    function PageTrackingComponent() {
      usePageTracking('Test Page', { custom: 'property' });
      return <div>Page Content</div>;
    }

    renderWithProvider(<PageTrackingComponent />);

    expect(analytics.pageView).toHaveBeenCalledWith('/test-path', 'Test Page', {
      custom: 'property',
    });
  });
});

describe('useComponentTracking', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('tracks component mount and unmount', () => {
    function TrackedComponent() {
      useComponentTracking('TrackedComponent');
      return <div>Tracked</div>;
    }

    const { unmount } = renderWithProvider(<TrackedComponent />);

    expect(analytics.track).toHaveBeenCalledWith('feature_used', 'feature', {
      feature_name: 'TrackedComponent',
      action: 'mounted',
    });

    unmount();

    expect(analytics.track).toHaveBeenCalledWith('feature_used', 'feature', {
      feature_name: 'TrackedComponent',
      action: 'unmounted',
    });
  });
});

describe('useInteractionTracking', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('provides trackButtonClick and trackForm functions', () => {
    function InteractionComponent() {
      const { trackButtonClick, trackForm } = useInteractionTracking();

      return (
        <div>
          <button onClick={() => trackButtonClick('btn-test', 'Test Button')}>
            Button Click
          </button>
          <button onClick={() => trackForm('form-test', 'Test Form', true)}>
            Form Submit
          </button>
        </div>
      );
    }

    renderWithProvider(<InteractionComponent />);

    fireEvent.click(screen.getByText('Button Click'));
    expect(analytics.trackClick).toHaveBeenCalledWith(
      'btn-test',
      'Test Button',
      undefined
    );

    fireEvent.click(screen.getByText('Form Submit'));
    expect(analytics.trackFormSubmit).toHaveBeenCalledWith(
      'form-test',
      'Test Form',
      true
    );
  });
});

describe('useBusinessTracking', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('provides business tracking functions', () => {
    function BusinessComponent() {
      const {
        trackCustomerCreated,
        trackJobScheduled,
        trackJobCompleted,
        trackQuoteSent,
        trackQuoteAccepted,
        trackInvoiceCreated,
        trackPaymentReceived,
      } = useBusinessTracking();

      return (
        <div>
          <button onClick={() => trackCustomerCreated('cust-1')}>
            Customer
          </button>
          <button onClick={() => trackJobScheduled('job-1', 100)}>
            Job Scheduled
          </button>
          <button onClick={() => trackJobCompleted('job-1', 100)}>
            Job Completed
          </button>
          <button onClick={() => trackQuoteSent('quote-1', 500)}>
            Quote Sent
          </button>
          <button onClick={() => trackQuoteAccepted('quote-1', 500)}>
            Quote Accepted
          </button>
          <button onClick={() => trackInvoiceCreated('inv-1', 500)}>
            Invoice
          </button>
          <button onClick={() => trackPaymentReceived('pay-1', 500)}>
            Payment
          </button>
        </div>
      );
    }

    renderWithProvider(<BusinessComponent />);

    fireEvent.click(screen.getByText('Customer'));
    expect(analytics.trackBusiness).toHaveBeenCalledWith(
      'customer_created',
      'cust-1',
      'customer',
      undefined,
      undefined
    );

    fireEvent.click(screen.getByText('Job Scheduled'));
    expect(analytics.trackBusiness).toHaveBeenCalledWith(
      'job_scheduled',
      'job-1',
      'job',
      100,
      undefined
    );

    fireEvent.click(screen.getByText('Job Completed'));
    expect(analytics.trackBusiness).toHaveBeenCalledWith(
      'job_completed',
      'job-1',
      'job',
      100,
      undefined
    );

    fireEvent.click(screen.getByText('Quote Sent'));
    expect(analytics.trackBusiness).toHaveBeenCalledWith(
      'quote_sent',
      'quote-1',
      'quote',
      500,
      undefined
    );

    fireEvent.click(screen.getByText('Quote Accepted'));
    expect(analytics.trackBusiness).toHaveBeenCalledWith(
      'quote_accepted',
      'quote-1',
      'quote',
      500,
      undefined
    );

    fireEvent.click(screen.getByText('Invoice'));
    expect(analytics.trackBusiness).toHaveBeenCalledWith(
      'invoice_created',
      'inv-1',
      'invoice',
      500,
      undefined
    );

    fireEvent.click(screen.getByText('Payment'));
    expect(analytics.trackBusiness).toHaveBeenCalledWith(
      'payment_received',
      'pay-1',
      'payment',
      500,
      undefined
    );
  });
});

describe('AnalyticsProvider edge cases', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('handles null userId', () => {
    const { rerender } = renderWithProvider(<div>Test</div>, {
      providerProps: { userId: 'user-123' },
    });

    expect(analytics.setUserId).toHaveBeenCalledWith('user-123');

    // Simulate logout
    rerender(
      <MemoryRouter>
        <AnalyticsProvider userId={null}>
          <div>Test</div>
        </AnalyticsProvider>
      </MemoryRouter>
    );

    expect(analytics.setUserId).toHaveBeenCalledWith(null);
  });

  it('handles consent changes', () => {
    const { rerender } = renderWithProvider(<div>Test</div>, {
      providerProps: { hasConsent: false },
    });

    expect(analytics.setConsent).toHaveBeenCalledWith(false);

    // Update consent
    rerender(
      <MemoryRouter>
        <AnalyticsProvider hasConsent={true}>
          <div>Test</div>
        </AnalyticsProvider>
      </MemoryRouter>
    );

    expect(analytics.setConsent).toHaveBeenCalledWith(true);
  });
});

describe('useApiTracking', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('tracks successful API calls', async () => {
    function ApiTrackingComponent() {
      const { trackApiCall } = useApiTracking();
      const [result, setResult] = React.useState<string | null>(null);

      const handleClick = async () => {
        const data = await trackApiCall('getUsers', '/api/users', async () => {
          return { users: [] };
        });
        setResult(JSON.stringify(data));
      };

      return (
        <div>
          <button onClick={handleClick}>Make API Call</button>
          {result && <span data-testid="result">{result}</span>}
        </div>
      );
    }

    renderWithProvider(<ApiTrackingComponent />);

    await fireEvent.click(screen.getByText('Make API Call'));

    // Wait for the result
    await vi.waitFor(() => {
      expect(screen.getByTestId('result')).toBeInTheDocument();
    });

    expect(analytics.trackTiming).toHaveBeenCalledWith(
      'api',
      'getUsers',
      expect.any(Number),
      '/api/users'
    );
  });

  it('tracks failed API calls', async () => {
    function ApiTrackingErrorComponent() {
      const { trackApiCall } = useApiTracking();
      const [error, setError] = React.useState<string | null>(null);

      const handleClick = async () => {
        try {
          await trackApiCall('failedCall', '/api/fail', async () => {
            throw new Error('API Error');
          });
        } catch (e) {
          setError((e as Error).message);
        }
      };

      return (
        <div>
          <button onClick={handleClick}>Make Failed API Call</button>
          {error && <span data-testid="error">{error}</span>}
        </div>
      );
    }

    renderWithProvider(<ApiTrackingErrorComponent />);

    await fireEvent.click(screen.getByText('Make Failed API Call'));

    await vi.waitFor(() => {
      expect(screen.getByTestId('error')).toBeInTheDocument();
    });

    expect(analytics.trackTiming).toHaveBeenCalled();
    expect(analytics.trackError).toHaveBeenCalledWith(
      'api_error',
      'API Error',
      expect.objectContaining({
        api_name: 'failedCall',
        endpoint: '/api/fail',
      })
    );
  });
});

describe('useSearchTracking', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('tracks search with result count', () => {
    function SearchTrackingComponent() {
      const { trackSearch } = useSearchTracking();

      return (
        <button onClick={() => trackSearch('test query', 5, 'users')}>
          Search
        </button>
      );
    }

    renderWithProvider(<SearchTrackingComponent />);

    fireEvent.click(screen.getByText('Search'));

    expect(analytics.trackSearch).toHaveBeenCalledWith('test query', 5);
  });

  it('tracks search without result count', () => {
    function SearchTrackingNoResultComponent() {
      const { trackSearch } = useSearchTracking();

      return (
        <button onClick={() => trackSearch('another query')}>Search</button>
      );
    }

    renderWithProvider(<SearchTrackingNoResultComponent />);

    fireEvent.click(screen.getByText('Search'));

    expect(analytics.trackSearch).toHaveBeenCalledWith(
      'another query',
      undefined
    );
  });
});
