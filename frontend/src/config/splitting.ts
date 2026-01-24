/**
 * =============================================================================
 * Code Splitting Configuration
 * =============================================================================
 * Configuration for lazy loading, preloading, and chunk management
 */

import { ComponentType, lazy } from 'react';

// =============================================================================
// Types
// =============================================================================

export interface RouteConfig {
  path: string;
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  component: () => Promise<{ default: ComponentType<any> }>;
  preload?: 'eager' | 'hover' | 'visible' | 'none';
  chunkName?: string;
  priority?: 'high' | 'medium' | 'low';
}

export interface PreloadOptions {
  delay?: number;
  timeout?: number;
}

// =============================================================================
// Route Definitions with Lazy Loading
// =============================================================================

export const routeConfigs: Record<string, RouteConfig> = {
  // Public routes
  landing: {
    path: '/',
    component: () =>
      import(/* webpackChunkName: "page-landing" */ '@pages/Landing').then(
        (m) => ({ default: m.LandingPage })
      ),
    preload: 'eager',
    priority: 'high',
  },
  login: {
    path: '/login',
    component: () =>
      import(/* webpackChunkName: "page-auth" */ '@pages/Login').then((m) => ({
        default: m.LoginPage,
      })),
    preload: 'hover',
    priority: 'high',
  },
  register: {
    path: '/register',
    component: () =>
      import(/* webpackChunkName: "page-auth" */ '@pages/Register').then(
        (m) => ({ default: m.RegisterPage })
      ),
    preload: 'hover',
    priority: 'medium',
  },
  forgotPassword: {
    path: '/forgot-password',
    component: () =>
      import(/* webpackChunkName: "page-auth" */ '@pages/ForgotPassword').then(
        (m) => ({ default: m.ForgotPasswordPage })
      ),
    preload: 'hover',
    priority: 'low',
  },
  resetPassword: {
    path: '/reset-password',
    component: () =>
      import(/* webpackChunkName: "page-auth" */ '@pages/ResetPassword').then(
        (m) => ({ default: m.ResetPasswordPage })
      ),
    preload: 'none',
    priority: 'low',
  },
  verifyEmail: {
    path: '/verify-email',
    component: () =>
      import(/* webpackChunkName: "page-auth" */ '@pages/VerifyEmail').then(
        (m) => ({ default: m.VerifyEmailPage })
      ),
    preload: 'none',
    priority: 'low',
  },

  // Protected routes
  dashboard: {
    path: '/dashboard',
    component: () =>
      import(/* webpackChunkName: "page-dashboard" */ '@pages/Dashboard').then(
        (m) => ({ default: m.DashboardPage })
      ),
    preload: 'eager',
    priority: 'high',
  },
  settings: {
    path: '/settings',
    component: () =>
      import(/* webpackChunkName: "page-settings" */ '@pages/Settings').then(
        (m) => ({ default: m.SettingsPage })
      ),
    preload: 'hover',
    priority: 'medium',
  },
  revenueReport: {
    path: '/reports/revenue',
    component: () =>
      import(/* webpackChunkName: "page-reports" */ '@pages/Reports').then(
        (m) => ({ default: m.RevenueReportPage })
      ),
    preload: 'hover',
    priority: 'medium',
  },
  customerReport: {
    path: '/reports/customers',
    component: () =>
      import(/* webpackChunkName: "page-reports" */ '@pages/Reports').then(
        (m) => ({ default: m.CustomerReportPage })
      ),
    preload: 'hover',
    priority: 'medium',
  },

  // Error routes
  notFound: {
    path: '/404',
    component: () =>
      import(/* webpackChunkName: "page-error" */ '@pages/NotFound').then(
        (m) => ({ default: m.NotFoundPage })
      ),
    preload: 'none',
    priority: 'low',
  },
  unauthorized: {
    path: '/unauthorized',
    component: () =>
      import(/* webpackChunkName: "page-error" */ '@pages/Unauthorized').then(
        (m) => ({ default: m.UnauthorizedPage })
      ),
    preload: 'none',
    priority: 'low',
  },
};

// =============================================================================
// Preload Strategies
// =============================================================================

/**
 * Preload cache to prevent duplicate loading
 */
const preloadedModules = new Set<string>();

/**
 * Preload a route component
 */
export function preloadRoute(
  routeKey: string,
  options: PreloadOptions = {}
): Promise<void> {
  const config = routeConfigs[routeKey];
  if (!config || preloadedModules.has(routeKey)) {
    return Promise.resolve();
  }

  const { delay = 0, timeout = 5000 } = options;

  return new Promise((resolve) => {
    const timeoutId = setTimeout(() => {
      preloadedModules.add(routeKey);

      const loadPromise = config.component();

      // Use requestIdleCallback if available for low-priority routes
      if (config.priority === 'low' && 'requestIdleCallback' in window) {
        (
          window as Window & {
            requestIdleCallback: (
              cb: () => void,
              opts?: { timeout: number }
            ) => void;
          }
        ).requestIdleCallback(
          () => {
            loadPromise.then(() => resolve()).catch(() => resolve());
          },
          { timeout }
        );
      } else {
        loadPromise.then(() => resolve()).catch(() => resolve());
      }
    }, delay);

    // Cleanup on page unload
    if (typeof window !== 'undefined') {
      window.addEventListener('beforeunload', () => clearTimeout(timeoutId), {
        once: true,
      });
    }
  });
}

/**
 * Preload multiple routes
 */
export function preloadRoutes(
  routeKeys: string[],
  options: PreloadOptions = {}
): Promise<void[]> {
  return Promise.all(routeKeys.map((key) => preloadRoute(key, options)));
}

/**
 * Preload routes based on priority
 */
export function preloadByPriority(
  priority: 'high' | 'medium' | 'low'
): Promise<void[]> {
  const routes = Object.entries(routeConfigs)
    .filter(
      ([, config]) => config.priority === priority && config.preload !== 'none'
    )
    .map(([key]) => key);

  return preloadRoutes(routes);
}

/**
 * Preload all high-priority routes (call on app mount)
 */
export function preloadEagerRoutes(): void {
  // Preload high priority routes immediately
  preloadByPriority('high');

  // Preload medium priority routes after idle
  if ('requestIdleCallback' in window) {
    (
      window as Window & { requestIdleCallback: (cb: () => void) => void }
    ).requestIdleCallback(() => {
      preloadByPriority('medium');
    });
  } else {
    setTimeout(() => preloadByPriority('medium'), 2000);
  }
}

// =============================================================================
// Lazy Component Factory
// =============================================================================

/**
 * Create a lazy component with proper chunk naming
 */
export function createLazyComponent<T extends ComponentType<unknown>>(
  routeKey: string
): React.LazyExoticComponent<T> {
  const config = routeConfigs[routeKey];
  if (!config) {
    throw new Error(`Route config not found for: ${routeKey}`);
  }

  return lazy(config.component as () => Promise<{ default: T }>);
}

// =============================================================================
// Preload Hooks Integration
// =============================================================================

/**
 * Get preload handler for link hover
 */
export function getPreloadHandler(routeKey: string): () => void {
  let preloadTimeout: ReturnType<typeof setTimeout> | null = null;

  return () => {
    if (preloadTimeout) return;

    preloadTimeout = setTimeout(() => {
      preloadRoute(routeKey, { delay: 0 });
      preloadTimeout = null;
    }, 100); // Small delay to avoid preloading on quick mouse movements
  };
}

// =============================================================================
// Chunk Configuration
// =============================================================================

export const chunkConfig = {
  // Minimum size for a chunk to be created (in bytes)
  minSize: 20000,

  // Maximum size before warning (in bytes)
  maxSize: 244000,

  // Maximum number of parallel requests for on-demand chunks
  maxAsyncRequests: 6,

  // Maximum number of parallel requests at entry point
  maxInitialRequests: 4,

  // Chunk naming configuration
  names: {
    vendor: 'vendor',
    common: 'common',
    runtime: 'runtime',
  },

  // Split configuration for vendor chunks
  vendorGroups: {
    react: ['react', 'react-dom', 'react-router-dom'],
    ui: ['@headlessui/react', 'lucide-react', 'clsx'],
    forms: ['react-hook-form', '@hookform/resolvers', 'zod'],
    data: ['@tanstack/react-query', 'zustand', 'axios'],
    charts: ['chart.js', 'react-chartjs-2'],
    dates: ['date-fns', 'react-datepicker'],
    stripe: ['@stripe/react-stripe-js', '@stripe/stripe-js'],
  },
};

// =============================================================================
// Performance Monitoring
// =============================================================================

/**
 * Track chunk load performance
 */
export function trackChunkLoad(chunkName: string, startTime: number): void {
  const duration = performance.now() - startTime;

  // Log to console in development
  if (process.env.NODE_ENV === 'development') {
    // eslint-disable-next-line no-console
    console.log(`[Chunk] ${chunkName} loaded in ${duration.toFixed(2)}ms`);
  }

  // Report to analytics if available
  if (typeof window !== 'undefined' && 'performance' in window) {
    performance.mark(`chunk-${chunkName}-loaded`);
    performance.measure(`chunk-${chunkName}-load`, {
      start: startTime,
      end: performance.now(),
    });
  }
}

/**
 * Get chunk load metrics
 */
export function getChunkMetrics(): PerformanceEntryList {
  if (typeof window === 'undefined' || !('performance' in window)) {
    return [];
  }

  return performance
    .getEntriesByType('measure')
    .filter((entry) => entry.name.startsWith('chunk-'));
}
