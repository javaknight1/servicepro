/**
 * =============================================================================
 * Bundle Optimization Configuration
 * =============================================================================
 * Vite optimization settings for production builds
 */

import type { BuildOptions, PluginOption } from 'vite';

// =============================================================================
// Chunk Splitting Configuration
// =============================================================================

export const manualChunks = {
  // Core React
  // react-router, @remix-run/router and scheduler sit with react-dom /
  // react-router-dom: left in `vendor` they form a vendor <-> vendor-react
  // import cycle, which Rolldown's CommonJS wrappers can't evaluate
  // ("r is not a function" at startup).
  'vendor-react': [
    'react',
    'react-dom',
    'react-router-dom',
    'react-router',
    '@remix-run/router',
    'scheduler',
  ],

  // UI Framework
  'vendor-ui': ['@headlessui/react', 'lucide-react', 'clsx'],

  // Forms & Validation
  'vendor-forms': ['react-hook-form', '@hookform/resolvers', 'zod'],

  // Data Management
  'vendor-data': [
    '@tanstack/react-query',
    '@tanstack/react-query-persist-client',
    '@tanstack/query-sync-storage-persister',
    'zustand',
    'axios',
  ],

  // Charts & Visualization
  'vendor-charts': ['chart.js', 'react-chartjs-2'],

  // Date utilities
  'vendor-date': ['date-fns', 'react-datepicker'],

  // Stripe (lazy loaded)
  'vendor-stripe': ['@stripe/react-stripe-js', '@stripe/stripe-js'],

  // Error tracking + product analytics (loaded on every page)
  'vendor-observability': ['@sentry', '@sentry-internal', 'posthog-js'],

  // Calendar — separate chunk to avoid circular dependency with vendor-react
  'vendor-calendar': [
    'react-big-calendar',
    'react-overlays',
    'dom-helpers',
    'date-arithmetic',
  ],
};

/** True when `id` lives in `node_modules/<pkg>` (exact package boundary, so
 * 'react' does not match 'react-big-calendar'). */
function isPackage(id: string, pkg: string): boolean {
  const needle = `node_modules/${pkg}`;
  const idx = id.indexOf(needle);
  if (idx === -1) return false;
  const after = idx + needle.length;
  return after >= id.length || id[after] === '/';
}

/**
 * Rolldown code-splitting groups (Vite 8), replacing Rollup's function-form
 * `manualChunks`. Ported as one group per chunk with explicit priorities —
 * named vendor chunks first, then catch-all `vendor`, then the app feature
 * chunks — and `includeDependenciesRecursively: false` (set below). With
 * Rolldown's default (true) a group also captures its dependencies, so e.g.
 * page-dashboard swallowed the shared app core (api, stores, layout) and the
 * named vendor chunks drained `vendor`. Chunk names are unchanged and vendor
 * contents match the Rollup split, so scripts/analyze-bundle.cjs budgets hold.
 */
const appChunks: Array<[name: string, test: (id: string) => boolean]> = [
  ['page-dashboard', (id) => id.includes('/pages/Dashboard')],
  ['page-settings', (id) => id.includes('/pages/Settings')],
  [
    'page-auth',
    (id) => id.includes('/pages/Login') || id.includes('/pages/Register'),
  ],
  ['feature-calendar', (id) => id.includes('/components/calendar')],
  ['feature-scheduling', (id) => id.includes('/components/scheduling')],
  ['feature-quotes', (id) => id.includes('/components/quotes')],
];

const codeSplittingGroups = [
  ...Object.entries(manualChunks).map(([name, packages]) => ({
    name,
    test: (id: string) => packages.some((pkg) => isPackage(id, pkg)),
    priority: 30,
  })),
  {
    name: 'vendor-workbox',
    test: (id: string) => id.includes('node_modules') && id.includes('workbox'),
    priority: 25,
  },
  // Everything else from node_modules, plus Rolldown's injected transform
  // helpers (virtual `\0@oxc-project+runtime/...` ids, e.g. private-field
  // lowering for safari14). Left unassigned they land in the entry chunk and
  // vendor chunks would import from the app entry (a chunk cycle).
  {
    name: 'vendor',
    test: (id: string) =>
      id.includes('node_modules') || id.startsWith('\0@oxc-project+runtime'),
    priority: 20,
  },
  ...appChunks.map(([name, test]) => ({
    name,
    test: (id: string) => !id.includes('node_modules') && test(id),
    priority: 10,
  })),
];

// =============================================================================
// Build Options
// =============================================================================

export const buildOptions: BuildOptions = {
  // Output directory
  outDir: 'dist',

  // Asset directory within outDir
  assetsDir: 'assets',

  // Hidden source maps: emitted for Sentry upload but not referenced from the
  // bundles. public/.assetsignore keeps *.map out of the Workers deploy.
  // Set GENERATE_SOURCEMAP=false to skip them (faster local builds).
  sourcemap: process.env.GENERATE_SOURCEMAP === 'false' ? false : 'hidden',

  // Minification
  minify: 'terser',

  // Terser options for better minification
  terserOptions: {
    compress: {
      // Remove console.log in production
      drop_console: process.env.NODE_ENV === 'production',
      drop_debugger: true,
      // Optimize booleans
      booleans_as_integers: false,
      // Remove unused code
      dead_code: true,
      // Collapse variables
      collapse_vars: true,
      // Reduce variable names
      reduce_vars: true,
      // Inline functions called once
      inline: 2,
      // Remove unreachable code
      unused: true,
    },
    mangle: {
      // Mangle property names (careful with this)
      properties: false,
    },
    format: {
      // Remove comments
      comments: false,
    },
  },

  // Rolldown options (Vite 8; `rollupOptions` is a deprecated alias)
  rolldownOptions: {
    output: {
      // Manual chunk splitting — see codeSplittingGroups
      codeSplitting: {
        groups: codeSplittingGroups,
        includeDependenciesRecursively: false,
      },

      // Asset file naming
      assetFileNames: (assetInfo) => {
        const info = assetInfo.names[0] || '';

        // Fonts
        if (/\.(woff2?|ttf|eot|otf)$/.test(info)) {
          return 'assets/fonts/[name]-[hash][extname]';
        }

        // Images
        if (/\.(png|jpe?g|gif|svg|webp|avif|ico)$/.test(info)) {
          return 'assets/images/[name]-[hash][extname]';
        }

        // CSS
        if (/\.css$/.test(info)) {
          return 'assets/css/[name]-[hash][extname]';
        }

        // Default
        return 'assets/[name]-[hash][extname]';
      },

      // Chunk file naming
      chunkFileNames: 'assets/js/[name]-[hash].js',

      // Entry file naming
      entryFileNames: 'assets/js/[name]-[hash].js',
    },

    // Tree shaking
    treeshake: {
      // Respect package.json sideEffects field
      moduleSideEffects: 'no-external',
      // Remove unused exports
      propertyReadSideEffects: false,
      // (Rollup's tryCatchDeoptimization has no Rolldown equivalent)
    },
  },

  // Chunk size warning limit (in kB)
  chunkSizeWarningLimit: 500,

  // CSS code splitting
  cssCodeSplit: true,

  // Target browsers
  target: ['es2020', 'edge88', 'firefox78', 'chrome87', 'safari14'],

  // Report compressed size
  reportCompressedSize: true,
};

// =============================================================================
// Dependency Optimization
// =============================================================================

export const optimizeDeps = {
  // Pre-bundle these dependencies
  include: [
    'react',
    'react-dom',
    'react-router',
    '@headlessui/react',
    'lucide-react',
    'clsx',
    'zustand',
    'axios',
    '@tanstack/react-query',
    'react-hook-form',
    'zod',
    'date-fns',
    'react-easy-crop',
    'tslib',
  ],

  // Don't pre-bundle these (for development)
  exclude: ['@stripe/stripe-js'],

  // Force optimization even in dev
  force: false,

  // Entry points for dependency scanning
  entries: ['./src/main.tsx', './src/App.tsx'],

  // Rolldown options for pre-bundling (Vite 8 replaced esbuildOptions)
  rolldownOptions: {
    // Target for dependency pre-bundling
    transform: { target: 'es2020' },
  },
};

// =============================================================================
// CSS Optimization
// =============================================================================

export const cssOptions = {
  // CSS modules configuration
  modules: {
    // Scoped class name format
    generateScopedName:
      process.env.NODE_ENV === 'production'
        ? '[hash:base64:8]'
        : '[name]__[local]__[hash:base64:5]',
  },

  // Preprocessor options
  preprocessorOptions: {
    scss: {
      // Add global SCSS variables/mixins
      additionalData: `@import "@/styles/variables.scss";`,
    },
  },

  // Dev source maps
  devSourcemap: true,
};

// =============================================================================
// Compression Plugin Configuration
// =============================================================================

export const compressionConfig = {
  // Gzip compression
  gzip: {
    algorithm: 'gzip' as const,
    ext: '.gz',
    threshold: 10240, // Only compress files > 10KB
    minRatio: 0.8,
    deleteOriginFile: false,
  },

  // Brotli compression
  brotli: {
    algorithm: 'brotliCompress' as const,
    ext: '.br',
    threshold: 10240,
    minRatio: 0.8,
    deleteOriginFile: false,
  },
};

// =============================================================================
// Bundle Analyzer Configuration
// =============================================================================

export const analyzerConfig = {
  // Analyzer mode: 'server' | 'static' | 'json' | 'disabled'
  analyzerMode: 'static' as const,

  // Report filename
  reportFilename: 'bundle-report.html',

  // Open report automatically
  openAnalyzer: false,

  // Generate stats file
  generateStatsFile: true,
  statsFilename: 'bundle-stats.json',

  // Stats options
  statsOptions: {
    source: false,
  },

  // Exclude patterns
  excludeAssets: /\.map$/,
};

// =============================================================================
// Performance Budgets
// =============================================================================

export const performanceBudgets = {
  // Maximum bundle sizes (in KB, gzipped)
  maxBundleSizes: {
    // React 19's react-dom client is ~22 KB gzipped bigger than 18's (44 -> 66 KB
    // measured 2026-10-18, T164); no server code is bundled. Under Vite 8 the
    // chunk also holds react-router (+ scheduler): Rolldown can't run the
    // vendor <-> vendor-react cycle Rollup tolerated, so they moved in (~80 KB).
    'vendor-react': 85,
    'vendor-ui': 25,
    'vendor-forms': 30,
    'vendor-data': 25,
    'vendor-charts': 70,
    'vendor-date': 10,
    'vendor-stripe': 10,
    'vendor-calendar': 50,
    'vendor-table': 10,
    vendor: 60,
    'page-dashboard': 10,
    'page-settings': 20,
    'page-auth': 20,
  },

  // Maximum total bundle size (gzipped)
  maxTotalSize: 500,

  // Maximum initial load size (gzipped)
  maxInitialSize: 300,

  // Maximum asset sizes
  maxAssetSizes: {
    js: 500,
    css: 100,
    image: 200,
    font: 100,
  },
};

// =============================================================================
// Image Optimization Configuration
// =============================================================================

export const imageOptimizationConfig = {
  // Quality settings
  quality: {
    jpeg: 80,
    webp: 80,
    avif: 65,
    png: 80,
  },

  // Responsive image breakpoints
  breakpoints: [320, 640, 768, 1024, 1280, 1536],

  // Output formats
  formats: ['webp', 'avif'] as const,

  // Source directory
  inputDir: 'src/assets/images',

  // Output directory
  outputDir: 'public/images',

  // Placeholder generation
  placeholder: {
    enabled: true,
    size: 20, // 20px width for blur placeholder
    quality: 50,
  },
};

// =============================================================================
// Export Helper Functions
// =============================================================================

/**
 * Get dynamic import path for code splitting
 */
export function getDynamicImport(
  componentPath: string
): () => Promise<unknown> {
  return () => import(/* @vite-ignore */ componentPath);
}

/**
 * Check if bundle size exceeds budget
 */
export function checkBundleBudget(
  chunkName: string,
  sizeInBytes: number
): { exceeded: boolean; budget: number; actual: number } {
  const budget =
    performanceBudgets.maxBundleSizes[
      chunkName as keyof typeof performanceBudgets.maxBundleSizes
    ];
  const sizeInKB = sizeInBytes / 1024;

  return {
    exceeded: budget ? sizeInKB > budget : false,
    budget: budget || 0,
    actual: Math.round(sizeInKB),
  };
}
