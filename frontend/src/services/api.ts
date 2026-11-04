import axios, {
  AxiosError,
  AxiosInstance,
  InternalAxiosRequestConfig,
} from 'axios';
import type {
  LoginRequest,
  RegisterRequest,
  RegisterResponse,
  PasswordResetRequestRequest,
  PasswordResetConfirmRequest,
  VerifyEmailRequest,
  ResendVerificationRequest,
  ApiError,
  User,
  NeonProvisionRequest,
  NeonProvisionResponse,
} from '@app-types';
import { API_BASE_URL } from '@config/endpoints';
import { isNeonAuthEnabled } from '@config/auth';
import { clearQueryCaches } from './queryCacheRegistry';
import { addBreadcrumb, setTag } from './errorTracking';
import { getApiErrorInfo } from '@/utils/error';

// Create axios instance. Defaults to same-origin "/api" (Workers proxy in
// production, Vite dev proxy locally); see src/config/endpoints.ts.
const api: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
    // CSRF: the backend rejects cookie-authenticated mutations without it.
    'X-Requested-With': 'XMLHttpRequest',
  },
  withCredentials: true, // Required for httpOnly cookies to be sent with requests
});

// Storage key for current tenant (matches tenantStore)
export const CURRENT_TENANT_KEY = 'current_tenant_id';

/** Remove any X-Tenant-ID default header from an axios instance. */
export function clearTenantHeader(instance: AxiosInstance): void {
  const common = instance.defaults?.headers?.common;
  if (common && 'X-Tenant-ID' in common) {
    delete common['X-Tenant-ID'];
  }
}

/** The Neon Auth module, loaded only when VITE_NEON_AUTH_URL is set (T158). */
const loadNeonAuth = () => import('./neonAuth');

// Request interceptor - add tenant ID (and, on Neon Auth, the bearer token)
// Cookie sign-in: auth tokens are httpOnly cookies (sent with withCredentials).
// Neon Auth sign-in: a short-lived JWT in memory, sent as Authorization: Bearer.
async function attachNeonToken(
  config: InternalAxiosRequestConfig
): Promise<InternalAxiosRequestConfig> {
  if (config.headers && !config.headers.Authorization) {
    try {
      const { getNeonAccessToken } = await loadNeonAuth();
      const token = await getNeonAccessToken();
      if (token) config.headers.Authorization = `Bearer ${token}`;
    } catch {
      // Neon Auth unreachable: send the request without a token; a
      // protected endpoint answers 401 and the response handler retries.
    }
  }
  return config;
}

api.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    // Add tenant ID header if available
    const tenantId = localStorage.getItem(CURRENT_TENANT_KEY);
    if (tenantId && config.headers) {
      config.headers['X-Tenant-ID'] = tenantId;
    }

    return isNeonAuthEnabled ? attachNeonToken(config) : config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

/**
 * Auth endpoints that must never trigger the refresh-and-redirect flow.
 * A 401 from these means "bad credentials / invalid token", and the caller
 * (e.g. the login page) needs to receive the error to display it.
 */
const AUTH_ENDPOINT_PATTERNS: RegExp[] = [
  /\/v1\/auth\/login\b/,
  /\/v1\/auth\/register\b/,
  /\/v1\/auth\/refresh\b/,
  /\/v1\/auth\/logout\b/,
  /\/v1\/auth\/reset-request\b/,
  /\/v1\/auth\/reset-password\b/,
  /\/v1\/auth\/forgot-password\b/,
  /\/v1\/auth\/verify\b/,
  /\/v1\/auth\/resend-verification\b/,
  /\/v1\/auth\/neon\/provision\b/,
];

export function isAuthEndpoint(url: string | undefined): boolean {
  if (!url) return false;
  return AUTH_ENDPOINT_PATTERNS.some((pattern) => pattern.test(url));
}

// Single-flight refresh: concurrent 401s share one refresh request.
let refreshPromise: Promise<unknown> | null = null;

function refreshSession(): Promise<unknown> {
  if (!refreshPromise) {
    refreshPromise = axios
      .post(
        `${API_BASE_URL}/v1/auth/refresh`,
        {},
        {
          withCredentials: true,
          headers: { 'X-Requested-With': 'XMLHttpRequest' },
        }
      )
      .finally(() => {
        refreshPromise = null;
      });
  }
  return refreshPromise;
}

/**
 * Server errors (5xx) leave their request ID in Sentry (T223): a
 * breadcrumb on any later event, plus a `request_id` tag, so a reported
 * error can be matched to the API's log lines. Only the path is recorded
 * (no query string, which may carry tokens).
 */
function recordServerError(error: AxiosError<ApiError>): void {
  const { status, requestId, code } = getApiErrorInfo(error);
  if (status === undefined || status < 500 || !requestId) return;
  const path = (error.config?.url ?? '').split('?')[0];
  setTag('request_id', requestId);
  addBreadcrumb({
    category: 'api',
    level: 'error',
    message: `API ${status} ${code ?? ''}`.trim(),
    data: { request_id: requestId, status, path },
  });
}

/**
 * The session is gone: clear local auth/tenant state and cached server data,
 * then send the user to the login page.
 */
function expireSession(): void {
  localStorage.removeItem('auth-storage'); // Clear zustand persisted auth state
  localStorage.removeItem('tenant-storage'); // Clear tenant state
  localStorage.removeItem(CURRENT_TENANT_KEY);
  clearTenantHeader(api);
  clearQueryCaches(); // Drop cached server data (memory + persisted)
  if (window.location.pathname !== '/login') {
    window.location.href = '/login';
  }
}

type RetryableRequest = InternalAxiosRequestConfig & { _retry?: boolean };

/**
 * Neon Auth 401s: retry once with a freshly minted JWT; if Neon Auth has no
 * session any more, the user is signed out. `account_not_linked` is left to
 * the sign-in flow (it calls POST /auth/neon/provision).
 */
async function handleNeonUnauthorized(
  error: AxiosError<ApiError>,
  originalRequest: RetryableRequest | undefined
): Promise<unknown> {
  if (
    !originalRequest ||
    isAuthEndpoint(originalRequest.url) ||
    getApiErrorInfo(error).code === 'account_not_linked'
  ) {
    return Promise.reject(error);
  }
  const neon = await loadNeonAuth();
  if (!originalRequest._retry) {
    originalRequest._retry = true;
    const token = await neon.getNeonAccessToken({ forceRefresh: true });
    if (token) {
      originalRequest.headers.Authorization = `Bearer ${token}`;
      return api(originalRequest);
    }
  }
  neon.clearNeonAccessToken();
  expireSession();
  return Promise.reject(error);
}

// Response interceptor - handle token refresh
// Cookie sign-in: tokens are in httpOnly cookies, refreshed via /auth/refresh.
api.interceptors.response.use(
  (response) => response,
  async (error: AxiosError<ApiError>) => {
    recordServerError(error);

    const originalRequest = error.config as RetryableRequest | undefined;

    if (isNeonAuthEnabled) {
      if (error.response?.status === 401) {
        return handleNeonUnauthorized(error, originalRequest);
      }
      return Promise.reject(error);
    }

    // If error is 401 and we haven't retried yet, try to refresh the token.
    // Auth endpoints are excluded so their errors reach the caller
    // (e.g. "invalid credentials" on the login page).
    if (
      error.response?.status === 401 &&
      originalRequest &&
      !originalRequest._retry &&
      !isAuthEndpoint(originalRequest.url)
    ) {
      originalRequest._retry = true;

      try {
        // Attempt to refresh the token via cookie
        // The refresh token cookie is automatically sent with this request
        await refreshSession();

        // Retry the original request (new access token cookie is automatically set)
        return api(originalRequest);
      } catch (refreshError) {
        // Refresh failed - clear local auth state and redirect to login
        expireSession();
        return Promise.reject(refreshError);
      }
    }

    return Promise.reject(error);
  }
);

// Login response type for cookie-based auth (no tokens in body)
interface LoginCookieResponse {
  message: string;
  expiresIn: number;
}

// Auth API endpoints
export const authApi = {
  login: (data: LoginRequest) =>
    api.post<LoginCookieResponse>('/v1/auth/login', data),

  register: (data: RegisterRequest) =>
    api.post<RegisterResponse>('/v1/auth/register', data),

  refreshToken: () => api.post<LoginCookieResponse>('/v1/auth/refresh', {}),

  logout: () => api.post('/v1/auth/logout', {}),

  requestPasswordReset: (data: PasswordResetRequestRequest) =>
    api.post('/v1/auth/reset-request', data),

  confirmPasswordReset: (data: PasswordResetConfirmRequest) =>
    api.post('/v1/auth/reset-password', data),

  verifyEmail: (data: VerifyEmailRequest) =>
    api.get('/v1/auth/verify', { params: { token: data.token } }),

  resendVerification: (data: ResendVerificationRequest) =>
    api.post('/v1/auth/resend-verification', data),

  /** Link or create the ServicePro user for a Neon Auth sign-in (T158). */
  provisionNeonAccount: (data: NeonProvisionRequest) =>
    api.post<NeonProvisionResponse>('/v1/auth/neon/provision', data),
};

// User API endpoints
export const userApi = {
  getCurrentUser: () => api.get<User>('/v1/users/me'),
};

export default api;
