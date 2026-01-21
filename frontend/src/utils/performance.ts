/**
 * =============================================================================
 * Performance Utilities
 * =============================================================================
 * Core Web Vitals measurement and performance monitoring
 */

// =============================================================================
// Types
// =============================================================================

export interface WebVitalsMetric {
  name: 'CLS' | 'FCP' | 'FID' | 'INP' | 'LCP' | 'TTFB';
  value: number;
  rating: 'good' | 'needs-improvement' | 'poor';
  delta: number;
  id: string;
  navigationType: 'navigate' | 'reload' | 'back-forward' | 'prerender';
}

export interface PerformanceMetrics {
  // Core Web Vitals
  lcp?: number; // Largest Contentful Paint
  fid?: number; // First Input Delay
  cls?: number; // Cumulative Layout Shift
  inp?: number; // Interaction to Next Paint
  fcp?: number; // First Contentful Paint
  ttfb?: number; // Time to First Byte

  // Additional metrics
  domContentLoaded?: number;
  windowLoad?: number;
  firstPaint?: number;
  domInteractive?: number;

  // Resource metrics
  resourceCount?: number;
  transferSize?: number;
  decodedBodySize?: number;

  // Custom metrics
  routeChangeTime?: number;
  hydrationTime?: number;
}

export interface PerformanceThresholds {
  lcp: { good: number; poor: number };
  fid: { good: number; poor: number };
  cls: { good: number; poor: number };
  inp: { good: number; poor: number };
  fcp: { good: number; poor: number };
  ttfb: { good: number; poor: number };
}

type MetricCallback = (metric: WebVitalsMetric) => void;

// =============================================================================
// Thresholds (Based on Google's recommendations)
// =============================================================================

export const PERFORMANCE_THRESHOLDS: PerformanceThresholds = {
  lcp: { good: 2500, poor: 4000 }, // milliseconds
  fid: { good: 100, poor: 300 }, // milliseconds
  cls: { good: 0.1, poor: 0.25 }, // score
  inp: { good: 200, poor: 500 }, // milliseconds
  fcp: { good: 1800, poor: 3000 }, // milliseconds
  ttfb: { good: 800, poor: 1800 }, // milliseconds
};

// =============================================================================
// Core Web Vitals Measurement
// =============================================================================

/**
 * Get performance rating based on value and thresholds
 */
export function getPerformanceRating(
  name: keyof PerformanceThresholds,
  value: number
): 'good' | 'needs-improvement' | 'poor' {
  const thresholds = PERFORMANCE_THRESHOLDS[name];
  if (value <= thresholds.good) return 'good';
  if (value <= thresholds.poor) return 'needs-improvement';
  return 'poor';
}

/**
 * Observe Largest Contentful Paint
 */
export function observeLCP(callback: MetricCallback): () => void {
  if (typeof PerformanceObserver === 'undefined') return () => {};

  let lcp: PerformanceEntry | null = null;

  const observer = new PerformanceObserver((list) => {
    const entries = list.getEntries();
    lcp = entries[entries.length - 1];
  });

  observer.observe({ type: 'largest-contentful-paint', buffered: true });

  // Report on visibility change or page unload
  const report = () => {
    if (lcp) {
      const value = lcp.startTime;
      callback({
        name: 'LCP',
        value,
        rating: getPerformanceRating('lcp', value),
        delta: value,
        id: `lcp-${Date.now()}`,
        navigationType: getNavigationType(),
      });
    }
  };

  document.addEventListener('visibilitychange', report, { once: true });
  window.addEventListener('pagehide', report, { once: true });

  return () => observer.disconnect();
}

/**
 * Observe First Input Delay
 */
export function observeFID(callback: MetricCallback): () => void {
  if (typeof PerformanceObserver === 'undefined') return () => {};

  const observer = new PerformanceObserver((list) => {
    const entries = list.getEntries() as PerformanceEventTiming[];

    for (const entry of entries) {
      const value = entry.processingStart - entry.startTime;
      callback({
        name: 'FID',
        value,
        rating: getPerformanceRating('fid', value),
        delta: value,
        id: `fid-${Date.now()}`,
        navigationType: getNavigationType(),
      });
    }
  });

  observer.observe({ type: 'first-input', buffered: true });

  return () => observer.disconnect();
}

/**
 * Observe Cumulative Layout Shift
 */
export function observeCLS(callback: MetricCallback): () => void {
  if (typeof PerformanceObserver === 'undefined') return () => {};

  let clsValue = 0;
  let sessionValue = 0;
  let sessionEntries: PerformanceEntry[] = [];

  const observer = new PerformanceObserver((list) => {
    for (const entry of list.getEntries() as (PerformanceEntry & {
      hadRecentInput?: boolean;
      value?: number;
    })[]) {
      if (!entry.hadRecentInput) {
        const firstSessionEntry = sessionEntries[0];
        const lastSessionEntry = sessionEntries[sessionEntries.length - 1];

        // New session if gap > 1s or session > 5s
        if (
          sessionValue &&
          firstSessionEntry &&
          lastSessionEntry &&
          (entry.startTime - lastSessionEntry.startTime > 1000 ||
            entry.startTime - firstSessionEntry.startTime > 5000)
        ) {
          if (sessionValue > clsValue) {
            clsValue = sessionValue;
          }
          sessionValue = 0;
          sessionEntries = [];
        }

        sessionEntries.push(entry);
        sessionValue += entry.value || 0;
      }
    }
  });

  observer.observe({ type: 'layout-shift', buffered: true });

  const report = () => {
    if (sessionValue > clsValue) {
      clsValue = sessionValue;
    }

    callback({
      name: 'CLS',
      value: clsValue,
      rating: getPerformanceRating('cls', clsValue),
      delta: clsValue,
      id: `cls-${Date.now()}`,
      navigationType: getNavigationType(),
    });
  };

  document.addEventListener('visibilitychange', report, { once: true });
  window.addEventListener('pagehide', report, { once: true });

  return () => observer.disconnect();
}

/**
 * Observe Interaction to Next Paint
 */
export function observeINP(callback: MetricCallback): () => void {
  if (typeof PerformanceObserver === 'undefined') return () => {};

  const interactions: number[] = [];

  const observer = new PerformanceObserver((list) => {
    for (const entry of list.getEntries() as PerformanceEventTiming[]) {
      if (entry.interactionId) {
        const duration = entry.duration;
        interactions.push(duration);
      }
    }
  });

  observer.observe({
    type: 'event',
    buffered: true,
    durationThreshold: 16,
  } as PerformanceObserverInit);

  const report = () => {
    if (interactions.length === 0) return;

    // INP is the 98th percentile of interactions
    interactions.sort((a, b) => b - a);
    const index = Math.min(
      interactions.length - 1,
      Math.floor(interactions.length * 0.02)
    );
    const value = interactions[index];

    callback({
      name: 'INP',
      value,
      rating: getPerformanceRating('inp', value),
      delta: value,
      id: `inp-${Date.now()}`,
      navigationType: getNavigationType(),
    });
  };

  document.addEventListener('visibilitychange', report, { once: true });
  window.addEventListener('pagehide', report, { once: true });

  return () => observer.disconnect();
}

/**
 * Observe First Contentful Paint
 */
export function observeFCP(callback: MetricCallback): () => void {
  if (typeof PerformanceObserver === 'undefined') return () => {};

  const observer = new PerformanceObserver((list) => {
    for (const entry of list.getEntries()) {
      if (entry.name === 'first-contentful-paint') {
        const value = entry.startTime;
        callback({
          name: 'FCP',
          value,
          rating: getPerformanceRating('fcp', value),
          delta: value,
          id: `fcp-${Date.now()}`,
          navigationType: getNavigationType(),
        });
      }
    }
  });

  observer.observe({ type: 'paint', buffered: true });

  return () => observer.disconnect();
}

/**
 * Observe Time to First Byte
 */
export function observeTTFB(callback: MetricCallback): () => void {
  if (typeof performance === 'undefined') return () => {};

  const navEntry = performance.getEntriesByType(
    'navigation'
  )[0] as PerformanceNavigationTiming;

  if (navEntry) {
    const value = navEntry.responseStart - navEntry.requestStart;
    callback({
      name: 'TTFB',
      value,
      rating: getPerformanceRating('ttfb', value),
      delta: value,
      id: `ttfb-${Date.now()}`,
      navigationType: getNavigationType(),
    });
  }

  return () => {};
}

// =============================================================================
// All Web Vitals Observer
// =============================================================================

/**
 * Observe all Core Web Vitals
 */
export function observeWebVitals(callback: MetricCallback): () => void {
  const unsubscribers: (() => void)[] = [];

  unsubscribers.push(observeLCP(callback));
  unsubscribers.push(observeFID(callback));
  unsubscribers.push(observeCLS(callback));
  unsubscribers.push(observeINP(callback));
  unsubscribers.push(observeFCP(callback));
  unsubscribers.push(observeTTFB(callback));

  return () => unsubscribers.forEach((fn) => fn());
}

// =============================================================================
// Performance Metrics Collection
// =============================================================================

/**
 * Get all performance metrics
 */
export function getPerformanceMetrics(): PerformanceMetrics {
  if (typeof performance === 'undefined') return {};

  const metrics: PerformanceMetrics = {};

  // Navigation timing
  const navEntry = performance.getEntriesByType(
    'navigation'
  )[0] as PerformanceNavigationTiming;
  if (navEntry) {
    metrics.ttfb = navEntry.responseStart - navEntry.requestStart;
    metrics.domContentLoaded =
      navEntry.domContentLoadedEventEnd - navEntry.fetchStart;
    metrics.windowLoad = navEntry.loadEventEnd - navEntry.fetchStart;
    metrics.domInteractive = navEntry.domInteractive - navEntry.fetchStart;
  }

  // Paint timing
  const paintEntries = performance.getEntriesByType('paint');
  for (const entry of paintEntries) {
    if (entry.name === 'first-paint') {
      metrics.firstPaint = entry.startTime;
    }
    if (entry.name === 'first-contentful-paint') {
      metrics.fcp = entry.startTime;
    }
  }

  // Resource timing
  const resources = performance.getEntriesByType(
    'resource'
  ) as PerformanceResourceTiming[];
  metrics.resourceCount = resources.length;
  metrics.transferSize = resources.reduce(
    (sum, r) => sum + (r.transferSize || 0),
    0
  );
  metrics.decodedBodySize = resources.reduce(
    (sum, r) => sum + (r.decodedBodySize || 0),
    0
  );

  return metrics;
}

/**
 * Get navigation type
 */
function getNavigationType(): WebVitalsMetric['navigationType'] {
  if (typeof performance === 'undefined') return 'navigate';

  const navEntry = performance.getEntriesByType(
    'navigation'
  )[0] as PerformanceNavigationTiming;
  if (!navEntry) return 'navigate';

  if (navEntry.type === 'reload') return 'reload';
  if (navEntry.type === 'back_forward') return 'back-forward';
  if (document.prerendering) return 'prerender';

  return 'navigate';
}

// =============================================================================
// Custom Performance Marks
// =============================================================================

/**
 * Mark a custom performance point
 */
export function markPerformance(name: string): void {
  if (typeof performance === 'undefined') return;
  performance.mark(name);
}

/**
 * Measure between two marks
 */
export function measurePerformance(
  name: string,
  startMark: string,
  endMark?: string
): number | null {
  if (typeof performance === 'undefined') return null;

  try {
    if (endMark) {
      performance.measure(name, startMark, endMark);
    } else {
      performance.measure(name, startMark);
    }

    const measures = performance.getEntriesByName(name, 'measure');
    return measures.length > 0 ? measures[measures.length - 1].duration : null;
  } catch {
    return null;
  }
}

/**
 * Clear performance marks
 */
export function clearPerformanceMarks(name?: string): void {
  if (typeof performance === 'undefined') return;

  if (name) {
    performance.clearMarks(name);
    performance.clearMeasures(name);
  } else {
    performance.clearMarks();
    performance.clearMeasures();
  }
}

// =============================================================================
// Route Change Performance
// =============================================================================

let routeChangeStart: number | null = null;

/**
 * Start measuring route change
 */
export function startRouteChange(): void {
  routeChangeStart = performance.now();
  markPerformance('route-change-start');
}

/**
 * End route change measurement
 */
export function endRouteChange(): number | null {
  if (routeChangeStart === null) return null;

  markPerformance('route-change-end');
  const duration = measurePerformance(
    'route-change',
    'route-change-start',
    'route-change-end'
  );

  routeChangeStart = null;
  return duration;
}

// =============================================================================
// Component Render Performance
// =============================================================================

const renderTimes = new Map<string, number[]>();

/**
 * Track component render time
 */
export function trackRender(componentName: string, duration: number): void {
  const times = renderTimes.get(componentName) || [];
  times.push(duration);

  // Keep last 100 measurements
  if (times.length > 100) {
    times.shift();
  }

  renderTimes.set(componentName, times);
}

/**
 * Get render statistics for a component
 */
export function getRenderStats(componentName: string): {
  count: number;
  average: number;
  min: number;
  max: number;
  p95: number;
} | null {
  const times = renderTimes.get(componentName);
  if (!times || times.length === 0) return null;

  const sorted = [...times].sort((a, b) => a - b);
  const sum = sorted.reduce((a, b) => a + b, 0);

  return {
    count: sorted.length,
    average: sum / sorted.length,
    min: sorted[0],
    max: sorted[sorted.length - 1],
    p95: sorted[Math.floor(sorted.length * 0.95)],
  };
}

// =============================================================================
// Long Task Detection
// =============================================================================

/**
 * Observe long tasks (> 50ms)
 */
export function observeLongTasks(
  callback: (entry: PerformanceEntry) => void
): () => void {
  if (typeof PerformanceObserver === 'undefined') return () => {};

  const observer = new PerformanceObserver((list) => {
    for (const entry of list.getEntries()) {
      callback(entry);
    }
  });

  observer.observe({ type: 'longtask', buffered: true });

  return () => observer.disconnect();
}

// =============================================================================
// Memory Usage
// =============================================================================

interface MemoryInfo {
  usedJSHeapSize: number;
  totalJSHeapSize: number;
  jsHeapSizeLimit: number;
}

/**
 * Get memory usage (Chrome only)
 */
export function getMemoryUsage(): MemoryInfo | null {
  if (typeof performance === 'undefined') return null;

  const memory = (performance as Performance & { memory?: MemoryInfo }).memory;
  if (!memory) return null;

  return {
    usedJSHeapSize: memory.usedJSHeapSize,
    totalJSHeapSize: memory.totalJSHeapSize,
    jsHeapSizeLimit: memory.jsHeapSizeLimit,
  };
}

// =============================================================================
// Performance Score
// =============================================================================

/**
 * Calculate overall performance score (0-100)
 */
export function calculatePerformanceScore(metrics: {
  lcp?: number;
  fid?: number;
  cls?: number;
  fcp?: number;
  ttfb?: number;
}): number {
  const scores: number[] = [];

  // LCP score (weight: 25%)
  if (metrics.lcp !== undefined) {
    if (metrics.lcp <= 2500) scores.push(100);
    else if (metrics.lcp <= 4000) scores.push(50);
    else scores.push(0);
  }

  // FID score (weight: 25%)
  if (metrics.fid !== undefined) {
    if (metrics.fid <= 100) scores.push(100);
    else if (metrics.fid <= 300) scores.push(50);
    else scores.push(0);
  }

  // CLS score (weight: 25%)
  if (metrics.cls !== undefined) {
    if (metrics.cls <= 0.1) scores.push(100);
    else if (metrics.cls <= 0.25) scores.push(50);
    else scores.push(0);
  }

  // FCP score (weight: 15%)
  if (metrics.fcp !== undefined) {
    if (metrics.fcp <= 1800) scores.push(100);
    else if (metrics.fcp <= 3000) scores.push(50);
    else scores.push(0);
  }

  // TTFB score (weight: 10%)
  if (metrics.ttfb !== undefined) {
    if (metrics.ttfb <= 800) scores.push(100);
    else if (metrics.ttfb <= 1800) scores.push(50);
    else scores.push(0);
  }

  if (scores.length === 0) return 0;

  return Math.round(scores.reduce((a, b) => a + b, 0) / scores.length);
}

// =============================================================================
// Reporter
// =============================================================================

/**
 * Report metrics to analytics
 */
export function reportMetrics(
  metrics: WebVitalsMetric | PerformanceMetrics,
  endpoint?: string
): void {
  // Log in development
  if (process.env.NODE_ENV === 'development') {
    console.log('[Performance]', metrics);
  }

  // Send to analytics endpoint
  if (
    endpoint &&
    typeof navigator !== 'undefined' &&
    'sendBeacon' in navigator
  ) {
    const data = JSON.stringify({
      ...metrics,
      timestamp: Date.now(),
      url: window.location.href,
      userAgent: navigator.userAgent,
    });

    navigator.sendBeacon(endpoint, data);
  }
}
