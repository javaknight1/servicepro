/**
 * URL sanitisation for third-party telemetry (PostHog, Sentry).
 *
 * Several routes carry single-use secrets in the query string
 * (`/reset-password?token=…`, `/verify-email?token=…`,
 * `/invitations/accept?token=…`, Stripe `session_id`, emails, codes).
 * Only an allowlist of harmless parameters is forwarded; everything else is
 * dropped, and fragments are removed.
 */

/** Query parameters that are safe to send to analytics / error tracking. */
export const SAFE_QUERY_PARAMS: ReadonlySet<string> = new Set([
  'page',
  'page_size',
  'per_page',
  'sort',
  'sort_by',
  'sort_order',
  'order',
  'status',
  'priority',
  'view',
  'tab',
  'type',
  'period',
  'range',
  'clone',
  'utm_source',
  'utm_medium',
  'utm_campaign',
  'utm_term',
  'utm_content',
]);

/**
 * Keep only allowlisted params from a `?a=b&c=d` search string.
 * Returns '' when nothing safe remains, otherwise a string starting with '?'.
 */
export function sanitizeSearch(search: string | null | undefined): string {
  if (!search) return '';
  const params = new URLSearchParams(
    search.startsWith('?') ? search.slice(1) : search
  );
  const safe = new URLSearchParams();
  params.forEach((value, key) => {
    if (SAFE_QUERY_PARAMS.has(key.toLowerCase())) {
      safe.append(key, value);
    }
  });
  const result = safe.toString();
  return result ? `?${result}` : '';
}

/**
 * Sanitise an absolute or relative URL: keep origin + path, filter the query
 * string through the allowlist and drop the fragment. Non-URL strings are
 * returned with any query/fragment removed.
 */
export function sanitizeUrl(url: string): string {
  if (!url) return url;
  const isAbsolute = /^[a-z][a-z\d+\-.]*:\/\//i.test(url);
  try {
    const parsed = new URL(url, 'http://placeholder.invalid');
    const search = sanitizeSearch(parsed.search);
    if (isAbsolute) {
      return `${parsed.origin}${parsed.pathname}${search}`;
    }
    // Preserve relative form
    const path = url.startsWith('/')
      ? parsed.pathname
      : parsed.pathname.slice(1);
    return `${path}${search}`;
  } catch {
    return url.split(/[?#]/)[0];
  }
}

/** PostHog properties that carry URLs. */
const URL_PROPERTIES = [
  '$current_url',
  '$referrer',
  '$initial_current_url',
  '$initial_referrer',
  '$pathname',
  'url',
  'search',
];

/** Sanitise URL-bearing properties of an analytics event in place. */
export function sanitizeUrlProperties<T extends Record<string, unknown>>(
  properties: T
): T {
  const out: Record<string, unknown> = { ...properties };
  for (const key of URL_PROPERTIES) {
    const value = out[key];
    if (typeof value === 'string') {
      out[key] = key === 'search' ? sanitizeSearch(value) : sanitizeUrl(value);
    }
  }
  if (out.$set && typeof out.$set === 'object') {
    out.$set = sanitizeUrlProperties(out.$set as Record<string, unknown>);
  }
  if (out.$set_once && typeof out.$set_once === 'object') {
    out.$set_once = sanitizeUrlProperties(
      out.$set_once as Record<string, unknown>
    );
  }
  return out as T;
}

/**
 * Validate a post-login `returnUrl` query value. Only same-origin, absolute
 * paths are allowed (no `//host`, `/\host`, schemes or control characters),
 * so the login page cannot be used as an open redirect.
 */
export function getSafeReturnUrl(
  raw: string | null | undefined
): string | null {
  if (!raw) return null;
  if (!raw.startsWith('/') || raw.startsWith('//') || raw.startsWith('/\\')) {
    return null;
  }
  // eslint-disable-next-line no-control-regex
  if (/[\u0000-\u001f\u007f]/.test(raw)) return null;
  try {
    const url = new URL(raw, 'https://placeholder.invalid');
    if (url.origin !== 'https://placeholder.invalid') return null;
    return `${url.pathname}${url.search}${url.hash}`;
  } catch {
    return null;
  }
}
