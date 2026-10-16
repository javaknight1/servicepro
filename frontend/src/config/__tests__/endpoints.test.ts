import { describe, expect, it } from 'vitest';
import {
  DEFAULT_API_BASE_URL,
  DEV_WS_URL,
  resolveApiBaseUrl,
  resolveWebSocketUrl,
} from '../endpoints';

describe('resolveApiBaseUrl', () => {
  it.each([
    [undefined, DEFAULT_API_BASE_URL],
    ['', DEFAULT_API_BASE_URL],
    ['   ', DEFAULT_API_BASE_URL],
    ['/api', '/api'],
    ['/api/', '/api'],
    ['/api/v1', '/api'],
    ['/api/v1/', '/api'],
    ['/v1', DEFAULT_API_BASE_URL],
    ['http://localhost:8080/api/v1', 'http://localhost:8080/api'],
    ['https://api.example.test/api', 'https://api.example.test/api'],
  ])('%j -> %s', (input, expected) => {
    expect(resolveApiBaseUrl(input)).toBe(expected);
  });
});

describe('resolveWebSocketUrl', () => {
  const https = { protocol: 'https:', host: 'servicepro.sharkfins.xyz' };
  const http = { protocol: 'http:', host: 'localhost:3000' };

  it('uses the localhost default in dev when unset', () => {
    expect(resolveWebSocketUrl(undefined, true, http)).toBe(DEV_WS_URL);
  });

  it('disables WebSockets in production when unset', () => {
    expect(resolveWebSocketUrl(undefined, false, https)).toBeNull();
    expect(resolveWebSocketUrl('  ', false, https)).toBeNull();
  });

  it('turns a relative path into a same-origin wss URL on https', () => {
    expect(resolveWebSocketUrl('/api/v1/ws', false, https)).toBe(
      'wss://servicepro.sharkfins.xyz/api/v1/ws'
    );
  });

  it('turns a relative path into a same-origin ws URL on http', () => {
    expect(resolveWebSocketUrl('/api/v1/ws', true, http)).toBe(
      'ws://localhost:3000/api/v1/ws'
    );
  });

  it('returns null for a relative path without a location', () => {
    expect(resolveWebSocketUrl('/api/v1/ws', false, null)).toBeNull();
  });

  it('passes absolute URLs through', () => {
    expect(resolveWebSocketUrl('wss://rt.example.test/ws', false, https)).toBe(
      'wss://rt.example.test/ws'
    );
  });
});
