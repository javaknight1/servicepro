import { useCallback } from 'react';
import { trackEvent, AnalyticsEvents } from '@services/analytics';

/**
 * Hook for tracking analytics events in components.
 */
export function useAnalytics() {
  const track = useCallback(
    (event: string, properties?: Record<string, unknown>) => {
      trackEvent(event, properties);
    },
    []
  );

  return { trackEvent: track, AnalyticsEvents };
}
