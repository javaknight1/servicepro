import { describe, it, expect } from 'vitest';
import { AxiosError } from 'axios';
import {
  getApiErrorInfo,
  getErrorMessage,
  getTenantErrorMessage,
  isAxiosError,
  isAxiosErrorWithStatus,
  TENANT_ACCESS_DENIED_MESSAGE,
  TENANT_PERMISSION_MESSAGE,
  TOO_MANY_ATTEMPTS_MESSAGE,
} from '../error';

describe('error utilities', () => {
  describe('getErrorMessage', () => {
    it('should extract message from AxiosError with response data message', () => {
      const error = new AxiosError('Request failed');
      error.response = {
        data: { message: 'User not found' },
        status: 404,
        statusText: 'Not Found',
        headers: {},
        config: {} as never,
      };

      expect(getErrorMessage(error)).toBe('User not found');
    });

    it('should extract error from AxiosError with response data error field', () => {
      const error = new AxiosError('Request failed');
      error.response = {
        data: { error: 'Invalid credentials' },
        status: 401,
        statusText: 'Unauthorized',
        headers: {},
        config: {} as never,
      };

      expect(getErrorMessage(error)).toBe('Invalid credentials');
    });

    it('should fall back to error.message for AxiosError without response data', () => {
      const error = new AxiosError('Network Error');

      expect(getErrorMessage(error)).toBe('Network Error');
    });

    it('should extract message from standard Error', () => {
      const error = new Error('Something went wrong');

      expect(getErrorMessage(error)).toBe('Something went wrong');
    });

    it('should return string errors directly', () => {
      const error = 'An error occurred';

      expect(getErrorMessage(error)).toBe('An error occurred');
    });

    it('should return fallback for unknown error types', () => {
      expect(getErrorMessage(null)).toBe('An error occurred');
      expect(getErrorMessage(undefined)).toBe('An error occurred');
      expect(getErrorMessage(42)).toBe('An error occurred');
      expect(getErrorMessage({})).toBe('An error occurred');
    });

    it('should use custom fallback message', () => {
      expect(getErrorMessage(null, 'Custom fallback')).toBe('Custom fallback');
    });

    it('should handle AxiosError with empty response data', () => {
      const error = new AxiosError('Request failed');
      error.response = {
        data: {},
        status: 500,
        statusText: 'Internal Server Error',
        headers: {},
        config: {} as never,
      };

      expect(getErrorMessage(error)).toBe('Request failed');
    });

    it('should use fallback for AxiosError with empty message and no data', () => {
      const error = new AxiosError('');
      error.response = {
        data: {},
        status: 500,
        statusText: 'Internal Server Error',
        headers: {},
        config: {} as never,
      };

      expect(getErrorMessage(error, 'Fallback message')).toBe(
        'Fallback message'
      );
    });

    it('should use fallback for Error with empty message', () => {
      const error = new Error('');

      expect(getErrorMessage(error, 'Empty error fallback')).toBe(
        'Empty error fallback'
      );
    });
  });

  describe('isAxiosError', () => {
    it('should return true for AxiosError instances', () => {
      const error = new AxiosError('Test error');
      expect(isAxiosError(error)).toBe(true);
    });

    it('should return false for standard Error', () => {
      const error = new Error('Test error');
      expect(isAxiosError(error)).toBe(false);
    });

    it('should return false for non-error types', () => {
      expect(isAxiosError(null)).toBe(false);
      expect(isAxiosError(undefined)).toBe(false);
      expect(isAxiosError('string error')).toBe(false);
      expect(isAxiosError(42)).toBe(false);
      expect(isAxiosError({})).toBe(false);
    });
  });

  describe('isAxiosErrorWithStatus', () => {
    it('should return true for AxiosError with matching status', () => {
      const error = new AxiosError('Not found');
      error.response = {
        data: {},
        status: 404,
        statusText: 'Not Found',
        headers: {},
        config: {} as never,
      };

      expect(isAxiosErrorWithStatus(error, 404)).toBe(true);
    });

    it('should return false for AxiosError with different status', () => {
      const error = new AxiosError('Server error');
      error.response = {
        data: {},
        status: 500,
        statusText: 'Internal Server Error',
        headers: {},
        config: {} as never,
      };

      expect(isAxiosErrorWithStatus(error, 404)).toBe(false);
    });

    it('should return false for AxiosError without response', () => {
      const error = new AxiosError('Network error');
      expect(isAxiosErrorWithStatus(error, 404)).toBe(false);
    });

    it('should return false for non-AxiosError', () => {
      const error = new Error('Test error');
      expect(isAxiosErrorWithStatus(error, 500)).toBe(false);
    });
  });
});

describe('API error helpers', () => {
  const apiError = (status: number, data: Record<string, string>) => ({
    response: { status, data },
  });

  it('getApiErrorInfo reads status, code and message structurally', () => {
    expect(
      getApiErrorInfo(apiError(403, { error: 'forbidden', message: 'No' }))
    ).toEqual({ status: 403, code: 'forbidden', message: 'No' });
    expect(getApiErrorInfo(new Error('x'))).toEqual({});
    expect(getApiErrorInfo(null)).toEqual({});
  });

  it('getTenantErrorMessage maps 403s to permission messages', () => {
    expect(
      getTenantErrorMessage(
        apiError(403, {
          error: 'forbidden',
          message: 'required: tenants.manage',
        })
      )
    ).toBe(TENANT_PERMISSION_MESSAGE);
    expect(
      getTenantErrorMessage(apiError(403, { error: 'tenant_access_denied' }))
    ).toBe(TENANT_ACCESS_DENIED_MESSAGE);
    expect(getTenantErrorMessage(apiError(429, {}))).toBe(
      TOO_MANY_ATTEMPTS_MESSAGE
    );
    expect(
      getTenantErrorMessage(apiError(400, { message: 'Bad slug' }), 'x')
    ).toBe('Bad slug');
    expect(getTenantErrorMessage({}, 'fallback')).toBe('fallback');
  });
});
