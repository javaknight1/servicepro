import { AxiosError } from 'axios';

/**
 * API Error response structure
 */
interface ApiErrorResponse {
  message?: string;
  error?: string;
}

/**
 * Extract error message from various error types
 * Handles Axios errors, standard errors, and unknown values
 */
export function getErrorMessage(
  error: unknown,
  fallback = 'An error occurred'
): string {
  if (error instanceof AxiosError) {
    const data = error.response?.data as ApiErrorResponse | undefined;
    return data?.message || data?.error || error.message || fallback;
  }

  if (error instanceof Error) {
    return error.message || fallback;
  }

  if (typeof error === 'string') {
    return error;
  }

  return fallback;
}

/**
 * Type guard for checking if error is an AxiosError
 */
export function isAxiosError(error: unknown): error is AxiosError {
  return error instanceof AxiosError;
}

/**
 * Type guard for checking if error is an AxiosError with a specific status code
 */
export function isAxiosErrorWithStatus(
  error: unknown,
  status: number
): error is AxiosError {
  return isAxiosError(error) && error.response?.status === status;
}

/**
 * Status, machine code and message of an API error response.
 * Read structurally so it works for AxiosError and for plain
 * `{ response: { status, data } }` shapes (e.g. test mocks).
 */
export interface ApiErrorInfo {
  status?: number;
  code?: string;
  message?: string;
}

export function getApiErrorInfo(error: unknown): ApiErrorInfo {
  if (typeof error !== 'object' || error === null || !('response' in error)) {
    return {};
  }
  const response: unknown = error.response;
  if (typeof response !== 'object' || response === null) return {};

  const info: ApiErrorInfo = {};
  if ('status' in response && typeof response.status === 'number') {
    info.status = response.status;
  }
  const data: unknown = 'data' in response ? response.data : undefined;
  if (typeof data === 'object' && data !== null) {
    if ('error' in data && typeof data.error === 'string') {
      info.code = data.error;
    }
    if ('message' in data && typeof data.message === 'string') {
      info.message = data.message;
    }
  }
  return info;
}

/** Shown for throttled requests (HTTP 429); the API's own wording is generic. */
export const TOO_MANY_ATTEMPTS_MESSAGE =
  'Too many attempts, try again shortly.';

/** Shown when the API refuses an organization action (HTTP 403). */
export const TENANT_PERMISSION_MESSAGE =
  "You don't have permission to do this in this organization. Ask an organization admin if you need access.";

/** Shown when the caller is not a member of the organization (HTTP 403). */
export const TENANT_ACCESS_DENIED_MESSAGE =
  "You don't have access to this organization.";

/**
 * Error message for organization (`/tenants/:id/*`) calls: a 403 from the
 * membership or permission middleware becomes a clear permission message
 * instead of a generic failure. Other 403s carry a specific code and human
 * `message` (e.g. `insufficient_role` from the invitation handler), and the
 * message is kept.
 */
export function getTenantErrorMessage(
  error: unknown,
  fallback = 'An error occurred'
): string {
  const { status, code, message } = getApiErrorInfo(error);
  if (status === 403) {
    if (code === 'tenant_access_denied') return TENANT_ACCESS_DENIED_MESSAGE;
    if (!code || code === 'forbidden') return TENANT_PERMISSION_MESSAGE;
    return message || code;
  }
  if (status === 429) return TOO_MANY_ATTEMPTS_MESSAGE;
  return message || code || getErrorMessage(error, fallback);
}
