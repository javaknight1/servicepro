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
