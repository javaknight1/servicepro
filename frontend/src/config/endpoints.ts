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

/**
 * URL of the API's build-identity endpoint, GET /version.
 *
 * /version is deliberately unversioned and lives at the API's root, not under
 * "/api" (backend routes.go; the Worker proxies it next to /api/*), so it is
 * derived from the API base by dropping a trailing "/api":
 *
 * - "/api"                  -> "/version" (same-origin)
 * - "http://host:8080/api"  -> "http://host:8080/version"
 */
export function resolveVersionUrl(apiBaseUrl: string): string {
  return `${apiBaseUrl.replace(/\/+$/, '').replace(/\/api$/, '')}/version`;
}

export const VERSION_URL = resolveVersionUrl(API_BASE_URL);
