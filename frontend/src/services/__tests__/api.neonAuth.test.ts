/**
 * The axios interceptors on a Neon Auth build (VITE_NEON_AUTH_URL set, T158):
 * Bearer JWT on every request, one retry with a fresh JWT on 401, sign-out
 * when Neon Auth has no session, account_not_linked left to the caller.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import axios, {
  AxiosError,
  type AxiosAdapter,
  type AxiosResponse,
  type InternalAxiosRequestConfig,
} from 'axios';

vi.mock('@config/auth', () => ({
  NEON_AUTH_URL: 'https://auth.example.test/neondb/auth',
  isNeonAuthEnabled: true,
  isGoogleSignInEnabled: false,
}));

const getNeonAccessToken = vi.fn();
const clearNeonAccessToken = vi.fn();
vi.mock('../neonAuth', () => ({ getNeonAccessToken, clearNeonAccessToken }));

import api from '../api';

type Reply = { status: number; data?: object };

/** Adapter answering each request in turn, recording its Authorization. */
function scripted(replies: Reply[], seen: (string | undefined)[]): AxiosAdapter {
  return (config: InternalAxiosRequestConfig) => {
    seen.push(config.headers.Authorization as string | undefined);
    const { status, data = {} } = replies.shift() ?? { status: 200 };
    const response: AxiosResponse = {
      data,
      status,
      statusText: String(status),
      headers: {},
      config,
    };
    if (status >= 400) {
      return Promise.reject(
        new AxiosError('failed', String(status), config, null, response)
      );
    }
    return Promise.resolve(response);
  };
}

describe('api interceptors with Neon Auth', () => {
  const originalAdapter = api.defaults.adapter;
  const originalLocation = window.location;
  let hrefSetter: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    vi.clearAllMocks();
    hrefSetter = vi.fn();
    Object.defineProperty(window, 'location', {
      configurable: true,
      value: {
        ...originalLocation,
        pathname: '/dashboard',
        set href(value: string) {
          hrefSetter(value);
        },
        get href() {
          return 'http://localhost/dashboard';
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
  });

  it('sends the Neon Auth JWT as a Bearer token', async () => {
    const seen: (string | undefined)[] = [];
    getNeonAccessToken.mockResolvedValue('jwt-1');
    api.defaults.adapter = scripted([{ status: 200 }], seen);

    await api.get('/v1/users/me');
    expect(seen).toEqual(['Bearer jwt-1']);
  });

  it('sends no Authorization header when signed out', async () => {
    const seen: (string | undefined)[] = [];
    getNeonAccessToken.mockResolvedValue(null);
    api.defaults.adapter = scripted([{ status: 200 }], seen);

    await api.get('/v1/billing/config');
    expect(seen).toEqual([undefined]);
  });

  it('still sends the request when Neon Auth is unreachable', async () => {
    const seen: (string | undefined)[] = [];
    getNeonAccessToken.mockRejectedValue(new TypeError('Failed to fetch'));
    api.defaults.adapter = scripted([{ status: 200 }], seen);

    await api.get('/v1/billing/config');
    expect(seen).toEqual([undefined]);
  });

  it('retries a 401 once with a freshly minted JWT', async () => {
    const seen: (string | undefined)[] = [];
    getNeonAccessToken
      .mockResolvedValueOnce('stale')
      .mockResolvedValueOnce('fresh');
    api.defaults.adapter = scripted(
      [{ status: 401, data: { error: 'unauthorized' } }, { status: 200 }],
      seen
    );

    const res = await api.get('/v1/jobs');
    expect(res.status).toBe(200);
    expect(seen).toEqual(['Bearer stale', 'Bearer fresh']);
    expect(getNeonAccessToken).toHaveBeenLastCalledWith({ forceRefresh: true });
    expect(hrefSetter).not.toHaveBeenCalled();
  });

  it('signs out to /login when Neon Auth has no session any more', async () => {
    const seen: (string | undefined)[] = [];
    getNeonAccessToken.mockResolvedValueOnce('stale').mockResolvedValueOnce(null);
    localStorage.setItem('auth-storage', '{"state":{"isAuthenticated":true}}');
    api.defaults.adapter = scripted(
      [{ status: 401, data: { error: 'unauthorized' } }],
      seen
    );

    await expect(api.get('/v1/jobs')).rejects.toMatchObject({
      response: { status: 401 },
    });
    expect(clearNeonAccessToken).toHaveBeenCalled();
    expect(localStorage.getItem('auth-storage')).toBeNull();
    expect(hrefSetter).toHaveBeenCalledWith('/login');
  });

  it('signs out when the retried request is refused again', async () => {
    const seen: (string | undefined)[] = [];
    getNeonAccessToken.mockResolvedValue('jwt');
    api.defaults.adapter = scripted(
      [
        { status: 401, data: { error: 'unauthorized' } },
        { status: 401, data: { error: 'unauthorized' } },
      ],
      seen
    );

    await expect(api.get('/v1/jobs')).rejects.toBeDefined();
    expect(seen).toHaveLength(2);
    expect(hrefSetter).toHaveBeenCalledWith('/login');
  });

  it('leaves account_not_linked to the sign-in flow (no retry, no redirect)', async () => {
    const seen: (string | undefined)[] = [];
    getNeonAccessToken.mockResolvedValue('jwt');
    api.defaults.adapter = scripted(
      [{ status: 401, data: { error: 'account_not_linked' } }],
      seen
    );

    await expect(api.get('/v1/users/me')).rejects.toMatchObject({
      response: { status: 401 },
    });
    expect(seen).toHaveLength(1);
    expect(hrefSetter).not.toHaveBeenCalled();
  });

  it('never calls the cookie refresh endpoint', async () => {
    const postSpy = vi.spyOn(axios, 'post');
    getNeonAccessToken.mockResolvedValue(null);
    api.defaults.adapter = scripted(
      [{ status: 401, data: { error: 'unauthorized' } }],
      []
    );

    await expect(api.get('/v1/jobs')).rejects.toBeDefined();
    expect(postSpy).not.toHaveBeenCalled();
  });

  it('passes provision errors straight to the caller', async () => {
    getNeonAccessToken.mockResolvedValue('jwt');
    api.defaults.adapter = scripted(
      [{ status: 401, data: { error: 'unauthorized' } }],
      []
    );
    await expect(api.post('/v1/auth/neon/provision', {})).rejects.toBeDefined();
    expect(getNeonAccessToken).toHaveBeenCalledTimes(1);
    expect(hrefSetter).not.toHaveBeenCalled();
  });
});
