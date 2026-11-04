/**
 * Errors from the Neon Auth sign-in flows (T158), kept apart from
 * neonAuth.ts so pages can recognise them without loading the auth client.
 */
import { getApiErrorInfo, type ApiErrorInfo } from '@/utils/error';

/**
 * A failed Neon Auth step. `code` is either the Neon Auth SDK's normalised
 * code (e.g. "invalid_credentials", "user_already_exists") or one of ours:
 * "not_signed_in" when no Neon session exists.
 */
export class NeonAuthError extends Error {
  readonly code: string;
  readonly status?: number;

  constructor(message: string, code: string, status?: number) {
    super(message);
    this.name = 'NeonAuthError';
    this.code = code;
    this.status = status;
  }
}

export function isNeonAuthError(err: unknown): err is NeonAuthError {
  return err instanceof NeonAuthError;
}

/**
 * Status / code / message of a failed auth call, whether it came from the
 * API (axios) or from Neon Auth.
 */
export function getAuthErrorInfo(err: unknown): ApiErrorInfo {
  if (isNeonAuthError(err)) {
    return { status: err.status, code: err.code, message: err.message };
  }
  return getApiErrorInfo(err);
}
