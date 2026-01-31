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
} from '@app-types';

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
const CURRENT_TENANT_KEY = 'current_tenant_id';

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

// Response interceptor - handle token refresh
// Tokens are in httpOnly cookies, so refresh is handled automatically by the backend
api.interceptors.response.use(
  (response) => response,
  async (error: AxiosError<ApiError>) => {
    const originalRequest = error.config as InternalAxiosRequestConfig & {
      _retry?: boolean;
    };

    // If error is 401 and we haven't retried yet, try to refresh the token
    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true;

      try {
        // Attempt to refresh the token via cookie
        // The refresh token cookie is automatically sent with this request
        await axios.post('/api/v1/auth/refresh', {}, { withCredentials: true });

        // Retry the original request (new access token cookie is automatically set)
        return api(originalRequest);
      } catch (refreshError) {
        // Refresh failed - clear local auth state and redirect to login
        localStorage.removeItem('auth-storage'); // Clear zustand persisted auth state
        localStorage.removeItem('tenant-storage'); // Clear tenant state
        localStorage.removeItem(CURRENT_TENANT_KEY);
        window.location.href = '/login';
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
    api.get(`/v1/auth/verify?token=${data.token}`),

  resendVerification: (data: ResendVerificationRequest) =>
    api.post('/v1/auth/resend-verification', data),
};

// User API endpoints
export const userApi = {
  getCurrentUser: () => api.get<User>('/v1/users/me'),
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  updateUser: (id: string, data: any) => api.put(`/v1/users/${id}`, data),
};

export default api;
