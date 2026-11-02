import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App.tsx';
// Inter is self-hosted (bundled woff2), not hot-linked from Google Fonts (T262).
import '@fontsource-variable/inter';
import './index.css';
import { initErrorTracking } from './services/errorTracking';
import { initAnalytics } from './services/analytics';
import { ErrorBoundary } from './components/error/ErrorBoundary';

initErrorTracking({
  dsn: import.meta.env.VITE_SENTRY_DSN || '',
  environment: import.meta.env.VITE_ENV || 'development',
  // Same id as the source maps release-deploy.yml uploads (github.sha).
  release: import.meta.env.VITE_GIT_SHA || 'unknown',
});

initAnalytics({
  apiKey: import.meta.env.VITE_POSTHOG_API_KEY || '',
  apiHost: import.meta.env.VITE_POSTHOG_HOST || 'https://us.i.posthog.com',
});

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <ErrorBoundary>
      <App />
    </ErrorBoundary>
  </React.StrictMode>
);
