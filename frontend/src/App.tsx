import { useEffect } from 'react';
import { BrowserRouter, useLocation } from 'react-router-dom';
import { AppRoutes } from './routes';
import { StripeProvider } from './providers/StripeProvider';
import { QueryProvider } from './providers/QueryProvider';
import { trackEvent, AnalyticsEvents } from './services/analytics';
import { sanitizeSearch } from './utils/urlSanitizer';

export function PageViewTracker() {
  const location = useLocation();
  // Only allowlisted query params are sent: tokens, codes and emails in
  // reset/verify/invitation links must never reach analytics.
  const search = sanitizeSearch(location.search);

  useEffect(() => {
    trackEvent(AnalyticsEvents.PAGE_VIEWED, {
      path: location.pathname,
      search,
    });
  }, [location.pathname, search]);

  return null;
}

function App() {
  return (
    <BrowserRouter>
      <PageViewTracker />
      <QueryProvider>
        <StripeProvider>
          <AppRoutes />
        </StripeProvider>
      </QueryProvider>
    </BrowserRouter>
  );
}

export default App;
