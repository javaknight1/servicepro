/**
 * Query Parameters Utilities
 *
 * Centralized utilities for building URL query parameters.
 * Consolidates duplicate URLSearchParams logic across the codebase.
 */

type QueryValue = string | number | boolean | undefined | null;

/**
 * Builds URLSearchParams from an object, automatically handling:
 * - Filtering out undefined, null, and empty string values
 * - Converting numbers and booleans to strings
 *
 * @example
 * const params = buildQueryParams({
 *   page: 1,
 *   search: 'test',
 *   active: true,
 *   empty: undefined,
 * });
 * // Results in: "page=1&search=test&active=true"
 */
export function buildQueryParams(
  obj: Record<string, QueryValue>
): URLSearchParams {
  const params = new URLSearchParams();

  Object.entries(obj).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== '') {
      params.append(key, String(value));
    }
  });

  return params;
}

/**
 * Builds a query string from an object.
 * Returns the string with a leading '?' if there are params, or empty string if not.
 *
 * @example
 * const queryString = buildQueryString({ page: 1, search: 'test' });
 * // Returns: "?page=1&search=test"
 */
export function buildQueryString(obj: Record<string, QueryValue>): string {
  const params = buildQueryParams(obj);
  const str = params.toString();
  return str ? `?${str}` : '';
}

/**
 * Appends query parameters to a base URL.
 *
 * @example
 * const url = appendQueryParams('/api/users', { page: 1, search: 'test' });
 * // Returns: "/api/users?page=1&search=test"
 */
export function appendQueryParams(
  baseUrl: string,
  obj: Record<string, QueryValue>
): string {
  const queryString = buildQueryString(obj);
  return `${baseUrl}${queryString}`;
}
