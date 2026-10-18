/// <reference types="vitest/config" />
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';
import { visualizer } from 'rollup-plugin-visualizer';
import viteCompression from 'vite-plugin-compression';
import path from 'path';

// Import optimization configuration
import {
  buildOptions,
  optimizeDeps,
  cssOptions,
  compressionConfig,
  analyzerConfig,
} from './config/optimization';

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => ({
  plugins: [
    react(),
    tailwindcss(),

    // Bundle analyzer (production only)
    mode === 'production' &&
      process.env.ANALYZE === 'true' &&
      visualizer({
        filename: analyzerConfig.reportFilename,
        open: analyzerConfig.openAnalyzer,
        gzipSize: true,
        brotliSize: true,
      }),

    // Gzip compression (production only)
    mode === 'production' &&
      viteCompression({
        algorithm: 'gzip',
        ext: compressionConfig.gzip.ext,
        threshold: compressionConfig.gzip.threshold,
        deleteOriginFile: compressionConfig.gzip.deleteOriginFile,
      }),

    // Brotli compression (production only)
    mode === 'production' &&
      viteCompression({
        algorithm: 'brotliCompress',
        ext: compressionConfig.brotli.ext,
        threshold: compressionConfig.brotli.threshold,
        deleteOriginFile: compressionConfig.brotli.deleteOriginFile,
      }),
  ].filter(Boolean),
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
      '@components': path.resolve(__dirname, './src/components'),
      '@pages': path.resolve(__dirname, './src/pages'),
      '@hooks': path.resolve(__dirname, './src/hooks'),
      '@services': path.resolve(__dirname, './src/services'),
      '@utils': path.resolve(__dirname, './src/utils'),
      '@store': path.resolve(__dirname, './src/store'),
      '@app-types': path.resolve(__dirname, './src/types'),
      '@config': path.resolve(__dirname, './src/config'),
      '@providers': path.resolve(__dirname, './src/providers'),
      // Mock @sentry/tracing (not installed as a dependency)
      '@sentry/tracing': path.resolve(
        __dirname,
        './src/test/__mocks__/sentry-tracing.ts'
      ),
    },
  },
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: process.env.VITE_PROXY_TARGET || 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    include: [
      'src/**/*.{test,spec}.{ts,tsx}',
      'tests/**/*.{test,spec}.{ts,tsx}',
      'worker/**/*.{test,spec}.ts',
    ],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'text-summary', 'json', 'html', 'lcov'],
      exclude: [
        'node_modules/',
        'src/test/',
        '**/*.d.ts',
        '**/*.config.*',
        '**/index.ts',
        '**/__stories__/**',
        '**/types/**',
      ],
      thresholds: {
        statements: 69.5,
        branches: 64.5,
        functions: 70,
        lines: 70,
      },
    },
    // Test timeout
    testTimeout: 20000,
    // No retries: flaky tests must surface rather than be masked
    pool: 'forks',
  },

  // Build configuration from optimization config
  build: buildOptions,

  // Dependency optimization
  optimizeDeps,

  // CSS configuration
  css: cssOptions,

  // Vite 8 gives a default import from a CommonJS module Node semantics
  // (`default` = module.exports) because this package is "type": "module".
  // CJS-only deps such as react-big-calendar/lib/addons/dragAndDrop then break
  // at runtime in the production build ("L.default is not a function"). Keep
  // the pre-Vite 8 interop until those imports are audited.
  legacy: {
    inconsistentCjsInterop: true,
  },
}));
