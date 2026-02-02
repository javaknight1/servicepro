/* eslint-disable react-refresh/only-export-components */
/**
 * Analytics Context Provider
 *
 * Provides analytics functionality throughout the React application
 */

import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useCallback,
  ReactNode,
} from 'react';
import { useLocation } from 'react-router-dom';
import {
  analytics,
  AnalyticsConfig,
  EventCategory,
  EventName,
  EventProperties,
} from '../services/analytics';

// Context value interface
interface AnalyticsContextValue {
  // Core tracking methods
  track: (
    name: EventName | string,
    category?: EventCategory,
    properties?: EventProperties
  ) => void;
  pageView: (
    path: string,
    title?: string,
    properties?: EventProperties
  ) => void;

  // User tracking
  setUserId: (userId: string | null) => void;
  setTenantId: (tenantId: string | null) => void;
  setConsent: (hasConsent: boolean) => void;

  // Convenience methods
  trackClick: (
    buttonId: string,
    buttonText?: string,
    properties?: EventProperties
  ) => void;
  trackFormSubmit: (
    formId: string,
    formName?: string,
    success?: boolean
  ) => void;
  trackSearch: (searchTerm: string, resultCount?: number) => void;
  trackFeature: (
    featureName: string,
    action: string,
    properties?: EventProperties
  ) => void;
  trackBusiness: (
    eventName: EventName,
    entityId: string,
    entityType: string,
    value?: number,
    properties?: EventProperties
  ) => void;
  trackTiming: (
    category: string,
    variable: string,
    timeMs: number,
    label?: string
  ) => void;
  trackError: (
    errorType: string,
    errorMessage: string,
    properties?: EventProperties
  ) => void;

  // Auth tracking
  trackLogin: (method?: string, userId?: string) => void;
  trackSignup: (method?: string, userId?: string) => void;
  trackLogout: () => void;

  // Session info
  getSessionId: () => string;

  // Utility
  flush: () => Promise<void>;
  reset: () => void;
}

// Create context
const AnalyticsContext = createContext<AnalyticsContextValue | undefined>(
  undefined
);

// Provider props
interface AnalyticsProviderProps {
  children: ReactNode;
  config?: Partial<AnalyticsConfig>;
  userId?: string | null;
  tenantId?: string | null;
  hasConsent?: boolean;
  trackPageViews?: boolean;
}

/**
 * Analytics Provider Component
 */
export function AnalyticsProvider({
  children,
  config,
  userId,
  tenantId,
  hasConsent = false,
  trackPageViews = true,
}: AnalyticsProviderProps) {
  const location = useLocation();

  // Initialize analytics on mount
  useEffect(() => {
    if (config) {
      analytics.init(config);
    }
  }, [config]);

  // Set user ID when it changes
  useEffect(() => {
    analytics.setUserId(userId || null);
  }, [userId]);

  // Set tenant ID when it changes
  useEffect(() => {
    analytics.setTenantId(tenantId || null);
  }, [tenantId]);

  // Set consent when it changes
  useEffect(() => {
    analytics.setConsent(hasConsent);
  }, [hasConsent]);

  // Track page views on route changes
  useEffect(() => {
    if (trackPageViews) {
      analytics.pageView(location.pathname + location.search);
    }
  }, [location.pathname, location.search, trackPageViews]);

  // Memoized context value
  const value = useMemo<AnalyticsContextValue>(
    () => ({
      track: analytics.track.bind(analytics),
      pageView: analytics.pageView.bind(analytics),
      setUserId: analytics.setUserId.bind(analytics),
      setTenantId: analytics.setTenantId.bind(analytics),
      setConsent: analytics.setConsent.bind(analytics),
      trackClick: analytics.trackClick.bind(analytics),
      trackFormSubmit: analytics.trackFormSubmit.bind(analytics),
      trackSearch: analytics.trackSearch.bind(analytics),
      trackFeature: analytics.trackFeature.bind(analytics),
      trackBusiness: analytics.trackBusiness.bind(analytics),
      trackTiming: analytics.trackTiming.bind(analytics),
      trackError: analytics.trackError.bind(analytics),
      trackLogin: analytics.trackLogin.bind(analytics),
      trackSignup: analytics.trackSignup.bind(analytics),
      trackLogout: analytics.trackLogout.bind(analytics),
      getSessionId: analytics.getSessionId.bind(analytics),
      flush: analytics.flush.bind(analytics),
      reset: analytics.reset.bind(analytics),
    }),
    []
  );

  return (
    <AnalyticsContext.Provider value={value}>
      {children}
    </AnalyticsContext.Provider>
  );
}

/**
 * Hook to access analytics context
 */
export function useAnalytics(): AnalyticsContextValue {
  const context = useContext(AnalyticsContext);

  if (!context) {
    throw new Error('useAnalytics must be used within an AnalyticsProvider');
  }

  return context;
}

/**
 * Hook for tracking page views with custom data
 */
export function usePageTracking(
  pageName: string,
  properties?: EventProperties
) {
  const { pageView } = useAnalytics();
  const location = useLocation();

  useEffect(() => {
    pageView(location.pathname, pageName, properties);
  }, [location.pathname, pageName, properties, pageView]);
}

/**
 * Hook for tracking component lifecycle
 */
export function useComponentTracking(componentName: string) {
  const { track } = useAnalytics();

  useEffect(() => {
    track('feature_used', 'feature', {
      feature_name: componentName,
      action: 'mounted',
    });

    return () => {
      track('feature_used', 'feature', {
        feature_name: componentName,
        action: 'unmounted',
      });
    };
  }, [componentName, track]);
}

/**
 * Hook for tracking user interactions
 */
export function useInteractionTracking() {
  const { trackClick, trackFormSubmit } = useAnalytics();

  const trackButtonClick = useCallback(
    (
      buttonId: string,
      buttonText?: string,
      additionalProps?: EventProperties
    ) => {
      trackClick(buttonId, buttonText, additionalProps);
    },
    [trackClick]
  );

  const trackForm = useCallback(
    (formId: string, formName?: string, success: boolean = true) => {
      trackFormSubmit(formId, formName, success);
    },
    [trackFormSubmit]
  );

  return {
    trackButtonClick,
    trackForm,
  };
}

/**
 * Hook for tracking API calls
 */
export function useApiTracking() {
  const { trackTiming, trackError } = useAnalytics();

  const trackApiCall = useCallback(
    async <T,>(
      apiName: string,
      endpoint: string,
      apiCall: () => Promise<T>
    ): Promise<T> => {
      const startTime = performance.now();

      try {
        const result = await apiCall();
        const duration = performance.now() - startTime;

        trackTiming('api', apiName, Math.round(duration), endpoint);

        return result;
      } catch (error) {
        const duration = performance.now() - startTime;

        trackTiming('api', apiName, Math.round(duration), endpoint);
        trackError('api_error', (error as Error).message, {
          api_name: apiName,
          endpoint,
        });

        throw error;
      }
    },
    [trackTiming, trackError]
  );

  return { trackApiCall };
}

/**
 * Hook for tracking search functionality
 */
export function useSearchTracking() {
  const { trackSearch: trackSearchEvent } = useAnalytics();

  const trackSearch = useCallback(
    (searchTerm: string, resultCount?: number, _searchType?: string) => {
      trackSearchEvent(searchTerm, resultCount);
    },
    [trackSearchEvent]
  );

  return { trackSearch };
}

/**
 * Hook for tracking business events
 */
export function useBusinessTracking() {
  const { trackBusiness } = useAnalytics();

  const trackCustomerCreated = useCallback(
    (customerId: string, properties?: EventProperties) => {
      trackBusiness(
        'customer_created',
        customerId,
        'customer',
        undefined,
        properties
      );
    },
    [trackBusiness]
  );

  const trackJobScheduled = useCallback(
    (jobId: string, value?: number, properties?: EventProperties) => {
      trackBusiness('job_scheduled', jobId, 'job', value, properties);
    },
    [trackBusiness]
  );

  const trackJobCompleted = useCallback(
    (jobId: string, value?: number, properties?: EventProperties) => {
      trackBusiness('job_completed', jobId, 'job', value, properties);
    },
    [trackBusiness]
  );

  const trackQuoteSent = useCallback(
    (quoteId: string, value?: number, properties?: EventProperties) => {
      trackBusiness('quote_sent', quoteId, 'quote', value, properties);
    },
    [trackBusiness]
  );

  const trackQuoteAccepted = useCallback(
    (quoteId: string, value?: number, properties?: EventProperties) => {
      trackBusiness('quote_accepted', quoteId, 'quote', value, properties);
    },
    [trackBusiness]
  );

  const trackInvoiceCreated = useCallback(
    (invoiceId: string, value?: number, properties?: EventProperties) => {
      trackBusiness('invoice_created', invoiceId, 'invoice', value, properties);
    },
    [trackBusiness]
  );

  const trackPaymentReceived = useCallback(
    (paymentId: string, value?: number, properties?: EventProperties) => {
      trackBusiness(
        'payment_received',
        paymentId,
        'payment',
        value,
        properties
      );
    },
    [trackBusiness]
  );

  return {
    trackCustomerCreated,
    trackJobScheduled,
    trackJobCompleted,
    trackQuoteSent,
    trackQuoteAccepted,
    trackInvoiceCreated,
    trackPaymentReceived,
  };
}

// Re-export types
export type { AnalyticsConfig, EventCategory, EventName, EventProperties };
