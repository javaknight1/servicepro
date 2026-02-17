import { useEffect } from 'react';
import { BrowserRouter, useLocation } from 'react-router-dom';
import { AppRoutes } from './routes';
import { StripeProvider } from './providers/StripeProvider';
import { QueryProvider } from './providers/QueryProvider';
import { trackEvent, AnalyticsEvents } from './services/analytics';

function PageViewTracker() {
  const location = useLocation();

  useEffect(() => {
    trackEvent(AnalyticsEvents.PAGE_VIEWED, {
      path: location.pathname,
      search: location.search,
    });
  }, [location.pathname, location.search]);

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
