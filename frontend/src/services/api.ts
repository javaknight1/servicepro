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
  UpdateUserRequest,
  ApiError,
  User,
} from '@app-types';
import { clearQueryCaches } from './queryCacheRegistry';

// Create axios instance
const api: AxiosInstance = axios.create({
  baseURL: '/api',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
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

// Request interceptor - add tenant ID to requests
// Note: Auth tokens are handled via httpOnly cookies (automatically sent with withCredentials: true)
api.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    // Add tenant ID header if available
    const tenantId = localStorage.getItem(CURRENT_TENANT_KEY);
    if (tenantId && config.headers) {
      config.headers['X-Tenant-ID'] = tenantId;
    }

    return config;
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
      .post('/api/v1/auth/refresh', {}, { withCredentials: true })
      .finally(() => {
        refreshPromise = null;
      });
  }
  return refreshPromise;
}

// Response interceptor - handle token refresh
// Tokens are in httpOnly cookies, so refresh is handled automatically by the backend
api.interceptors.response.use(
  (response) => response,
  async (error: AxiosError<ApiError>) => {
    const originalRequest = error.config as
      | (InternalAxiosRequestConfig & { _retry?: boolean })
      | undefined;

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
        localStorage.removeItem('auth-storage'); // Clear zustand persisted auth state
        localStorage.removeItem('tenant-storage'); // Clear tenant state
        localStorage.removeItem(CURRENT_TENANT_KEY);
        clearTenantHeader(api);
        clearQueryCaches(); // Drop cached server data (memory + persisted)
        if (window.location.pathname !== '/login') {
          window.location.href = '/login';
        }
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
};

// User API endpoints
export const userApi = {
  getCurrentUser: () => api.get<User>('/v1/users/me'),
  updateUser: (id: string, data: UpdateUserRequest) =>
    api.put<User>(`/v1/users/${id}`, data),
};

export default api;
