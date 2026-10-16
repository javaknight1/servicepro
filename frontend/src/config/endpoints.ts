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

export const API_BASE_URL = resolveApiBaseUrl(import.meta.env.VITE_API_URL);
