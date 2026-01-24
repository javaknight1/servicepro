/**
 * Analytics Service
 *
 * Provides event tracking and analytics functionality
 * Supports both custom backend analytics and Google Analytics 4 (GA4)
 */

// Event categories
export type EventCategory =
  | 'user'
  | 'page'
  | 'feature'
  | 'business'
  | 'performance'
  | 'error'
  | 'api';

// Standard event names
export type EventName =
  // User events
  | 'user_signup'
  | 'user_login'
  | 'user_logout'
  | 'user_profile_view'
  | 'user_settings_edit'
  // Page events
  | 'page_view'
  | 'page_scroll'
  | 'page_exit'
  // Feature events
  | 'feature_used'
  | 'button_click'
  | 'form_submit'
  | 'search_performed'
  | 'filter_applied'
  | 'export_generated'
  // Business events
  | 'customer_created'
  | 'customer_updated'
  | 'job_scheduled'
  | 'job_completed'
  | 'job_cancelled'
  | 'quote_sent'
  | 'quote_accepted'
  | 'quote_declined'
  | 'invoice_created'
  | 'invoice_sent'
  | 'payment_received'
  | 'payment_failed'
  // Performance events
  | 'api_latency'
  | 'page_load_time'
  | 'database_query'
  | 'cache_hit'
  | 'cache_miss'
  // Error events
  | 'error_occurred'
  | 'error_resolved';

// Event properties
export interface EventProperties {
  [key: string]: string | number | boolean | null | undefined;
}

// Analytics event
export interface AnalyticsEvent {
  name: EventName | string;
  category: EventCategory;
  properties?: EventProperties;
  timestamp?: Date;
  userId?: string;
  sessionId?: string;
  tenantId?: string;
}

// Analytics configuration
export interface AnalyticsConfig {
  // Enable/disable analytics
  enabled: boolean;

  // Backend API endpoint for custom analytics
  apiEndpoint?: string;

  // GA4 Measurement ID
  ga4MeasurementId?: string;

  // Enable GA4 tracking
  enableGA4: boolean;

  // Enable backend tracking
  enableBackend: boolean;

  // Debug mode
  debug: boolean;

  // Batch size for backend events
  batchSize: number;

  // Flush interval in milliseconds
  flushInterval: number;

  // Sample rate (0.0 to 1.0)
  sampleRate: number;

  // Default properties to include with every event
  defaultProperties?: EventProperties;

  // User consent status
  hasConsent: boolean;
}

// Default configuration
const defaultConfig: AnalyticsConfig = {
  enabled: true,
  enableGA4: true,
  enableBackend: true,
  debug: process.env.NODE_ENV === 'development',
  batchSize: 10,
  flushInterval: 5000,
  sampleRate: 1.0,
  hasConsent: false,
};

// Analytics service class
class AnalyticsService {
  private config: AnalyticsConfig;
  private eventQueue: AnalyticsEvent[] = [];
  private flushTimer: ReturnType<typeof setInterval> | null = null;
  private sessionId: string;
  private userId: string | null = null;
  private tenantId: string | null = null;

  constructor(config: Partial<AnalyticsConfig> = {}) {
    this.config = { ...defaultConfig, ...config };
    this.sessionId = this.generateSessionId();

    if (this.config.enabled && this.config.enableBackend) {
      this.startFlushTimer();
    }
  }

  /**
   * Initialize analytics with configuration
   */
  init(config: Partial<AnalyticsConfig>): void {
    this.config = { ...this.config, ...config };

    if (this.config.enabled) {
      if (this.config.enableGA4 && this.config.ga4MeasurementId) {
        this.initGA4();
      }

      if (this.config.enableBackend) {
        this.startFlushTimer();
      }
    }

    if (this.config.debug) {
      // eslint-disable-next-line no-console
      console.log('[Analytics] Initialized with config:', this.config);
    }
  }

  /**
   * Initialize Google Analytics 4
   */
  private initGA4(): void {
    if (typeof window === 'undefined') return;

    const measurementId = this.config.ga4MeasurementId;
    if (!measurementId) return;

    // Load gtag script if not already loaded
    if (!window.gtag) {
      const script = document.createElement('script');
      script.async = true;
      script.src = `https://www.googletagmanager.com/gtag/js?id=${measurementId}`;
      document.head.appendChild(script);

      window.dataLayer = window.dataLayer || [];
      window.gtag = function gtag(...args: unknown[]) {
        window.dataLayer.push(args);
      };
      window.gtag('js', new Date());
      window.gtag('config', measurementId, {
        send_page_view: false, // We'll handle page views manually
      });
    }

    if (this.config.debug) {
      // eslint-disable-next-line no-console
      console.log('[Analytics] GA4 initialized with:', measurementId);
    }
  }

  /**
   * Set user ID for tracking
   */
  setUserId(userId: string | null): void {
    this.userId = userId;

    if (this.config.enableGA4 && window.gtag) {
      if (userId) {
        window.gtag('config', this.config.ga4MeasurementId!, {
          user_id: userId,
        });
      }
    }

    if (this.config.debug) {
      // eslint-disable-next-line no-console
      console.log('[Analytics] User ID set:', userId);
    }
  }

  /**
   * Set tenant ID for multi-tenant tracking
   */
  setTenantId(tenantId: string | null): void {
    this.tenantId = tenantId;

    if (this.config.debug) {
      // eslint-disable-next-line no-console
      console.log('[Analytics] Tenant ID set:', tenantId);
    }
  }

  /**
   * Set user consent status
   */
  setConsent(hasConsent: boolean): void {
    this.config.hasConsent = hasConsent;

    if (this.config.enableGA4 && window.gtag) {
      window.gtag('consent', 'update', {
        analytics_storage: hasConsent ? 'granted' : 'denied',
      });
    }

    if (this.config.debug) {
      // eslint-disable-next-line no-console
      console.log('[Analytics] Consent status:', hasConsent);
    }
  }

  /**
   * Track an event
   */
  track(
    name: EventName | string,
    category: EventCategory = 'feature',
    properties?: EventProperties
  ): void {
    if (!this.config.enabled) return;
    if (!this.config.hasConsent && this.requiresConsent(category)) return;
    if (!this.shouldSample()) return;

    const event: AnalyticsEvent = {
      name,
      category,
      properties: {
        ...this.config.defaultProperties,
        ...properties,
      },
      timestamp: new Date(),
      userId: this.userId || undefined,
      sessionId: this.sessionId,
      tenantId: this.tenantId || undefined,
    };

    // Track to GA4
    if (this.config.enableGA4 && typeof window.gtag === 'function') {
      this.trackToGA4(event);
    }

    // Queue for backend
    if (this.config.enableBackend) {
      this.queueEvent(event);
    }

    if (this.config.debug) {
      // eslint-disable-next-line no-console
      console.log('[Analytics] Event tracked:', event);
    }
  }

  /**
   * Track a page view
   */
  pageView(path: string, title?: string, properties?: EventProperties): void {
    this.track('page_view', 'page', {
      page_path: path,
      page_title: title || document.title,
      page_location: window.location.href,
      ...properties,
    });

    // GA4 page view
    if (this.config.enableGA4 && window.gtag) {
      window.gtag('event', 'page_view', {
        page_path: path,
        page_title: title || document.title,
        page_location: window.location.href,
      });
    }
  }

  /**
   * Track user login
   */
  trackLogin(method: string = 'email', userId?: string): void {
    if (userId) this.setUserId(userId);

    this.track('user_login', 'user', { method });

    if (this.config.enableGA4 && window.gtag) {
      window.gtag('event', 'login', { method });
    }
  }

  /**
   * Track user signup
   */
  trackSignup(method: string = 'email', userId?: string): void {
    if (userId) this.setUserId(userId);

    this.track('user_signup', 'user', { method });

    if (this.config.enableGA4 && window.gtag) {
      window.gtag('event', 'sign_up', { method });
    }
  }

  /**
   * Track user logout
   */
  trackLogout(): void {
    this.track('user_logout', 'user');
    this.setUserId(null);
  }

  /**
   * Track button click
   */
  trackClick(
    buttonId: string,
    buttonText?: string,
    properties?: EventProperties
  ): void {
    this.track('button_click', 'feature', {
      button_id: buttonId,
      button_text: buttonText,
      ...properties,
    });
  }

  /**
   * Track form submission
   */
  trackFormSubmit(
    formId: string,
    formName?: string,
    success: boolean = true
  ): void {
    this.track('form_submit', 'feature', {
      form_id: formId,
      form_name: formName,
      success,
    });
  }

  /**
   * Track search
   */
  trackSearch(searchTerm: string, resultCount?: number): void {
    this.track('search_performed', 'feature', {
      search_term: searchTerm,
      result_count: resultCount,
    });

    if (this.config.enableGA4 && window.gtag) {
      window.gtag('event', 'search', { search_term: searchTerm });
    }
  }

  /**
   * Track feature usage
   */
  trackFeature(
    featureName: string,
    action: string,
    properties?: EventProperties
  ): void {
    this.track('feature_used', 'feature', {
      feature_name: featureName,
      action,
      ...properties,
    });
  }

  /**
   * Track business event
   */
  trackBusiness(
    eventName: EventName,
    entityId: string,
    entityType: string,
    value?: number,
    properties?: EventProperties
  ): void {
    this.track(eventName, 'business', {
      entity_id: entityId,
      entity_type: entityType,
      value,
      ...properties,
    });
  }

  /**
   * Track performance timing
   */
  trackTiming(
    category: string,
    variable: string,
    timeMs: number,
    label?: string
  ): void {
    this.track('api_latency', 'performance', {
      timing_category: category,
      timing_variable: variable,
      timing_value: timeMs,
      timing_label: label,
    });

    if (this.config.enableGA4 && window.gtag) {
      window.gtag('event', 'timing_complete', {
        name: variable,
        value: timeMs,
        event_category: category,
        event_label: label,
      });
    }
  }

  /**
   * Track error
   */
  trackError(
    errorType: string,
    errorMessage: string,
    properties?: EventProperties
  ): void {
    this.track('error_occurred', 'error', {
      error_type: errorType,
      error_message: errorMessage,
      ...properties,
    });

    if (this.config.enableGA4 && window.gtag) {
      window.gtag('event', 'exception', {
        description: `${errorType}: ${errorMessage}`,
        fatal: false,
      });
    }
  }

  /**
   * Flush queued events to backend
   */
  async flush(): Promise<void> {
    if (this.eventQueue.length === 0) return;
    if (!this.config.apiEndpoint) return;

    const events = [...this.eventQueue];
    this.eventQueue = [];

    try {
      const response = await fetch(this.config.apiEndpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ events }),
      });

      if (!response.ok) {
        throw new Error(`Analytics API error: ${response.status}`);
      }

      if (this.config.debug) {
        // eslint-disable-next-line no-console
        console.log('[Analytics] Flushed', events.length, 'events');
      }
    } catch (error) {
      // Re-queue events on failure
      this.eventQueue = [...events, ...this.eventQueue];

      if (this.config.debug) {
        console.error('[Analytics] Flush failed:', error);
      }
    }
  }

  /**
   * Reset analytics state (e.g., on logout)
   */
  reset(): void {
    this.userId = null;
    this.sessionId = this.generateSessionId();
    this.eventQueue = [];
  }

  /**
   * Get current session ID
   */
  getSessionId(): string {
    return this.sessionId;
  }

  // Private methods

  private generateSessionId(): string {
    return `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
  }

  private shouldSample(): boolean {
    return Math.random() < this.config.sampleRate;
  }

  private requiresConsent(category: EventCategory): boolean {
    // Categories that require consent
    return ['user', 'page'].includes(category);
  }

  private trackToGA4(event: AnalyticsEvent): void {
    if (!window.gtag) return;

    window.gtag('event', event.name, {
      event_category: event.category,
      ...event.properties,
    });
  }

  private queueEvent(event: AnalyticsEvent): void {
    this.eventQueue.push(event);

    if (this.eventQueue.length >= this.config.batchSize) {
      this.flush();
    }
  }

  private startFlushTimer(): void {
    if (this.flushTimer) return;

    this.flushTimer = setInterval(() => {
      this.flush();
    }, this.config.flushInterval);

    // Flush on page unload
    if (typeof window !== 'undefined') {
      window.addEventListener('beforeunload', () => this.flush());
      window.addEventListener('visibilitychange', () => {
        if (document.visibilityState === 'hidden') {
          this.flush();
        }
      });
    }
  }
}

// Singleton instance
export const analytics = new AnalyticsService();

// Export class for custom instances
export { AnalyticsService };

// Type declarations for gtag
declare global {
  interface Window {
    gtag: (...args: unknown[]) => void;
    dataLayer: unknown[];
  }
}
