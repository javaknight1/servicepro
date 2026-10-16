import { describe, it, expect, vi } from 'vitest';
import type { Event } from '@sentry/react';

// The global test setup mocks this module; test the real implementation.
vi.unmock('../sentry');

describe('Sentry URL sanitisation', () => {
  it('strips tokens from navigation and fetch breadcrumbs', async () => {
    const { sanitizeBreadcrumbUrls } = await import('../sentry');
    const data = sanitizeBreadcrumbUrls({
      from: '/login',
      to: '/reset-password?token=secret',
      url: '/api/v1/auth/verify?token=secret',
      method: 'GET',
    });
    expect(JSON.stringify(data)).not.toContain('secret');
    expect(data.to).toBe('/reset-password');
    expect(data.method).toBe('GET');
  });

  it('strips tokens from the event request URL and query string', async () => {
    const { sanitizeEvent } = await import('../sentry');
    const event = sanitizeEvent({
      request: {
        url: 'https://app.example.com/invitations/accept?token=secret&page=2',
        query_string: 'token=secret&page=2',
      },
    } as Event);
    expect(JSON.stringify(event)).not.toContain('secret');
    expect(event.request?.url).toBe(
      'https://app.example.com/invitations/accept?page=2'
    );
    expect(event.request?.query_string).toBe('page=2');
  });
});
