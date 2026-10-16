import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import worker from './index';
import {
  EDGE_CLIENT_IP_HEADER,
  EDGE_SECRET_HEADER,
  handleRequest,
  isProxiedPath,
} from './proxy';

const API_ORIGIN = 'https://servicepro-api-abc123-uc.a.run.app';
const SECRET = 'edge-secret-for-tests';

interface UpstreamCall {
  url: string;
  init: RequestInit;
  headers: Headers;
}

function makeEnv(overrides: Partial<Env> = {}): Env & {
  assetsFetch: ReturnType<typeof vi.fn>;
} {
  const assetsFetch = vi.fn(
    async () => new Response('<!doctype html>', { status: 200 })
  );
  const ASSETS = { fetch: assetsFetch } as unknown as Fetcher;
  return {
    ASSETS,
    API_ORIGIN,
    EDGE_PROXY_SECRET: SECRET,
    ...overrides,
    assetsFetch,
  };
}

function makeRequest(path: string, init: RequestInit = {}): Request {
  return new Request(`https://servicepro.sharkfins.xyz${path}`, init);
}

describe('edge worker', () => {
  let calls: UpstreamCall[];
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    calls = [];
    fetchMock = vi.fn(async (input: string, init: RequestInit) => {
      calls.push({ url: input, init, headers: new Headers(init.headers) });
      return new Response(JSON.stringify({ ok: true }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      });
    });
    vi.stubGlobal('fetch', fetchMock);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  describe('isProxiedPath', () => {
    it.each([
      ['/api', true],
      ['/api/', true],
      ['/api/v1/customers', true],
      ['/version', true],
      ['/', false],
      ['/apiary', false],
      ['/versions', false],
      ['/dashboard', false],
      ['/assets/index-abc.js', false],
    ])('%s -> %s', (path, expected) => {
      expect(isProxiedPath(path)).toBe(expected);
    });
  });

  it('returns 503 JSON when API_ORIGIN is not configured', async () => {
    const env = makeEnv({ API_ORIGIN: '' });
    const res = await handleRequest(makeRequest('/api/v1/auth/me'), env);

    expect(res.status).toBe(503);
    expect(res.headers.get('Content-Type')).toBe('application/json');
    expect(await res.json()).toEqual({ error: 'api_not_configured' });
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('returns 503 for /version when API_ORIGIN is whitespace only', async () => {
    const env = makeEnv({ API_ORIGIN: '   ' });
    const res = await handleRequest(makeRequest('/version'), env);
    expect(res.status).toBe(503);
  });

  it('forwards path and query to API_ORIGIN (trailing slash trimmed)', async () => {
    const env = makeEnv({ API_ORIGIN: `${API_ORIGIN}/` });
    await handleRequest(
      makeRequest('/api/v1/customers?page=2&search=a%20b'),
      env
    );

    expect(calls).toHaveLength(1);
    expect(calls[0].url).toBe(
      `${API_ORIGIN}/api/v1/customers?page=2&search=a%20b`
    );
    expect(calls[0].init.method).toBe('GET');
    expect(calls[0].init.redirect).toBe('manual');
    expect(calls[0].init.body).toBeUndefined();
  });

  it('proxies /version', async () => {
    await handleRequest(makeRequest('/version'), makeEnv());
    expect(calls[0].url).toBe(`${API_ORIGIN}/version`);
  });

  it('strips caller-supplied edge headers and injects the secret and client IP', async () => {
    const req = makeRequest('/api/v1/jobs', {
      headers: {
        [EDGE_SECRET_HEADER]: 'forged',
        [EDGE_CLIENT_IP_HEADER]: '6.6.6.6',
        'CF-Connecting-IP': '203.0.113.7',
        'X-Forwarded-Host': 'evil.example',
        'X-Forwarded-Proto': 'http',
        Cookie: 'sp_session=abc',
        'X-Tenant-ID': 'tenant-1',
      },
    });
    await handleRequest(req, makeEnv());

    const headers = calls[0].headers;
    expect(headers.get(EDGE_SECRET_HEADER)).toBe(SECRET);
    expect(headers.get(EDGE_CLIENT_IP_HEADER)).toBe('203.0.113.7');
    expect(headers.get('X-Forwarded-Host')).toBe('servicepro.sharkfins.xyz');
    expect(headers.get('X-Forwarded-Proto')).toBe('https');
    // Ordinary headers pass through.
    expect(headers.get('Cookie')).toBe('sp_session=abc');
    expect(headers.get('X-Tenant-ID')).toBe('tenant-1');
  });

  it('never forwards a forged secret or client IP when the secret is unset', async () => {
    const req = makeRequest('/api/v1/jobs', {
      headers: {
        [EDGE_SECRET_HEADER]: 'forged',
        [EDGE_CLIENT_IP_HEADER]: '6.6.6.6',
      },
    });
    await handleRequest(req, makeEnv({ EDGE_PROXY_SECRET: '' }));

    const headers = calls[0].headers;
    expect(headers.has(EDGE_SECRET_HEADER)).toBe(false);
    expect(headers.has(EDGE_CLIENT_IP_HEADER)).toBe(false);
  });

  it('strips hop-by-hop headers, including ones named in Connection', async () => {
    const req = makeRequest('/api/v1/jobs', {
      headers: {
        Connection: 'keep-alive, X-Custom-Hop',
        'Keep-Alive': 'timeout=5',
        'Proxy-Authorization': 'Basic Zm9vOmJhcg==',
        TE: 'trailers',
        'X-Custom-Hop': '1',
        Authorization: 'Bearer token',
      },
    });
    await handleRequest(req, makeEnv());

    const headers = calls[0].headers;
    for (const name of [
      'Connection',
      'Keep-Alive',
      'Proxy-Authorization',
      'TE',
      'X-Custom-Hop',
    ]) {
      expect(headers.has(name)).toBe(false);
    }
    expect(headers.get('Authorization')).toBe('Bearer token');
  });

  it('keeps Upgrade/Connection for WebSocket upgrades', async () => {
    const req = makeRequest('/api/v1/ws', {
      headers: { Upgrade: 'websocket', Connection: 'Upgrade' },
    });
    await handleRequest(req, makeEnv());

    expect(calls[0].headers.get('Upgrade')).toBe('websocket');
    expect(calls[0].headers.get('Connection')).toBe('Upgrade');
  });

  it('streams the raw request body untouched (Stripe webhooks)', async () => {
    const payload = '{"id":"evt_1","type":"invoice.paid"}\n  ';
    const req = makeRequest('/api/v1/stripe/webhooks', {
      method: 'POST',
      body: payload,
      headers: {
        'Content-Type': 'application/json',
        'Stripe-Signature': 't=1,v1=abc',
      },
    });
    await handleRequest(req, makeEnv());

    const { init, headers } = calls[0];
    expect(init.method).toBe('POST');
    expect(init.body).toBe(req.body);
    expect(await new Response(init.body).text()).toBe(payload);
    expect(headers.get('Stripe-Signature')).toBe('t=1,v1=abc');
  });

  it('passes upstream redirects and statuses through unchanged', async () => {
    fetchMock.mockResolvedValueOnce(
      new Response(null, {
        status: 302,
        headers: { Location: 'https://connect.stripe.com/x' },
      })
    );
    const res = await handleRequest(
      makeRequest('/api/v1/stripe/connect'),
      makeEnv()
    );
    expect(res.status).toBe(302);
    expect(res.headers.get('Location')).toBe('https://connect.stripe.com/x');
  });

  it('returns 502 JSON when the upstream fetch throws', async () => {
    const consoleError = vi
      .spyOn(console, 'error')
      .mockImplementation(() => undefined);
    fetchMock.mockRejectedValueOnce(new TypeError('network down'));
    const res = await handleRequest(makeRequest('/api/v1/jobs'), makeEnv());
    expect(res.status).toBe(502);
    expect(await res.json()).toEqual({ error: 'api_unavailable' });
    consoleError.mockRestore();
  });

  it('falls through to static assets for non-API paths', async () => {
    const env = makeEnv();
    const req = makeRequest('/customers/123');
    const res = await handleRequest(req, env);

    expect(res.status).toBe(200);
    expect(env.assetsFetch).toHaveBeenCalledWith(req);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('exposes a module fetch handler', async () => {
    const env = makeEnv({ API_ORIGIN: '' });
    const ctx = {} as ExecutionContext;
    const handler = worker.fetch as (
      req: Request,
      env: Env,
      ctx: ExecutionContext
    ) => Promise<Response>;
    const res = await handler(makeRequest('/api/v1/x'), env, ctx);
    expect(res.status).toBe(503);
  });
});
