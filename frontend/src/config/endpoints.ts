/**
 * API and WebSocket endpoint resolution.
 *
 * Production is served from Cloudflare Workers, which proxies /api/* to the
 * Go API same-origin (frontend/worker/index.ts), so the default API base is
 * the relative path "/api". Local dev uses the same path through the Vite dev
 * server proxy (vite.config.ts server.proxy, target VITE_PROXY_TARGET or
 * http://localhost:8080).
 */

/** Default API root. Service paths already include the version ("/v1/..."). */
export const DEFAULT_API_BASE_URL = '/api';

/**
 * Normalise VITE_API_URL into the axios baseURL.
 *
 * - empty/unset            -> "/api" (same-origin; the production default)
 * - "/api" or "/api/v1"    -> "/api" (a trailing "/v1" is dropped because
 *                             every request path already starts with "/v1")
 * - "http://host:8080/api" -> used as-is (cross-origin; the API must allow
 *                             credentialed CORS from this origin)
 */
export function resolveApiBaseUrl(value: string | undefined): string {
  const trimmed = (value ?? '').trim().replace(/\/+$/, '');
  if (!trimmed) {
    return DEFAULT_API_BASE_URL;
  }
  const withoutVersion = trimmed.replace(/\/v1$/, '');
  return withoutVersion || DEFAULT_API_BASE_URL;
}

/** Default WebSocket URL used only in local development. */
export const DEV_WS_URL = 'ws://localhost:8080/api/v1/ws';

interface LocationLike {
  protocol: string;
  host: string;
}

/**
 * Resolve the WebSocket URL from VITE_WS_URL.
 *
 * - absolute ws:// or wss:// URL -> used as-is
 * - relative path ("/api/v1/ws") -> same-origin, wss: on https pages
 * - unset, dev build             -> ws://localhost:8080/api/v1/ws
 * - unset, production build      -> null: real-time updates are disabled
 *   rather than pointing a production page at localhost.
 */
export function resolveWebSocketUrl(
  value: string | undefined,
  isDev: boolean,
  location: LocationLike | null = typeof window !== 'undefined'
    ? window.location
    : null
): string | null {
  const trimmed = (value ?? '').trim();
  if (!trimmed) {
    return isDev ? DEV_WS_URL : null;
  }
  if (trimmed.startsWith('/')) {
    if (!location) {
      return null;
    }
    const scheme = location.protocol === 'https:' ? 'wss:' : 'ws:';
    return `${scheme}//${location.host}${trimmed}`;
  }
  return trimmed;
}

export const API_BASE_URL = resolveApiBaseUrl(import.meta.env.VITE_API_URL);
