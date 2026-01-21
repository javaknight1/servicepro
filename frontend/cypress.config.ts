import { defineConfig } from 'cypress';

export default defineConfig({
  // ==========================================================================
  // Project Configuration
  // ==========================================================================
  projectId: 'servicepro-e2e',

  // ==========================================================================
  // Reporter Configuration
  // ==========================================================================
  reporter: 'mochawesome',
  reporterOptions: {
    reportDir: 'cypress/reports/mochawesome',
    overwrite: false,
    html: true,
    json: true,
    timestamp: 'mmddyyyy_HHMMss',
    charts: true,
  },

  // ==========================================================================
  // E2E Testing Configuration
  // ==========================================================================
  e2e: {
    baseUrl: 'http://localhost:3000',
    specPattern: 'cypress/e2e/**/*.cy.{js,jsx,ts,tsx}',
    supportFile: 'cypress/support/e2e.ts',

    // Test isolation
    testIsolation: true,

    // Timeouts
    defaultCommandTimeout: 10000,
    requestTimeout: 10000,
    responseTimeout: 30000,
    pageLoadTimeout: 30000,

    // Retries
    retries: {
      runMode: 2,
      openMode: 0,
    },

    // Viewport
    viewportWidth: 1280,
    viewportHeight: 720,

    // Videos and Screenshots
    video: true,
    videoCompression: 32,
    screenshotOnRunFailure: true,
    screenshotsFolder: 'cypress/screenshots',
    videosFolder: 'cypress/videos',

    // Experimental features
    experimentalRunAllSpecs: true,

    // Setup node events
    setupNodeEvents(on, config) {
      // Load environment-specific config
      const environment = config.env.environment || 'development';

      // Environment-specific base URLs
      const environmentConfigs: Record<
        string,
        { baseUrl: string; apiUrl: string }
      > = {
        development: {
          baseUrl: 'http://localhost:3000',
          apiUrl: 'http://localhost:8080',
        },
        staging: {
          baseUrl: 'https://staging.servicepro.app',
          apiUrl: 'https://staging-api.servicepro.app',
        },
        production: {
          baseUrl: 'https://app.servicepro.app',
          apiUrl: 'https://api.servicepro.app',
        },
      };

      const envConfig =
        environmentConfigs[environment] || environmentConfigs.development;
      config.baseUrl = envConfig.baseUrl;
      config.env.apiUrl = envConfig.apiUrl;

      // Task for database seeding
      on('task', {
        log(message) {
          console.log(message);
          return null;
        },
        table(message) {
          console.table(message);
          return null;
        },
        clearTestData() {
          // Hook for clearing test data
          console.log('Clearing test data...');
          return null;
        },
        seedTestData(data) {
          // Hook for seeding test data
          console.log('Seeding test data:', data);
          return null;
        },
      });

      // Code coverage (if enabled)
      // require('@cypress/code-coverage/task')(on, config);

      return config;
    },
  },

  // ==========================================================================
  // Component Testing Configuration (Optional)
  // ==========================================================================
  component: {
    devServer: {
      framework: 'react',
      bundler: 'vite',
    },
    specPattern: 'src/**/*.cy.{js,jsx,ts,tsx}',
    supportFile: 'cypress/support/component.ts',
  },

  // ==========================================================================
  // Global Configuration
  // ==========================================================================

  // Environment variables
  env: {
    // API endpoints
    apiUrl: 'http://localhost:8080',

    // Test user credentials
    testUserEmail: 'test@servicepro.test',
    testUserPassword: 'TestPassword123!',
    adminEmail: 'admin@servicepro.test',
    adminPassword: 'AdminPassword123!',

    // Feature flags
    enableVisualTesting: false,
    enableNetworkStubbing: true,

    // Timeouts
    loginTimeout: 10000,

    // Coverage
    codeCoverage: false,
  },

  // ==========================================================================
  // Browser Configuration
  // ==========================================================================

  // Chrome-specific settings
  chromeWebSecurity: false,

  // Disable waiting for actionability
  waitForAnimations: true,
  animationDistanceThreshold: 5,

  // Number of tests to keep in memory
  numTestsKeptInMemory: 50,

  // Watch for file changes
  watchForFileChanges: true,
});
