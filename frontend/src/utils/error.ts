import { AxiosError } from 'axios';

/**
 * The API's error envelope (T223, docs/api/errors.md):
 * `{ error: code, message, request_id, fields?, details? }`.
 * Endpoints written before T223 may send only `error` (sometimes a
 * sentence) — every field is optional here.
 */
export interface ApiErrorResponse {
  /** Stable snake_case machine code (legacy: may be a sentence). */
  error?: string;
  /** Human-readable, safe to show. */
  message?: string;
  /** Matches the X-Request-ID header and the server's log lines. */
  request_id?: string;
  /** Validation errors: request field -> problem. */
  fields?: Record<string, string>;
  /** Extra messages, e.g. per-row import errors. */
  details?: string[];
}

/**
 * Appends "(Reference: <request id>)" to a message for server errors
 * (5xx), so a user's report can be matched to the server's log lines.
 * Other errors are returned unchanged: they are the user's to fix.
 */
export function withErrorReference(message: string, error: unknown): string {
  const { status, requestId } = getApiErrorInfo(error);
  if (!requestId || status === undefined || status < 500) return message;
  return `${message} (Reference: ${requestId})`;
}

/**
 * Extract error message from various error types
 * Handles Axios errors, standard errors, and unknown values.
 * Server errors (5xx) carry "(Reference: <request id>)".
 */
export function getErrorMessage(
  error: unknown,
  fallback = 'An error occurred'
): string {
  if (error instanceof AxiosError) {
    const data = error.response?.data as ApiErrorResponse | undefined;
    const message = data?.message || data?.error || error.message || fallback;
    return withErrorReference(message, error);
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
  /** From the envelope's request_id, else the X-Request-ID header. */
  requestId?: string;
  /** Validation errors: request field -> problem. */
  fields?: Record<string, string>;
}

function isStringRecord(value: unknown): value is Record<string, string> {
  return (
    typeof value === 'object' &&
    value !== null &&
    !Array.isArray(value) &&
    Object.values(value).every((v) => typeof v === 'string')
  );
}

/** X-Request-ID from axios response headers (plain object or AxiosHeaders). */
function headerRequestId(response: object): string | undefined {
  if (!('headers' in response)) return undefined;
  const headers: unknown = response.headers;
  if (typeof headers !== 'object' || headers === null) return undefined;
  const value: unknown =
    'x-request-id' in headers
      ? headers['x-request-id']
      : 'X-Request-ID' in headers
        ? headers['X-Request-ID']
        : undefined;
  return typeof value === 'string' && value !== '' ? value : undefined;
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
    if (
      'request_id' in data &&
      typeof data.request_id === 'string' &&
      data.request_id !== ''
    ) {
      info.requestId = data.request_id;
    }
    if ('fields' in data && isStringRecord(data.fields)) {
      info.fields = data.fields;
    }
  }
  info.requestId ??= headerRequestId(response);
  return info;
}

/** Per-field validation messages from an API error (empty if none). */
export function getFieldErrors(error: unknown): Record<string, string> {
  return getApiErrorInfo(error).fields ?? {};
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
  const apiText = message || code;
  return apiText
    ? withErrorReference(apiText, error)
    : getErrorMessage(error, fallback);
}
