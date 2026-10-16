/**
 * Same-origin API proxy for the ServicePro edge Worker (servicepro.sharkfins.xyz).
 * Entry point: worker/index.ts. Kept separate because every named export of
 * the main module is treated as a Worker entrypoint by the runtime.
 *
 * Static assets (the Vite build in ./dist) are served by Workers static
 * assets with SPA fallback; this script only runs for the paths listed in
 * wrangler.jsonc `assets.run_worker_first` (/api/* and /version). Those are
 * proxied same-origin to the Go API on Cloud Run, so the browser never makes
 * a cross-origin call: no CORS preflight, and auth cookies stay first-party.
 *
 * Client IP: our subrequest reaches Cloud Run from a Cloudflare egress
 * address, so the API would otherwise see the same IP for every visitor (one
 * rate-limit bucket for everyone). We forward CF-Connecting-IP (set by
 * Cloudflare, not the browser) as X-SP-Client-IP, authenticated by
 * X-SP-Edge-Secret. The API must trust X-SP-Client-IP only when that secret
 * matches its own EDGE_PROXY_SECRET.
 *
 * Request bodies are streamed through untouched: Stripe webhook signature
 * verification needs the exact raw bytes.
 */

export const EDGE_SECRET_HEADER = 'X-SP-Edge-Secret';
export const EDGE_CLIENT_IP_HEADER = 'X-SP-Client-IP';

// `Env` (ASSETS, API_ORIGIN, EDGE_PROXY_SECRET) is generated from wrangler.jsonc
// into worker/worker-configuration.d.ts by `npm run cf:typegen`.

/**
 * Hop-by-hop headers (RFC 9110 section 7.6.1) describe the client's
 * connection to us, not ours to the origin, so they are never forwarded.
 * `upgrade`/`connection` are kept for WebSocket upgrades (see below).
 */
const HOP_BY_HOP_HEADERS = [
  'connection',
  'keep-alive',
  'proxy-authenticate',
  'proxy-authorization',
  'proxy-connection',
  'te',
  'trailer',
  'transfer-encoding',
  'upgrade',
];

/** Headers a caller must never be able to supply itself. */
const SPOOFABLE_HEADERS = [
  EDGE_SECRET_HEADER,
  EDGE_CLIENT_IP_HEADER,
  'X-Forwarded-Host',
  'X-Forwarded-Proto',
];

export function isProxiedPath(pathname: string): boolean {
  return (
    pathname === '/version' ||
    pathname === '/api' ||
    pathname.startsWith('/api/')
  );
}

function jsonError(status: number, error: string): Response {
  return new Response(JSON.stringify({ error }), {
    status,
    headers: {
      'Content-Type': 'application/json',
      'Cache-Control': 'no-store',
    },
  });
}

export function buildUpstreamHeaders(request: Request, env: Env): Headers {
  const url = new URL(request.url);
  const headers = new Headers(request.headers);

  const isWebSocket =
    request.headers.get('Upgrade')?.toLowerCase() === 'websocket';

  // Headers named in Connection are hop-by-hop too.
  const connectionTokens = (request.headers.get('Connection') ?? '')
    .split(',')
    .map((token) => token.trim().toLowerCase())
    .filter(Boolean);

  for (const name of [...HOP_BY_HOP_HEADERS, ...connectionTokens]) {
    if (isWebSocket && (name === 'upgrade' || name === 'connection')) {
      continue;
    }
    headers.delete(name);
  }

  for (const name of SPOOFABLE_HEADERS) {
    headers.delete(name);
  }

  const clientIP = request.headers.get('CF-Connecting-IP');
  if (clientIP) {
    headers.set(EDGE_CLIENT_IP_HEADER, clientIP);
  }
  if (env.EDGE_PROXY_SECRET) {
    headers.set(EDGE_SECRET_HEADER, env.EDGE_PROXY_SECRET);
  }

  headers.set('X-Forwarded-Host', url.host);
  headers.set('X-Forwarded-Proto', url.protocol.replace(/:$/, ''));

  return headers;
}

export async function proxyToApi(
  request: Request,
  env: Env
): Promise<Response> {
  const apiOrigin = (env.API_ORIGIN ?? '').trim().replace(/\/+$/, '');
  if (!apiOrigin) {
    return jsonError(503, 'api_not_configured');
  }

  const url = new URL(request.url);
  const target = apiOrigin + url.pathname + url.search;
  const hasBody = request.method !== 'GET' && request.method !== 'HEAD';

  try {
    return await fetch(target, {
      method: request.method,
      headers: buildUpstreamHeaders(request, env),
      // Stream the raw body; never parse or re-encode it.
      body: hasBody ? request.body : undefined,
      // Pass redirects to the browser instead of following them here.
      redirect: 'manual',
    });
  } catch (err) {
    console.error('[EDGE-PROXY] upstream fetch failed', err);
    return jsonError(502, 'api_unavailable');
  }
}

export async function handleRequest(
  request: Request,
  env: Env
): Promise<Response> {
  const { pathname } = new URL(request.url);
  if (isProxiedPath(pathname)) {
    return proxyToApi(request, env);
  }
  // Only reachable if run_worker_first is widened; let assets (with SPA
  // fallback) answer.
  return env.ASSETS.fetch(request);
}
