/**
 * User-facing messages for the Neon Auth sign-in flows (T158): errors from
 * Neon Auth itself and from POST /auth/neon/provision.
 */
import { getAuthErrorInfo } from '@services/authErrors';
import { TOO_MANY_ATTEMPTS_MESSAGE } from './error';

export const INVALID_CREDENTIALS_MESSAGE = 'Invalid email or password.';

export const NO_ACCOUNT_MESSAGE =
  "There's no ServicePro account for this email yet. ServicePro is invite-only: ask your company's admin for an invitation, or create an account with your invite code.";

export const INVALID_INVITE_CODE_MESSAGE =
  "That invite code isn't valid. It may have expired, been used up or been revoked. Check the code or ask the person who invited you for a new one.";

export const INVALID_INVITATION_MESSAGE =
  'This invitation is no longer valid for this email address. Ask the person who invited you to send a new invitation.';

const MESSAGES_BY_CODE: Record<string, string> = {
  // POST /auth/neon/provision
  registration_closed: NO_ACCOUNT_MESSAGE,
  invalid_invite_code: INVALID_INVITE_CODE_MESSAGE,
  invalid_invitation: INVALID_INVITATION_MESSAGE,
  identity_conflict:
    'This email is already linked to a different sign-in. Contact support to sort it out.',
  terms_not_accepted:
    'Please agree to the Terms of Service and Privacy Policy to create an account.',
  email_not_verified: 'Verify your email address, then try again.',
  // Neon Auth (normalised codes from @neondatabase/auth)
  invalid_credentials: INVALID_CREDENTIALS_MESSAGE,
  user_already_exists:
    'You already have a sign-in for this email. Sign in instead, or reset your password.',
  email_exists:
    'You already have a sign-in for this email. Sign in instead, or reset your password.',
  weak_password: 'Choose a longer password (at least 8 characters).',
  over_request_rate_limit: TOO_MANY_ATTEMPTS_MESSAGE,
  not_signed_in: 'Your sign-in has expired. Please sign in again.',
};

/** Query param on /login when Google (via Neon Auth) sends the user back. */
export const OAUTH_CALLBACK_PARAM = 'oauth';

/** /verify-email for this address (Neon Auth: code entry). */
export function verifyEmailPath(
  email: string,
  returnUrl?: string | null
): string {
  const params = new URLSearchParams({ email });
  if (returnUrl) params.set('returnUrl', returnUrl);
  return `/verify-email?${params.toString()}`;
}

/** Codes that mean "verify the email address first". */
export function needsEmailVerification(err: unknown): boolean {
  const { code } = getAuthErrorInfo(err);
  return code === 'email_not_verified' || code === 'email_not_confirmed';
}

/** A message for a failed Neon Auth step, or `fallback`. */
export function getNeonAuthErrorMessage(err: unknown, fallback: string): string {
  const { status, code, message } = getAuthErrorInfo(err);
  if (code && MESSAGES_BY_CODE[code]) return MESSAGES_BY_CODE[code];
  if (status === 429) return TOO_MANY_ATTEMPTS_MESSAGE;
  return message || fallback;
}
