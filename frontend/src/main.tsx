import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App.tsx';
import './index.css';
import { initErrorTracking } from './services/errorTracking';
import { ErrorBoundary } from './components/error/ErrorBoundary';

initErrorTracking({
  dsn: import.meta.env.VITE_SENTRY_DSN || '',
  environment: import.meta.env.VITE_ENV || 'development',
  release: import.meta.env.VITE_APP_VERSION || '1.0.0',
});

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <ErrorBoundary>
      <App />
    </ErrorBoundary>
  </React.StrictMode>
);
