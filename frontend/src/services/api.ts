import axios, {
  AxiosError,
  AxiosInstance,
  InternalAxiosRequestConfig,
} from 'axios';
import type {
  LoginRequest,
  LoginResponse,
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
});

// Storage key for current tenant (matches tenantStore)
const CURRENT_TENANT_KEY = 'current_tenant_id';

// Request interceptor - add auth token and tenant ID to requests
api.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const token = localStorage.getItem('access_token');
    if (token && config.headers) {
      config.headers.Authorization = `Bearer ${token}`;
    }

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
        const refreshToken = localStorage.getItem('refresh_token');
        if (!refreshToken) {
          throw new Error('No refresh token available');
        }

        // Attempt to refresh the token (backend expects snake_case)
        const response = await axios.post<LoginResponse>(
          '/api/v1/auth/refresh',
          {
            refresh_token: refreshToken,
          }
        );

        const { token, refreshToken: newRefreshToken } = response.data;

        // Save new tokens
        localStorage.setItem('access_token', token);
        localStorage.setItem('refresh_token', newRefreshToken);

        // Retry the original request with new token
        if (originalRequest.headers) {
          originalRequest.headers.Authorization = `Bearer ${token}`;
        }
        return api(originalRequest);
      } catch (refreshError) {
        // Refresh failed - clear ALL auth-related storage and redirect to login
        localStorage.removeItem('access_token');
        localStorage.removeItem('refresh_token');
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

// Auth API endpoints
export const authApi = {
  login: (data: LoginRequest) =>
    api.post<LoginResponse>('/v1/auth/login', data),

  register: (data: RegisterRequest) =>
    api.post<RegisterResponse>('/v1/auth/register', data),

  refreshToken: (refreshToken: string) =>
    api.post<LoginResponse>('/v1/auth/refresh', {
      refresh_token: refreshToken,
    }),

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
  updateUser: (id: string, data: any) => api.put(`/v1/users/${id}`, data),
};

export default api;
