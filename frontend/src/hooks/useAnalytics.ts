/**
 * Analytics Hooks
 *
 * Convenience hooks for common analytics patterns
 */

import { useCallback, useRef, useEffect } from 'react';
import {
  useAnalytics,
  usePageTracking,
  useComponentTracking,
  useInteractionTracking,
  useApiTracking,
  useSearchTracking,
  useBusinessTracking,
  EventProperties,
} from '../contexts/AnalyticsContext';

// Re-export context hooks
export {
  useAnalytics,
  usePageTracking,
  useComponentTracking,
  useInteractionTracking,
  useApiTracking,
  useSearchTracking,
  useBusinessTracking,
};

/**
 * Hook for tracking time spent on a page or component
 */
export function useTimeTracking(name: string) {
  const { track } = useAnalytics();
  const startTime = useRef<number>(Date.now());
  const hasTracked = useRef<boolean>(false);

  useEffect(() => {
    startTime.current = Date.now();
    hasTracked.current = false;

    return () => {
      if (!hasTracked.current) {
        const duration = Date.now() - startTime.current;
        track('feature_used', 'feature', {
          feature_name: name,
          action: 'time_spent',
          duration_ms: duration,
          duration_seconds: Math.round(duration / 1000),
        });
        hasTracked.current = true;
      }
    };
  }, [name, track]);

  const trackTimeNow = useCallback(() => {
    if (!hasTracked.current) {
      const duration = Date.now() - startTime.current;
      track('feature_used', 'feature', {
        feature_name: name,
        action: 'time_spent',
        duration_ms: duration,
        duration_seconds: Math.round(duration / 1000),
      });
      hasTracked.current = true;
    }
  }, [name, track]);

  return { trackTimeNow };
}

/**
 * Hook for tracking scroll depth
 */
export function useScrollTracking(thresholds: number[] = [25, 50, 75, 100]) {
  const { track } = useAnalytics();
  const trackedThresholds = useRef<Set<number>>(new Set());

  useEffect(() => {
    const handleScroll = () => {
      const scrollHeight =
        document.documentElement.scrollHeight - window.innerHeight;
      const scrollPercent = Math.round((window.scrollY / scrollHeight) * 100);

      thresholds.forEach((threshold) => {
        if (
          scrollPercent >= threshold &&
          !trackedThresholds.current.has(threshold)
        ) {
          trackedThresholds.current.add(threshold);
          track('page_scroll', 'page', {
            scroll_depth: threshold,
          });
        }
      });
    };

    window.addEventListener('scroll', handleScroll, { passive: true });
    return () => window.removeEventListener('scroll', handleScroll);
  }, [thresholds, track]);

  // Reset tracked thresholds when component remounts
  useEffect(() => {
    trackedThresholds.current.clear();
  }, []);
}

/**
 * Hook for tracking form interactions
 */
export function useFormTracking(formId: string, formName?: string) {
  const { track, trackFormSubmit } = useAnalytics();
  const startTime = useRef<number>(Date.now());

  const trackFieldFocus = useCallback(
    (fieldName: string) => {
      track('feature_used', 'feature', {
        feature_name: 'form_field',
        action: 'focus',
        form_id: formId,
        form_name: formName,
        field_name: fieldName,
      });
    },
    [formId, formName, track]
  );

  const trackFieldBlur = useCallback(
    (fieldName: string, hasValue: boolean) => {
      track('feature_used', 'feature', {
        feature_name: 'form_field',
        action: 'blur',
        form_id: formId,
        form_name: formName,
        field_name: fieldName,
        has_value: hasValue,
      });
    },
    [formId, formName, track]
  );

  const trackFieldError = useCallback(
    (fieldName: string, errorMessage: string) => {
      track('feature_used', 'feature', {
        feature_name: 'form_field',
        action: 'validation_error',
        form_id: formId,
        form_name: formName,
        field_name: fieldName,
        error_message: errorMessage,
      });
    },
    [formId, formName, track]
  );

  const trackSubmit = useCallback(
    (success: boolean, errorMessage?: string) => {
      const duration = Date.now() - startTime.current;
      trackFormSubmit(formId, formName, success);

      track('feature_used', 'feature', {
        feature_name: 'form_submit',
        action: success ? 'success' : 'error',
        form_id: formId,
        form_name: formName,
        duration_ms: duration,
        error_message: errorMessage,
      });

      // Reset start time for next submission attempt
      startTime.current = Date.now();
    },
    [formId, formName, track, trackFormSubmit]
  );

  return {
    trackFieldFocus,
    trackFieldBlur,
    trackFieldError,
    trackSubmit,
  };
}

/**
 * Hook for tracking element visibility
 */
export function useVisibilityTracking(
  elementRef: React.RefObject<HTMLElement>,
  name: string,
  options?: IntersectionObserverInit
) {
  const { track } = useAnalytics();
  const hasTracked = useRef<boolean>(false);

  useEffect(() => {
    const element = elementRef.current;
    if (!element) return;

    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting && !hasTracked.current) {
            hasTracked.current = true;
            track('feature_used', 'feature', {
              feature_name: name,
              action: 'viewed',
              visibility_ratio: entry.intersectionRatio,
            });
          }
        });
      },
      {
        threshold: 0.5,
        ...options,
      }
    );

    observer.observe(element);

    return () => {
      observer.disconnect();
    };
  }, [elementRef, name, options, track]);

  // Reset tracking on name change
  useEffect(() => {
    hasTracked.current = false;
  }, [name]);
}

/**
 * Hook for tracking clicks with debouncing
 */
export function useClickTracking(debounceMs: number = 300) {
  const { trackClick } = useAnalytics();
  const lastClickTime = useRef<number>(0);

  const handleClick = useCallback(
    (buttonId: string, buttonText?: string, properties?: EventProperties) => {
      const now = Date.now();
      if (now - lastClickTime.current >= debounceMs) {
        lastClickTime.current = now;
        trackClick(buttonId, buttonText, properties);
      }
    },
    [debounceMs, trackClick]
  );

  return { trackClick: handleClick };
}

/**
 * Hook for tracking feature toggles
 */
export function useFeatureToggleTracking() {
  const { track } = useAnalytics();

  const trackToggle = useCallback(
    (featureName: string, enabled: boolean, properties?: EventProperties) => {
      track('feature_used', 'feature', {
        feature_name: featureName,
        action: enabled ? 'enabled' : 'disabled',
        ...properties,
      });
    },
    [track]
  );

  return { trackToggle };
}

/**
 * Hook for tracking navigation
 */
export function useNavigationTracking() {
  const { track } = useAnalytics();

  const trackNavigation = useCallback(
    (
      from: string,
      to: string,
      method: 'link' | 'button' | 'programmatic' = 'link'
    ) => {
      track('feature_used', 'feature', {
        feature_name: 'navigation',
        action: 'navigate',
        from_path: from,
        to_path: to,
        navigation_method: method,
      });
    },
    [track]
  );

  const trackBackNavigation = useCallback(
    (from: string) => {
      track('feature_used', 'feature', {
        feature_name: 'navigation',
        action: 'back',
        from_path: from,
      });
    },
    [track]
  );

  return {
    trackNavigation,
    trackBackNavigation,
  };
}

/**
 * Hook for tracking filters and sorting
 */
export function useFilterTracking(context: string) {
  const { track } = useAnalytics();

  const trackFilter = useCallback(
    (filterName: string, filterValue: string | number | boolean | null) => {
      track('filter_applied', 'feature', {
        context,
        filter_name: filterName,
        filter_value: String(filterValue),
      });
    },
    [context, track]
  );

  const trackSort = useCallback(
    (sortField: string, sortDirection: 'asc' | 'desc') => {
      track('feature_used', 'feature', {
        feature_name: 'sort',
        action: 'applied',
        context,
        sort_field: sortField,
        sort_direction: sortDirection,
      });
    },
    [context, track]
  );

  const trackClearFilters = useCallback(() => {
    track('feature_used', 'feature', {
      feature_name: 'filter',
      action: 'cleared',
      context,
    });
  }, [context, track]);

  return {
    trackFilter,
    trackSort,
    trackClearFilters,
  };
}

/**
 * Hook for tracking export actions
 */
export function useExportTracking() {
  const { track } = useAnalytics();

  const trackExport = useCallback(
    (
      exportType: 'pdf' | 'csv' | 'excel' | 'json',
      dataType: string,
      recordCount?: number,
      properties?: EventProperties
    ) => {
      track('export_generated', 'feature', {
        export_type: exportType,
        data_type: dataType,
        record_count: recordCount,
        ...properties,
      });
    },
    [track]
  );

  return { trackExport };
}

/**
 * Hook for tracking modal interactions
 */
export function useModalTracking(modalId: string, modalName?: string) {
  const { track } = useAnalytics();
  const openTime = useRef<number>(0);

  const trackOpen = useCallback(() => {
    openTime.current = Date.now();
    track('feature_used', 'feature', {
      feature_name: 'modal',
      action: 'opened',
      modal_id: modalId,
      modal_name: modalName,
    });
  }, [modalId, modalName, track]);

  const trackClose = useCallback(
    (action: 'dismiss' | 'confirm' | 'cancel' = 'dismiss') => {
      const duration = Date.now() - openTime.current;
      track('feature_used', 'feature', {
        feature_name: 'modal',
        action: `closed_${action}`,
        modal_id: modalId,
        modal_name: modalName,
        duration_ms: duration,
      });
    },
    [modalId, modalName, track]
  );

  return {
    trackOpen,
    trackClose,
  };
}
