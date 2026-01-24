/// <reference types="vitest" />
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { VitePWA } from 'vite-plugin-pwa';
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
    VitePWA({
      strategies: 'injectManifest',
      srcDir: 'src',
      filename: 'service-worker.ts',
      registerType: 'prompt',
      injectRegister: false,
      manifest: {
        name: 'ServicePro',
        short_name: 'ServicePro',
        description: 'Service management application',
        theme_color: '#1a1a2e',
        background_color: '#ffffff',
        display: 'standalone',
        icons: [
          {
            src: '/icon-192.png',
            sizes: '192x192',
            type: 'image/png',
          },
          {
            src: '/icon-512.png',
            sizes: '512x512',
            type: 'image/png',
          },
        ],
      },
      workbox: {
        globPatterns: ['**/*.{js,css,html,ico,png,svg,woff,woff2}'],
        runtimeCaching: [
          {
            urlPattern: /^https:\/\/fonts\.googleapis\.com\/.*/i,
            handler: 'CacheFirst',
            options: {
              cacheName: 'google-fonts-cache',
              expiration: {
                maxEntries: 10,
                maxAgeSeconds: 60 * 60 * 24 * 365, // 1 year
              },
              cacheableResponse: {
                statuses: [0, 200],
              },
            },
          },
        ],
      },
      devOptions: {
        enabled: true,
        type: 'module',
      },
    }),

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
      '@theme': path.resolve(__dirname, './src/theme'),
      '@types': path.resolve(__dirname, './src/types'),
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
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
    deps: {
      inline: ['@sentry/tracing'],
    },
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
        statements: 70,
        branches: 65,
        functions: 70,
        lines: 70,
      },
    },
    // Test timeout
    testTimeout: 10000,
    // Retry failed tests
    retry: 1,
    // Pool configuration for parallel testing
    pool: 'forks',
    poolOptions: {
      forks: {
        singleFork: false,
      },
    },
  },

  // Build configuration from optimization config
  build: buildOptions,

  // Dependency optimization
  optimizeDeps,

  // CSS configuration
  css: cssOptions,
}));
