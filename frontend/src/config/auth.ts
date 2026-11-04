/**
 * Which sign-in the SPA uses (T158), fixed at build time.
 *
 * - VITE_NEON_AUTH_URL unset (today's production): the API's own sign-in —
 *   POST /v1/auth/login sets httpOnly cookies and axios sends them.
 * - VITE_NEON_AUTH_URL set to a Neon Auth URL (e.g.
 *   https://ep-….neonauth.c-7.us-east-2.aws.neon.tech/neondb/auth): Neon's
 *   managed Better Auth. The browser signs in against that URL, and every API
 *   request carries a short-lived Neon Auth JWT as `Authorization: Bearer`.
 *   The API must have NEON_AUTH_BASE_URL set to the same URL.
 */

/** Normalise the configured Neon Auth URL ("" when Neon Auth is off). */
export function resolveNeonAuthUrl(value: string | undefined): string {
  return (value ?? '').trim().replace(/\/+$/, '');
}

export const NEON_AUTH_URL = resolveNeonAuthUrl(
  import.meta.env.VITE_NEON_AUTH_URL
);

/** True when the SPA signs in with Neon Auth instead of the API's own JWTs. */
export const isNeonAuthEnabled = NEON_AUTH_URL !== '';

/**
 * Show "Continue with Google" (Neon Auth only). Set VITE_NEON_AUTH_GOOGLE=true
 * only when the Neon branch has the Google provider configured; production
 * needs its own Google OAuth credentials (Neon's shared ones are for
 * development).
 */
export const isGoogleSignInEnabled =
  isNeonAuthEnabled && import.meta.env.VITE_NEON_AUTH_GOOGLE === 'true';
