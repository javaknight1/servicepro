/**
 * =============================================================================
 * Loading Configuration
 * =============================================================================
 * Configuration for resource loading, prefetching, and optimization
 */

// =============================================================================
// Types
// =============================================================================

export interface ResourceHint {
  href: string;
  as?: 'script' | 'style' | 'font' | 'image' | 'fetch' | 'document';
  type?: string;
  crossOrigin?: 'anonymous' | 'use-credentials';
  media?: string;
}

export interface FontConfig {
  family: string;
  src: string;
  weight?: string | number;
  style?: string;
  display?: 'auto' | 'block' | 'swap' | 'fallback' | 'optional';
  preload?: boolean;
  unicodeRange?: string;
}

export interface ImageConfig {
  breakpoints: number[];
  formats: ('webp' | 'avif' | 'jpeg' | 'png')[];
  quality: number;
  placeholder: 'blur' | 'color' | 'none';
  lazyOffset: string;
  fadeInDuration: number;
}

// =============================================================================
// Resource Hints Configuration
// =============================================================================

export const resourceHints: {
  preconnect: ResourceHint[];
  dnsPrefetch: ResourceHint[];
  preload: ResourceHint[];
  prefetch: ResourceHint[];
} = {
  // Preconnect to critical third-party origins
  preconnect: [
    { href: 'https://fonts.googleapis.com', crossOrigin: 'anonymous' },
    { href: 'https://fonts.gstatic.com', crossOrigin: 'anonymous' },
    { href: 'https://js.stripe.com', crossOrigin: 'anonymous' },
  ],

  // DNS prefetch for less critical origins
  dnsPrefetch: [
    { href: 'https://www.google-analytics.com' },
    { href: 'https://cdn.segment.com' },
  ],

  // Preload critical resources
  preload: [
    {
      href: '/fonts/inter-var.woff2',
      as: 'font',
      type: 'font/woff2',
      crossOrigin: 'anonymous',
    },
  ],

  // Prefetch likely next resources
  prefetch: [{ href: '/assets/js/vendor-charts.js', as: 'script' }],
};

// =============================================================================
// Font Loading Configuration
// =============================================================================

export const fontConfig: {
  fonts: FontConfig[];
  fallbackFonts: string[];
  loadingStrategy: 'preload' | 'swap' | 'optional';
} = {
  fonts: [
    {
      family: 'Inter',
      src: '/fonts/inter-var.woff2',
      weight: '100 900',
      style: 'normal',
      display: 'swap',
      preload: true,
    },
    {
      family: 'Inter',
      src: '/fonts/inter-var-italic.woff2',
      weight: '100 900',
      style: 'italic',
      display: 'swap',
      preload: false,
    },
  ],

  fallbackFonts: [
    '-apple-system',
    'BlinkMacSystemFont',
    'Segoe UI',
    'Roboto',
    'Helvetica Neue',
    'Arial',
    'sans-serif',
  ],

  loadingStrategy: 'swap',
};

// =============================================================================
// Image Loading Configuration
// =============================================================================

export const imageConfig: ImageConfig = {
  // Responsive breakpoints
  breakpoints: [320, 640, 768, 1024, 1280, 1536],

  // Preferred formats in order of priority
  formats: ['avif', 'webp', 'jpeg'],

  // Default quality
  quality: 80,

  // Placeholder strategy
  placeholder: 'blur',

  // Lazy loading offset (Intersection Observer rootMargin)
  lazyOffset: '200px',

  // Fade-in animation duration in ms
  fadeInDuration: 300,
};

// =============================================================================
// Critical Resources
// =============================================================================

export const criticalResources = {
  // CSS that should be inlined for above-the-fold content
  inlineCSS: ['src/styles/critical.css'],

  // Scripts that block rendering
  blockingScripts: [] as string[],

  // Deferred scripts
  deferredScripts: ['analytics.js', 'chat-widget.js'],

  // Module scripts (type="module")
  moduleScripts: ['main.js'],
};

// =============================================================================
// Loading States Configuration
// =============================================================================

export const loadingStates = {
  // Minimum time to show loading state (prevents flash)
  minLoadingTime: 200,

  // Maximum time before showing timeout error
  maxLoadingTime: 30000,

  // Delay before showing loading indicator
  loadingDelay: 100,

  // Skeleton animation duration
  skeletonAnimationDuration: 1500,

  // Page transition duration
  pageTransitionDuration: 200,
};

// =============================================================================
// Connection-Aware Loading
// =============================================================================

export type ConnectionType = 'slow-2g' | '2g' | '3g' | '4g' | 'unknown';

export interface ConnectionConfig {
  preloadImages: boolean;
  autoplayVideos: boolean;
  loadHighResImages: boolean;
  prefetchRoutes: boolean;
  inlineStyles: boolean;
}

export const connectionSettings: Record<ConnectionType, ConnectionConfig> = {
  'slow-2g': {
    preloadImages: false,
    autoplayVideos: false,
    loadHighResImages: false,
    prefetchRoutes: false,
    inlineStyles: true,
  },
  '2g': {
    preloadImages: false,
    autoplayVideos: false,
    loadHighResImages: false,
    prefetchRoutes: false,
    inlineStyles: true,
  },
  '3g': {
    preloadImages: true,
    autoplayVideos: false,
    loadHighResImages: false,
    prefetchRoutes: true,
    inlineStyles: true,
  },
  '4g': {
    preloadImages: true,
    autoplayVideos: true,
    loadHighResImages: true,
    prefetchRoutes: true,
    inlineStyles: false,
  },
  unknown: {
    preloadImages: true,
    autoplayVideos: true,
    loadHighResImages: true,
    prefetchRoutes: true,
    inlineStyles: false,
  },
};

// =============================================================================
// Helper Functions
// =============================================================================

/**
 * Get current connection type
 */
export function getConnectionType(): ConnectionType {
  if (typeof navigator === 'undefined') return 'unknown';

  const connection = (
    navigator as Navigator & {
      connection?: {
        effectiveType?: string;
        saveData?: boolean;
      };
    }
  ).connection;

  if (!connection) return 'unknown';
  if (connection.saveData) return 'slow-2g';

  const effectiveType = connection.effectiveType;
  if (effectiveType === 'slow-2g') return 'slow-2g';
  if (effectiveType === '2g') return '2g';
  if (effectiveType === '3g') return '3g';
  if (effectiveType === '4g') return '4g';

  return 'unknown';
}

/**
 * Get loading configuration based on connection
 */
export function getConnectionConfig(): ConnectionConfig {
  const connectionType = getConnectionType();
  return connectionSettings[connectionType];
}

/**
 * Check if user prefers reduced data
 */
export function prefersReducedData(): boolean {
  if (typeof navigator === 'undefined') return false;

  const connection = (
    navigator as Navigator & {
      connection?: { saveData?: boolean };
    }
  ).connection;

  return connection?.saveData ?? false;
}

/**
 * Check if user prefers reduced motion
 */
export function prefersReducedMotion(): boolean {
  if (typeof window === 'undefined') return false;

  return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
}

/**
 * Generate srcset for responsive images
 */
export function generateSrcSet(
  basePath: string,
  breakpoints: number[] = imageConfig.breakpoints,
  format: string = 'webp'
): string {
  const ext = basePath.split('.').pop();
  const pathWithoutExt = basePath.replace(`.${ext}`, '');

  return breakpoints
    .map((width) => `${pathWithoutExt}-${width}w.${format} ${width}w`)
    .join(', ');
}

/**
 * Generate sizes attribute for responsive images
 */
export function generateSizes(
  config: { mobile?: string; tablet?: string; desktop?: string } = {}
): string {
  const { mobile = '100vw', tablet = '50vw', desktop = '33vw' } = config;

  return [
    `(max-width: 640px) ${mobile}`,
    `(max-width: 1024px) ${tablet}`,
    desktop,
  ].join(', ');
}

// =============================================================================
// Resource Hint Injection
// =============================================================================

/**
 * Inject resource hints into document head
 */
export function injectResourceHints(): void {
  if (typeof document === 'undefined') return;

  const head = document.head;

  // Preconnect
  resourceHints.preconnect.forEach((hint) => {
    const link = document.createElement('link');
    link.rel = 'preconnect';
    link.href = hint.href;
    if (hint.crossOrigin) link.crossOrigin = hint.crossOrigin;
    head.appendChild(link);
  });

  // DNS Prefetch
  resourceHints.dnsPrefetch.forEach((hint) => {
    const link = document.createElement('link');
    link.rel = 'dns-prefetch';
    link.href = hint.href;
    head.appendChild(link);
  });

  // Preload
  resourceHints.preload.forEach((hint) => {
    const link = document.createElement('link');
    link.rel = 'preload';
    link.href = hint.href;
    if (hint.as) link.as = hint.as;
    if (hint.type) link.type = hint.type;
    if (hint.crossOrigin) link.crossOrigin = hint.crossOrigin;
    head.appendChild(link);
  });
}

/**
 * Prefetch a resource
 */
export function prefetchResource(href: string, as?: string): void {
  if (typeof document === 'undefined') return;

  // Check if already prefetched
  const existing = document.querySelector(
    `link[rel="prefetch"][href="${href}"]`
  );
  if (existing) return;

  const link = document.createElement('link');
  link.rel = 'prefetch';
  link.href = href;
  if (as) link.as = as;

  document.head.appendChild(link);
}

/**
 * Preload a resource
 */
export function preloadResource(href: string, as: string, type?: string): void {
  if (typeof document === 'undefined') return;

  // Check if already preloaded
  const existing = document.querySelector(
    `link[rel="preload"][href="${href}"]`
  );
  if (existing) return;

  const link = document.createElement('link');
  link.rel = 'preload';
  link.href = href;
  link.as = as;
  if (type) link.type = type;

  document.head.appendChild(link);
}
