import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import type { NeonAuthClient, NeonSessionData } from '../neonAuth';

vi.mock('@config/auth', () => ({
  NEON_AUTH_URL: 'https://auth.example.test/neondb/auth',
  isNeonAuthEnabled: true,
  isGoogleSignInEnabled: false,
}));

const ok = <T>(data: T) => Promise.resolve({ data, error: null });

const client = {
  signIn: { email: vi.fn(), social: vi.fn() },
  signUp: { email: vi.fn() },
  signOut: vi.fn(),
  getSession: vi.fn(),
  emailOtp: {
    sendVerificationOtp: vi.fn(),
    verifyEmail: vi.fn(),
    requestPasswordReset: vi.fn(),
    resetPassword: vi.fn(),
  },
};
const createAuthClient = vi.fn((): NeonAuthClient => client);

vi.mock('@neondatabase/auth', () => ({ createAuthClient }));
vi.mock('@neondatabase/auth/vanilla/adapters', () => ({
  BetterAuthVanillaAdapter: () => 'vanilla-adapter',
}));

import * as neon from '../neonAuth';
import { NeonAuthError } from '../authErrors';

/** A JWT-shaped token expiring `secondsFromNow` from now. */
function jwt(secondsFromNow: number, tag = 'a'): string {
  const b64 = (o: object) =>
    btoa(JSON.stringify(o))
      .replace(/\+/g, '-')
      .replace(/\//g, '_')
      .replace(/=+$/, '');
  const exp = Math.floor(Date.now() / 1000) + secondsFromNow;
  return `${b64({ alg: 'EdDSA' })}.${b64({ exp, sub: tag })}.sig`;
}

function session(token: string): NeonSessionData {
  return {
    session: { token },
    user: { id: 'u1', email: 'a@example.com', emailVerified: true },
  };
}

describe('neonAuth', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    neon.resetNeonAuthForTests();
    sessionStorage.clear();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('creates the client for the configured Neon Auth URL with the vanilla adapter', async () => {
    await neon.getNeonAuthClient();
    expect(createAuthClient).toHaveBeenCalledWith(
      'https://auth.example.test/neondb/auth',
      { adapter: 'vanilla-adapter' }
    );
  });

  describe('getNeonAccessToken', () => {
    it('returns the JWT and reuses it while it is fresh', async () => {
      const token = jwt(900);
      client.getSession.mockReturnValue(ok(session(token)));

      expect(await neon.getNeonAccessToken()).toBe(token);
      expect(await neon.getNeonAccessToken()).toBe(token);
      expect(client.getSession).toHaveBeenCalledTimes(1);
      // Always bypasses the SDK's own session cache.
      expect(client.getSession).toHaveBeenCalledWith({
        fetchOptions: { headers: { 'X-Force-Fetch': 'true' } },
      });
    });

    it('refreshes a minute before the JWT expires', async () => {
      client.getSession
        .mockReturnValueOnce(ok(session(jwt(30, 'old'))))
        .mockReturnValueOnce(ok(session(jwt(900, 'new'))));

      const first = await neon.getNeonAccessToken();
      const second = await neon.getNeonAccessToken();
      expect(second).not.toBe(first);
      expect(client.getSession).toHaveBeenCalledTimes(2);
    });

    it('shares one request between concurrent callers', async () => {
      client.getSession.mockReturnValue(ok(session(jwt(900))));
      const [a, b] = await Promise.all([
        neon.getNeonAccessToken(),
        neon.getNeonAccessToken(),
      ]);
      expect(a).toBe(b);
      expect(client.getSession).toHaveBeenCalledTimes(1);
    });

    it('forceRefresh always asks Neon Auth', async () => {
      client.getSession.mockReturnValue(ok(session(jwt(900))));
      await neon.getNeonAccessToken();
      await neon.getNeonAccessToken({ forceRefresh: true });
      expect(client.getSession).toHaveBeenCalledTimes(2);
    });

    it('returns null without a session and remembers that until forced', async () => {
      client.getSession.mockReturnValue(ok(null));
      expect(await neon.getNeonAccessToken()).toBeNull();
      expect(await neon.getNeonAccessToken()).toBeNull();
      expect(client.getSession).toHaveBeenCalledTimes(1);

      client.getSession.mockReturnValue(ok(session(jwt(900))));
      expect(await neon.getNeonAccessToken({ forceRefresh: true })).not.toBeNull();
    });

    it('never hands out a session token that is not a JWT', async () => {
      client.getSession.mockReturnValue(ok(session('opaque-session-token')));
      expect(await neon.getNeonAccessToken()).toBeNull();
    });

    it('keeps the JWT out of web storage', async () => {
      const token = jwt(900);
      client.getSession.mockReturnValue(ok(session(token)));
      await neon.getNeonAccessToken();
      const stored =
        JSON.stringify({ ...localStorage }) + JSON.stringify({ ...sessionStorage });
      expect(stored).not.toContain(token);
    });
  });

  describe('flows', () => {
    it('signInWithEmail throws a NeonAuthError carrying status and code', async () => {
      client.signIn.email.mockReturnValue(
        Promise.resolve({
          data: null,
          error: { message: 'Invalid email or password', status: 401, code: 'invalid_credentials' },
        })
      );
      const err = await neon
        .signInWithEmail('a@example.com', 'wrong-password')
        .catch((e: unknown) => e);
      expect(err).toBeInstanceOf(NeonAuthError);
      expect(err).toMatchObject({ status: 401, code: 'invalid_credentials' });
    });

    it('wraps errors the SDK throws', async () => {
      client.signUp.email.mockRejectedValue({ message: 'boom', status: 500 });
      await expect(
        neon.signUpWithEmail('a@example.com', 'password123')
      ).rejects.toMatchObject({ code: 'unknown_error', status: 500 });
    });

    it('signs up with the email local part as the display name', async () => {
      client.signUp.email.mockReturnValue(ok({}));
      await neon.signUpWithEmail('pat@example.com', 'password123');
      expect(client.signUp.email).toHaveBeenCalledWith({
        email: 'pat@example.com',
        password: 'password123',
        name: 'pat',
      });
    });

    it('verifyEmailCode drops the cached JWT (its emailVerified claim is stale)', async () => {
      client.getSession.mockReturnValue(ok(session(jwt(900))));
      await neon.getNeonAccessToken();
      client.emailOtp.verifyEmail.mockReturnValue(ok({ status: true }));

      await neon.verifyEmailCode('a@example.com', '123456');
      await neon.getNeonAccessToken();
      expect(client.getSession).toHaveBeenCalledTimes(2);
    });

    it('sends email-verification and password-reset codes', async () => {
      client.emailOtp.sendVerificationOtp.mockReturnValue(ok({}));
      client.emailOtp.requestPasswordReset.mockReturnValue(ok({}));
      client.emailOtp.resetPassword.mockReturnValue(ok({}));
      await neon.sendVerificationCode('a@example.com');
      await neon.requestPasswordResetCode('a@example.com');
      await neon.resetPasswordWithCode('a@example.com', '123456', 'new-password1');
      expect(client.emailOtp.sendVerificationOtp).toHaveBeenCalledWith({
        email: 'a@example.com',
        type: 'email-verification',
      });
      expect(client.emailOtp.requestPasswordReset).toHaveBeenCalledWith({
        email: 'a@example.com',
      });
      expect(client.emailOtp.resetPassword).toHaveBeenCalledWith({
        email: 'a@example.com',
        otp: '123456',
        password: 'new-password1',
      });
    });

    it('starts Google sign-in with our callback URLs', async () => {
      client.signIn.social.mockReturnValue(ok({}));
      await neon.signInWithGoogle('https://app/login?oauth=1', 'https://app/login?oauth=error');
      expect(client.signIn.social).toHaveBeenCalledWith({
        provider: 'google',
        callbackURL: 'https://app/login?oauth=1',
        errorCallbackURL: 'https://app/login?oauth=error',
      });
    });

    it('neonSignOut forgets the token even when Neon Auth fails', async () => {
      client.getSession.mockReturnValue(ok(session(jwt(900))));
      await neon.getNeonAccessToken();
      client.signOut.mockRejectedValue(new Error('offline'));

      await expect(neon.neonSignOut()).rejects.toThrow('offline');
      expect(await neon.getNeonAccessToken()).toBeNull();
    });
  });

  describe('pending sign-up', () => {
    it('round-trips through sessionStorage and clears', () => {
      neon.savePendingSignup({
        email: 'a@example.com',
        inviteCode: 'ABC-123',
        acceptTerms: true,
      });
      expect(neon.readPendingSignup()).toEqual({
        email: 'a@example.com',
        inviteCode: 'ABC-123',
        invitationToken: undefined,
        acceptTerms: true,
      });
      neon.clearPendingSignup();
      expect(neon.readPendingSignup()).toBeNull();
    });

    it('ignores malformed data', () => {
      sessionStorage.setItem('sp-neon-pending-signup', '{not json');
      expect(neon.readPendingSignup()).toBeNull();
      sessionStorage.setItem(
        'sp-neon-pending-signup',
        JSON.stringify({ acceptTerms: 'yes', inviteCode: 5 })
      );
      expect(neon.readPendingSignup()).toEqual({
        email: undefined,
        inviteCode: undefined,
        invitationToken: undefined,
        acceptTerms: false,
      });
    });
  });
});
