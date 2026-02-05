import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import axios from 'axios';
import { authApi, userApi } from '../api';

// Mock axios.create to return a mock instance
vi.mock('axios', async () => {
  const mockAxiosInstance = {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
    interceptors: {
      request: { use: vi.fn() },
      response: { use: vi.fn() },
    },
  };

  return {
    default: {
      create: vi.fn(() => mockAxiosInstance),
      post: vi.fn(),
    },
    AxiosError: class AxiosError extends Error {},
  };
});

// Get the mock instance
const mockAxios = axios.create() as unknown as {
  get: ReturnType<typeof vi.fn>;
  post: ReturnType<typeof vi.fn>;
  put: ReturnType<typeof vi.fn>;
  delete: ReturnType<typeof vi.fn>;
};

describe('api service', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('authApi', () => {
    describe('login', () => {
      it('should call POST /v1/auth/login with credentials', async () => {
        const loginData = {
          email: 'test@example.com',
          password: 'password123',
        };
        const mockResponse = {
          data: { message: 'Login successful', expiresIn: 3600 },
        };
        mockAxios.post.mockResolvedValue(mockResponse);

        await authApi.login(loginData);

        expect(mockAxios.post).toHaveBeenCalledWith(
          '/v1/auth/login',
          loginData
        );
      });
    });

    describe('register', () => {
      it('should call POST /v1/auth/register with user data', async () => {
        const registerData = {
          email: 'new@example.com',
          password: 'password123',
          first_name: 'John',
          last_name: 'Doe',
        };
        const mockResponse = { data: { id: 'user-1' } };
        mockAxios.post.mockResolvedValue(mockResponse);

        await authApi.register(registerData);

        expect(mockAxios.post).toHaveBeenCalledWith(
          '/v1/auth/register',
          registerData
        );
      });
    });

    describe('refreshToken', () => {
      it('should call POST /v1/auth/refresh', async () => {
        const mockResponse = {
          data: { message: 'Token refreshed', expiresIn: 3600 },
        };
        mockAxios.post.mockResolvedValue(mockResponse);

        await authApi.refreshToken();

        expect(mockAxios.post).toHaveBeenCalledWith('/v1/auth/refresh', {});
      });
    });

    describe('logout', () => {
      it('should call POST /v1/auth/logout', async () => {
        mockAxios.post.mockResolvedValue({});

        await authApi.logout();

        expect(mockAxios.post).toHaveBeenCalledWith('/v1/auth/logout', {});
      });
    });

    describe('requestPasswordReset', () => {
      it('should call POST /v1/auth/reset-request with email', async () => {
        const resetData = { email: 'user@example.com' };
        mockAxios.post.mockResolvedValue({});

        await authApi.requestPasswordReset(resetData);

        expect(mockAxios.post).toHaveBeenCalledWith(
          '/v1/auth/reset-request',
          resetData
        );
      });
    });

    describe('confirmPasswordReset', () => {
      it('should call POST /v1/auth/reset-password with token and password', async () => {
        const resetData = { token: 'reset-token', password: 'newpassword123' };
        mockAxios.post.mockResolvedValue({});

        await authApi.confirmPasswordReset(resetData);

        expect(mockAxios.post).toHaveBeenCalledWith(
          '/v1/auth/reset-password',
          resetData
        );
      });
    });

    describe('verifyEmail', () => {
      it('should call GET /v1/auth/verify with token', async () => {
        const verifyData = { token: 'verify-token' };
        mockAxios.get.mockResolvedValue({});

        await authApi.verifyEmail(verifyData);

        expect(mockAxios.get).toHaveBeenCalledWith(
          '/v1/auth/verify?token=verify-token'
        );
      });
    });

    describe('resendVerification', () => {
      it('should call POST /v1/auth/resend-verification with email', async () => {
        const resendData = { email: 'user@example.com' };
        mockAxios.post.mockResolvedValue({});

        await authApi.resendVerification(resendData);

        expect(mockAxios.post).toHaveBeenCalledWith(
          '/v1/auth/resend-verification',
          resendData
        );
      });
    });
  });

  describe('userApi', () => {
    describe('getCurrentUser', () => {
      it('should call GET /v1/users/me', async () => {
        const mockUser = { id: 'user-1', email: 'test@example.com' };
        mockAxios.get.mockResolvedValue({ data: mockUser });

        await userApi.getCurrentUser();

        expect(mockAxios.get).toHaveBeenCalledWith('/v1/users/me');
      });
    });

    describe('updateUser', () => {
      it('should call PUT /v1/users/:id with update data', async () => {
        const updateData = { first_name: 'Updated', last_name: 'Name' };
        const mockResponse = { data: { id: 'user-1', ...updateData } };
        mockAxios.put.mockResolvedValue(mockResponse);

        await userApi.updateUser('user-1', updateData);

        expect(mockAxios.put).toHaveBeenCalledWith(
          '/v1/users/user-1',
          updateData
        );
      });
    });
  });
});
