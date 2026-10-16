import { describe, expect, it } from 'vitest';
import { DEFAULT_API_BASE_URL, resolveApiBaseUrl } from '../endpoints';

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
