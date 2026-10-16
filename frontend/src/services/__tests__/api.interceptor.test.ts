import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import axios, {
  AxiosError,
  type AxiosAdapter,
  type AxiosResponse,
  type InternalAxiosRequestConfig,
} from 'axios';
import api, { isAuthEndpoint } from '../api';

/**
 * Exercises the real response interceptor by swapping the axios adapter,
 * so no network is touched.
 */
function respondWith(status: number): AxiosAdapter {
  return (config: InternalAxiosRequestConfig) => {
    const response: AxiosResponse = {
      data: { error: status === 401 ? 'Invalid credentials' : 'ok' },
      status,
      statusText: String(status),
      headers: {},
      config,
    };
    if (status >= 400) {
      return Promise.reject(
        new AxiosError(
          `Request failed with status code ${status}`,
          String(status),
          config,
          null,
          response
        )
      );
    }
    return Promise.resolve(response);
  };
}

describe('api 401 interceptor', () => {
  const originalAdapter = api.defaults.adapter;
  const originalLocation = window.location;
  let hrefSetter: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    hrefSetter = vi.fn();
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: {
        ...originalLocation,
        pathname: '/login',
        set href(value: string) {
          hrefSetter(value);
        },
        get href() {
          return 'http://localhost/login';
        },
      },
    });
  });

  afterEach(() => {
    api.defaults.adapter = originalAdapter;
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: originalLocation,
    });
    vi.restoreAllMocks();
  });

  it('does not attempt refresh or redirect when login returns 401', async () => {
    const refreshSpy = vi.spyOn(axios, 'post');
    api.defaults.adapter = respondWith(401);

    await expect(
      api.post('/v1/auth/login', { email: 'a@b.c', password: 'wrong' })
    ).rejects.toMatchObject({ response: { status: 401 } });

    expect(refreshSpy).not.toHaveBeenCalled();
    expect(hrefSetter).not.toHaveBeenCalled();
  });

  it.each([
    '/v1/auth/register',
    '/v1/auth/refresh',
    '/v1/auth/reset-request',
    '/v1/auth/reset-password',
    '/v1/auth/verify',
  ])('passes 401 from %s straight to the caller', async (url) => {
    const refreshSpy = vi.spyOn(axios, 'post');
    api.defaults.adapter = respondWith(401);

    await expect(api.post(url, {})).rejects.toMatchObject({
      response: { status: 401 },
    });
    expect(refreshSpy).not.toHaveBeenCalled();
  });

  it('refreshes once and retries a protected request on 401', async () => {
    let calls = 0;
    api.defaults.adapter = (config: InternalAxiosRequestConfig) => {
      calls += 1;
      return calls === 1 ? respondWith(401)(config) : respondWith(200)(config);
    };
    const refreshSpy = vi
      .spyOn(axios, 'post')
      .mockResolvedValue({ data: {}, status: 200 });

    const res = await api.get('/v1/customers');

    expect(res.status).toBe(200);
    expect(refreshSpy).toHaveBeenCalledTimes(1);
    expect(refreshSpy).toHaveBeenCalledWith(
      '/api/v1/auth/refresh',
      {},
      { withCredentials: true }
    );
  });

  it('shares a single refresh across concurrent 401s', async () => {
    const seen = new Set<string>();
    api.defaults.adapter = (config: InternalAxiosRequestConfig) => {
      const key = config.url ?? '';
      if (!seen.has(key)) {
        seen.add(key);
        return respondWith(401)(config);
      }
      return respondWith(200)(config);
    };
    const refreshSpy = vi
      .spyOn(axios, 'post')
      .mockResolvedValue({ data: {}, status: 200 });

    await Promise.all([
      api.get('/v1/customers'),
      api.get('/v1/jobs'),
      api.get('/v1/invoices'),
    ]);

    expect(refreshSpy).toHaveBeenCalledTimes(1);
  });

  describe('isAuthEndpoint', () => {
    it('matches auth endpoints and not others', () => {
      expect(isAuthEndpoint('/v1/auth/login')).toBe(true);
      expect(isAuthEndpoint('/api/v1/auth/refresh')).toBe(true);
      expect(isAuthEndpoint('/v1/auth/verify?token=x')).toBe(true);
      expect(isAuthEndpoint('/v1/customers')).toBe(false);
      expect(isAuthEndpoint('/v1/users/me')).toBe(false);
      expect(isAuthEndpoint(undefined)).toBe(false);
    });
  });
});
