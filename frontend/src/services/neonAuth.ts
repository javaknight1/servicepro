/**
 * Neon Auth (managed Better Auth) client for the SPA (T158). Used only when
 * VITE_NEON_AUTH_URL is set (src/config/auth.ts); callers load this module
 * with a dynamic import so the cookie-auth build never ships it.
 *
 * Where the session lives (proposed decision D17): the Better Auth session is
 * an httpOnly, Secure, SameSite=None; Partitioned cookie on the Neon Auth host,
 * which this SPA never reads. The API is called with the 15-minute EdDSA JWT
 * the session yields; that JWT is kept in memory only (this module), never in
 * localStorage/sessionStorage, and is refreshed a minute before it expires.
 */
import { NEON_AUTH_URL } from '@config/auth';
import { NeonAuthError } from './authErrors';

// ---------------------------------------------------------------------------
// The slice of the Better Auth client we use (structural, so tests can fake it)
// ---------------------------------------------------------------------------

interface AuthCallError {
  message?: string;
  status?: number;
  code?: string;
}

interface AuthResult<T> {
  data: T | null;
  error: AuthCallError | null;
}

interface FetchOptionsArg {
  fetchOptions?: { headers?: Record<string, string> };
}

export interface NeonSessionData {
  session: { token: string };
  user: { id: string; email: string; emailVerified: boolean };
}

export interface NeonAuthClient {
  signIn: {
    email: (args: {
      email: string;
      password: string;
    }) => Promise<AuthResult<unknown>>;
    social: (args: {
      provider: 'google';
      callbackURL: string;
      errorCallbackURL?: string;
      newUserCallbackURL?: string;
    }) => Promise<AuthResult<unknown>>;
  };
  signUp: {
    email: (args: {
      email: string;
      password: string;
      name: string;
    }) => Promise<AuthResult<unknown>>;
  };
  signOut: () => Promise<AuthResult<unknown>>;
  getSession: (args?: FetchOptionsArg) => Promise<AuthResult<NeonSessionData>>;
  emailOtp: {
    sendVerificationOtp: (args: {
      email: string;
      type: 'email-verification' | 'forget-password';
    }) => Promise<AuthResult<unknown>>;
    verifyEmail: (args: {
      email: string;
      otp: string;
    }) => Promise<AuthResult<unknown>>;
    requestPasswordReset: (args: {
      email: string;
    }) => Promise<AuthResult<unknown>>;
    resetPassword: (args: {
      email: string;
      otp: string;
      password: string;
    }) => Promise<AuthResult<unknown>>;
  };
}

let clientPromise: Promise<NeonAuthClient> | null = null;

/** The Neon Auth client, created on first use. */
export function getNeonAuthClient(): Promise<NeonAuthClient> {
  if (!clientPromise) {
    clientPromise = Promise.all([
      import('@neondatabase/auth'),
      import('@neondatabase/auth/vanilla/adapters'),
    ]).then(
      ([{ createAuthClient }, { BetterAuthVanillaAdapter }]): NeonAuthClient =>
        createAuthClient(NEON_AUTH_URL, { adapter: BetterAuthVanillaAdapter() })
    );
  }
  return clientPromise;
}

/** Unwrap a Better Auth `{ data, error }` result, throwing NeonAuthError. */
async function call<T>(
  promise: Promise<AuthResult<T>>,
  fallback: string
): Promise<T | null> {
  let result: AuthResult<T>;
  try {
    result = await promise;
  } catch (err: unknown) {
    throw toNeonAuthError(err, fallback);
  }
  if (result.error) throw toNeonAuthError(result.error, fallback);
  return result.data;
}

function toNeonAuthError(err: unknown, fallback: string): NeonAuthError {
  if (err instanceof NeonAuthError) return err;
  if (typeof err === 'object' && err !== null) {
    const e = err as AuthCallError;
    return new NeonAuthError(
      typeof e.message === 'string' && e.message ? e.message : fallback,
      typeof e.code === 'string' && e.code ? e.code : 'unknown_error',
      typeof e.status === 'number' ? e.status : undefined
    );
  }
  return new NeonAuthError(fallback, 'unknown_error');
}

// ---------------------------------------------------------------------------
// API access token (Neon Auth JWT), memory only
// ---------------------------------------------------------------------------

/** Refresh the JWT this long before it expires (it lives 15 minutes). */
export const TOKEN_REFRESH_MARGIN_MS = 60_000;
/** Neon's client bypasses its own session cache when this header is set. */
const FORCE_FETCH_HEADER = 'X-Force-Fetch';

interface CachedToken {
  token: string;
  expiresAtMs: number;
}

let cachedToken: CachedToken | null = null;
/** True once a lookup found no Neon session (cleared by any sign-in step). */
let knownSignedOut = false;
let tokenRequest: Promise<string | null> | null = null;

function jwtExpiryMs(token: string): number | null {
  const parts = token.split('.');
  if (parts.length !== 3) return null;
  try {
    const json = atob(parts[1].replace(/-/g, '+').replace(/_/g, '/'));
    const payload: unknown = JSON.parse(json);
    if (
      typeof payload === 'object' &&
      payload !== null &&
      'exp' in payload &&
      typeof payload.exp === 'number'
    ) {
      return payload.exp * 1000;
    }
  } catch {
    return null;
  }
  return null;
}

async function fetchToken(): Promise<string | null> {
  const client = await getNeonAuthClient();
  const { data } = await client.getSession({
    fetchOptions: { headers: { [FORCE_FETCH_HEADER]: 'true' } },
  });
  // Neon's client swaps session.token for the JWT from the set-auth-jwt
  // response header; anything that is not a JWT is not usable as a bearer.
  const token = data?.session?.token ?? null;
  const expiresAtMs = token ? jwtExpiryMs(token) : null;
  if (!token || expiresAtMs === null) {
    cachedToken = null;
    knownSignedOut = true;
    return null;
  }
  cachedToken = { token, expiresAtMs };
  knownSignedOut = false;
  return token;
}

/**
 * The bearer token for the API, or null when not signed in to Neon Auth.
 * Reuses the cached JWT until a minute before it expires; `forceRefresh`
 * always asks Neon Auth (after a 401, or to re-check a session).
 */
export function getNeonAccessToken(
  options: { forceRefresh?: boolean } = {}
): Promise<string | null> {
  if (!options.forceRefresh) {
    if (
      cachedToken &&
      cachedToken.expiresAtMs - Date.now() > TOKEN_REFRESH_MARGIN_MS
    ) {
      return Promise.resolve(cachedToken.token);
    }
    if (!cachedToken && knownSignedOut) return Promise.resolve(null);
  }
  if (!tokenRequest) {
    tokenRequest = fetchToken().finally(() => {
      tokenRequest = null;
    });
  }
  return tokenRequest;
}

/** Forget the cached JWT (sign-out, or the session changed). */
export function clearNeonAccessToken(): void {
  cachedToken = null;
  knownSignedOut = false;
}

// ---------------------------------------------------------------------------
// Sign-in flows
// ---------------------------------------------------------------------------

export async function signInWithEmail(
  email: string,
  password: string
): Promise<void> {
  clearNeonAccessToken();
  const client = await getNeonAuthClient();
  await call(client.signIn.email({ email, password }), 'Failed to sign in.');
}

export async function signUpWithEmail(
  email: string,
  password: string
): Promise<void> {
  clearNeonAccessToken();
  const client = await getNeonAuthClient();
  // Better Auth requires a display name; ServicePro keeps names on its own
  // users row (onboarding), so the email's local part stands in.
  const name = email.split('@')[0] || email;
  await call(
    client.signUp.email({ email, password, name }),
    'Failed to create your sign-in.'
  );
}

/** Email a verification code (Neon Auth sends it). */
export async function sendVerificationCode(email: string): Promise<void> {
  const client = await getNeonAuthClient();
  await call(
    client.emailOtp.sendVerificationOtp({ email, type: 'email-verification' }),
    'Failed to send the verification code.'
  );
}

/** Verify the email with its code. Neon Auth signs the user in on success. */
export async function verifyEmailCode(
  email: string,
  otp: string
): Promise<void> {
  const client = await getNeonAuthClient();
  await call(
    client.emailOtp.verifyEmail({ email, otp }),
    'That code is not valid.'
  );
  // The JWT carries emailVerified, so a token minted before this is stale.
  clearNeonAccessToken();
}

/** Email a password-reset code. */
export async function requestPasswordResetCode(email: string): Promise<void> {
  const client = await getNeonAuthClient();
  await call(
    client.emailOtp.requestPasswordReset({ email }),
    'Failed to send the reset code.'
  );
}

export async function resetPasswordWithCode(
  email: string,
  otp: string,
  password: string
): Promise<void> {
  const client = await getNeonAuthClient();
  await call(
    client.emailOtp.resetPassword({ email, otp, password }),
    'Failed to reset your password.'
  );
}

/**
 * Start "Continue with Google": Neon Auth redirects to Google and back to
 * `callbackURL` (an origin on the branch's trusted-domain list).
 */
export async function signInWithGoogle(
  callbackURL: string,
  errorCallbackURL: string
): Promise<void> {
  clearNeonAccessToken();
  const client = await getNeonAuthClient();
  await call(
    client.signIn.social({ provider: 'google', callbackURL, errorCallbackURL }),
    'Failed to start Google sign-in.'
  );
}

export async function neonSignOut(): Promise<void> {
  clearNeonAccessToken();
  const client = await getNeonAuthClient();
  try {
    await client.signOut();
  } finally {
    clearNeonAccessToken();
    knownSignedOut = true;
  }
}

// ---------------------------------------------------------------------------
// Pending sign-up: what POST /auth/neon/provision needs after the email is
// verified (or after the Google redirect). Kept in sessionStorage so it
// survives that redirect in this tab only.
// ---------------------------------------------------------------------------

const PENDING_SIGNUP_KEY = 'sp-neon-pending-signup';

export interface PendingSignup {
  email?: string;
  inviteCode?: string;
  invitationToken?: string;
  acceptTerms: boolean;
}

export function savePendingSignup(pending: PendingSignup): void {
  try {
    sessionStorage.setItem(PENDING_SIGNUP_KEY, JSON.stringify(pending));
  } catch {
    // Storage blocked: provisioning then fails with a clear message.
  }
}

export function readPendingSignup(): PendingSignup | null {
  try {
    const raw = sessionStorage.getItem(PENDING_SIGNUP_KEY);
    if (!raw) return null;
    const parsed: unknown = JSON.parse(raw);
    if (typeof parsed !== 'object' || parsed === null) return null;
    const p = parsed as Record<string, unknown>;
    const str = (v: unknown) =>
      typeof v === 'string' && v !== '' ? v : undefined;
    return {
      email: str(p.email),
      inviteCode: str(p.inviteCode),
      invitationToken: str(p.invitationToken),
      acceptTerms: p.acceptTerms === true,
    };
  } catch {
    return null;
  }
}

export function clearPendingSignup(): void {
  try {
    sessionStorage.removeItem(PENDING_SIGNUP_KEY);
  } catch {
    // ignore
  }
}

/** Reset module state (tests only). */
export function resetNeonAuthForTests(): void {
  clientPromise = null;
  cachedToken = null;
  knownSignedOut = false;
  tokenRequest = null;
}
