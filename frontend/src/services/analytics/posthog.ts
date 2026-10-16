import posthog, { type CaptureResult } from 'posthog-js';
import { sanitizeUrlProperties } from '@/utils/urlSanitizer';

export interface AnalyticsConfig {
  apiKey: string;
  apiHost?: string;
}

interface UserIdentity {
  id: string;
  email?: string;
  role?: string;
}

let initialized = false;

/**
 * Strip secrets (reset/verify/invite tokens, emails, Stripe session ids…)
 * from URL-bearing properties before any event leaves the browser.
 */
export function sanitizeCaptureResult(
  result: CaptureResult | null
): CaptureResult | null {
  if (!result) return result;
  return {
    ...result,
    properties: sanitizeUrlProperties(result.properties ?? {}),
    ...(result.$set ? { $set: sanitizeUrlProperties(result.$set) } : {}),
    ...(result.$set_once
      ? { $set_once: sanitizeUrlProperties(result.$set_once) }
      : {}),
  };
}

/**
 * Initialize PostHog analytics.
 * No API key = no-op (analytics disabled).
 */
export function initAnalytics(config: AnalyticsConfig): void {
  if (!config.apiKey) {
    console.warn(
      '[ANALYTICS] PostHog API key not configured. Analytics disabled.'
    );
    return;
  }

  posthog.init(config.apiKey, {
    api_host: config.apiHost || 'https://us.i.posthog.com',
    autocapture: false,
    capture_pageview: false,
    capture_pageleave: false,
    disable_session_recording: true,
    persistence: 'localStorage',
    before_send: sanitizeCaptureResult,
  });

  initialized = true;
}

/**
 * Track a custom analytics event.
 */
export function trackEvent(
  event: string,
  properties?: Record<string, unknown>
): void {
  if (!initialized) return;
  posthog.capture(event, properties);
}

/**
 * Identify a user for analytics.
 */
export function identifyUser(user: UserIdentity): void {
  if (!initialized) return;
  posthog.identify(user.id, {
    email: user.email,
    role: user.role,
  });
}

/**
 * Clear user identity on logout.
 */
export function resetUser(): void {
  if (!initialized) return;
  posthog.reset();
}
