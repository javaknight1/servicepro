import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import type { CaptureResult } from 'posthog-js';
import {
  sanitizeSearch,
  sanitizeUrl,
  sanitizeUrlProperties,
} from '../urlSanitizer';

const trackEvent = vi.fn();
vi.mock('@/services/analytics', async () => {
  const events = await vi.importActual<
    typeof import('@/services/analytics/events')
  >('@/services/analytics/events');
  return {
    AnalyticsEvents: events.AnalyticsEvents,
    trackEvent: (...args: unknown[]) => trackEvent(...args),
  };
});

describe('urlSanitizer', () => {
  it('drops secret-bearing query params', () => {
    expect(sanitizeSearch('?token=abc123')).toBe('');
    expect(sanitizeSearch('?token=abc&email=a%40b.com&code=9')).toBe('');
    expect(sanitizeSearch('?session_id=cs_test_123')).toBe('');
  });

  it('keeps allowlisted params', () => {
    expect(sanitizeSearch('?page=2&token=abc&status=paid')).toBe(
      '?page=2&status=paid'
    );
    expect(sanitizeSearch('')).toBe('');
  });

  it('sanitises absolute and relative URLs and drops fragments', () => {
    expect(
      sanitizeUrl('https://app.example.com/reset-password?token=abc#x')
    ).toBe('https://app.example.com/reset-password');
    expect(sanitizeUrl('/invitations/accept?token=abc&page=1')).toBe(
      '/invitations/accept?page=1'
    );
    expect(sanitizeUrl('/verify-email?token=xyz')).toBe('/verify-email');
  });

  it('sanitises PostHog URL properties', () => {
    const props = sanitizeUrlProperties({
      $current_url: 'https://app.example.com/reset-password?token=secret',
      $referrer: 'https://mail.example.com/?email=a@b.com',
      search: '?token=secret',
      other: 'kept',
    });
    expect(JSON.stringify(props)).not.toContain('secret');
    expect(JSON.stringify(props)).not.toContain('a@b.com');
    expect(props.other).toBe('kept');
  });
});

describe('PostHog before_send', () => {
  it('removes tokens from $current_url', async () => {
    const { sanitizeCaptureResult } =
      await import('@/services/analytics/posthog');
    const result = sanitizeCaptureResult({
      uuid: '1',
      event: '$pageview',
      properties: {
        $current_url: 'https://app.example.com/verify-email?token=secret',
      },
      $set: { $initial_current_url: 'https://x.test/?token=secret' },
    } as CaptureResult);
    expect(JSON.stringify(result)).not.toContain('secret');
  });
});

describe('PageViewTracker', () => {
  beforeEach(() => {
    trackEvent.mockClear();
  });

  it('never sends reset/verify/invite tokens to analytics', async () => {
    const { PageViewTracker } = await import('../../App');

    for (const url of [
      '/reset-password?token=reset-secret',
      '/verify-email?token=verify-secret&email=a%40b.com',
      '/invitations/accept?token=invite-secret',
    ]) {
      render(
        <MemoryRouter initialEntries={[url]}>
          <PageViewTracker />
        </MemoryRouter>
      );
    }

    expect(trackEvent).toHaveBeenCalledTimes(3);
    const sent = JSON.stringify(trackEvent.mock.calls);
    expect(sent).not.toMatch(/secret|a%40b\.com|a@b\.com/);
    expect(trackEvent).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({ path: '/reset-password', search: '' })
    );
  });

  it('keeps allowlisted params like page', async () => {
    const { PageViewTracker } = await import('../../App');
    render(
      <MemoryRouter initialEntries={['/invoices?page=3&token=x']}>
        <PageViewTracker />
      </MemoryRouter>
    );
    expect(trackEvent).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({ path: '/invoices', search: '?page=3' })
    );
  });
});
