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
  'vendor-react': ['react', 'react-dom', 'react-router-dom'],

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
};

// =============================================================================
// Build Options
// =============================================================================

export const buildOptions: BuildOptions = {
  // Output directory
  outDir: 'dist',

  // Asset directory within outDir
  assetsDir: 'assets',

  // Enable source maps for production (can be disabled for smaller builds)
  sourcemap: process.env.GENERATE_SOURCEMAP === 'true',

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

  // Rollup options
  rollupOptions: {
    output: {
      // Manual chunk splitting
      manualChunks: (id: string) => {
        // Node modules chunking
        if (id.includes('node_modules')) {
          // Check against our defined chunks
          for (const [chunkName, packages] of Object.entries(manualChunks)) {
            if (packages.some((pkg) => id.includes(`node_modules/${pkg}`))) {
              return chunkName;
            }
          }

          // Workbox chunks
          if (id.includes('workbox')) {
            return 'vendor-workbox';
          }

          // Everything else goes to vendor
          return 'vendor';
        }

        // Application code splitting by feature
        if (id.includes('/pages/Dashboard')) return 'page-dashboard';
        if (id.includes('/pages/Settings')) return 'page-settings';
        if (id.includes('/pages/Login') || id.includes('/pages/Register')) {
          return 'page-auth';
        }
        if (id.includes('/components/calendar')) return 'feature-calendar';
        if (id.includes('/components/scheduling')) return 'feature-scheduling';
        if (id.includes('/components/quotes')) return 'feature-quotes';
        if (id.includes('/components/templates')) return 'feature-templates';

        return undefined;
      },

      // Asset file naming
      assetFileNames: (assetInfo) => {
        const info = assetInfo.name || '';

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
      // More aggressive tree shaking
      moduleSideEffects: (id, external) => {
        // Keep side effects for CSS
        if (id.endsWith('.css')) return true;

        // Check package.json sideEffects
        return false;
      },
      // Remove unused exports
      propertyReadSideEffects: false,
      // Remove unused assignments
      tryCatchDeoptimization: false,
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
    'react-router-dom',
    '@headlessui/react',
    'lucide-react',
    'clsx',
    'zustand',
    'axios',
    '@tanstack/react-query',
    'react-hook-form',
    'zod',
    'date-fns',
  ],

  // Don't pre-bundle these (for development)
  exclude: ['@stripe/stripe-js'],

  // Force optimization even in dev
  force: false,

  // Entry points for dependency scanning
  entries: ['./src/main.tsx', './src/App.tsx'],

  // ESBuild options for pre-bundling
  esbuildOptions: {
    // Target for dependency pre-bundling
    target: 'es2020',

    // Supported extensions
    supported: {
      'top-level-await': true,
    },
  },
};

// =============================================================================
// CSS Optimization
// =============================================================================

export const cssOptions = {
  // PostCSS config location
  postcss: './postcss.config.js',

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
  // Maximum bundle sizes (in KB)
  maxBundleSizes: {
    'vendor-react': 150,
    'vendor-ui': 100,
    'vendor-forms': 80,
    'vendor-data': 120,
    'vendor-charts': 200,
    'vendor-date': 80,
    'vendor-stripe': 100,
    vendor: 200,
    'page-dashboard': 100,
    'page-settings': 50,
    'page-auth': 30,
  },

  // Maximum total bundle size
  maxTotalSize: 1500, // 1.5MB

  // Maximum initial load size
  maxInitialSize: 400, // 400KB

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
