/**
 * authStore on a Neon Auth build (VITE_NEON_AUTH_URL set, T158).
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { AxiosError, type AxiosResponse } from 'axios';

vi.mock('@config/auth', () => ({
  NEON_AUTH_URL: 'https://auth.example.test/neondb/auth',
  isNeonAuthEnabled: true,
  isGoogleSignInEnabled: false,
}));

const neon = vi.hoisted(() => ({
  getNeonAccessToken: vi.fn(),
  signInWithEmail: vi.fn(),
  signUpWithEmail: vi.fn(),
  sendVerificationCode: vi.fn(),
  neonSignOut: vi.fn(),
  savePendingSignup: vi.fn(),
  readPendingSignup: vi.fn(),
  clearPendingSignup: vi.fn(),
}));
vi.mock('@services/neonAuth', () => neon);

const { authApi, userApi, clearSessionState } = vi.hoisted(() => ({
  authApi: {
    login: vi.fn(),
    register: vi.fn(),
    logout: vi.fn(),
    refreshToken: vi.fn(),
    provisionNeonAccount: vi.fn(),
  },
  userApi: { getCurrentUser: vi.fn() },
  clearSessionState: vi.fn(),
}));
vi.mock('@services/api', () => ({ authApi, userApi }));
vi.mock('@services/errorTracking', () => ({ setUser: vi.fn() }));
vi.mock('@services/analytics', () => ({
  identifyUser: vi.fn(),
  resetUser: vi.fn(),
}));
vi.mock('@services/session', () => ({ clearSessionState }));
vi.mock('@services/queryCacheRegistry', () => ({ clearQueryCaches: vi.fn() }));

import { useAuthStore } from '../authStore';

const user = { id: 'user-1', email: 'pat@example.com' };

function apiError(status: number, code: string): AxiosError {
  const response = {
    status,
    data: { error: code, message: code },
    statusText: '',
    headers: {},
    config: {},
  } as unknown as AxiosResponse;
  return new AxiosError(code, String(status), undefined, null, response);
}

describe('authStore with Neon Auth', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useAuthStore.setState({ user: null, isAuthenticated: false, isLoading: false });
    neon.getNeonAccessToken.mockResolvedValue('jwt');
    neon.neonSignOut.mockResolvedValue(undefined);
    neon.readPendingSignup.mockReturnValue(null);
  });

  describe('login', () => {
    it('signs in with Neon Auth and loads a linked user (no provisioning)', async () => {
      userApi.getCurrentUser.mockResolvedValue({ data: user });

      await useAuthStore.getState().login('pat@example.com', 'password123');

      expect(neon.signInWithEmail).toHaveBeenCalledWith('pat@example.com', 'password123');
      expect(authApi.login).not.toHaveBeenCalled();
      expect(authApi.provisionNeonAccount).not.toHaveBeenCalled();
      expect(useAuthStore.getState()).toMatchObject({ user, isAuthenticated: true, isLoading: false });
    });

    it('provisions on account_not_linked with the saved invite and terms, then loads the user', async () => {
      userApi.getCurrentUser
        .mockRejectedValueOnce(apiError(401, 'account_not_linked'))
        .mockResolvedValueOnce({ data: user });
      neon.readPendingSignup.mockReturnValue({
        inviteCode: 'ABC-123',
        invitationToken: 'inv-1',
        acceptTerms: true,
      });
      authApi.provisionNeonAccount.mockResolvedValue({ data: { created: true } });

      await useAuthStore.getState().login('pat@example.com', 'password123');

      expect(authApi.provisionNeonAccount).toHaveBeenCalledWith({
        invite_code: 'ABC-123',
        invitation_token: 'inv-1',
        accept_terms: true,
      });
      expect(neon.clearPendingSignup).toHaveBeenCalled();
      expect(useAuthStore.getState().isAuthenticated).toBe(true);
    });

    it('links an existing user with an empty provision body', async () => {
      userApi.getCurrentUser
        .mockRejectedValueOnce(apiError(401, 'account_not_linked'))
        .mockResolvedValueOnce({ data: user });
      authApi.provisionNeonAccount.mockResolvedValue({ data: { linked: true } });

      await useAuthStore.getState().login('pat@example.com', 'password123');
      expect(authApi.provisionNeonAccount).toHaveBeenCalledWith({
        invite_code: undefined,
        invitation_token: undefined,
        accept_terms: undefined,
      });
    });

    it('signs out of Neon Auth and rethrows when provisioning is refused', async () => {
      userApi.getCurrentUser.mockRejectedValue(apiError(401, 'account_not_linked'));
      authApi.provisionNeonAccount.mockRejectedValue(apiError(403, 'registration_closed'));

      await expect(
        useAuthStore.getState().login('pat@example.com', 'password123')
      ).rejects.toMatchObject({ response: { status: 403 } });
      expect(neon.neonSignOut).toHaveBeenCalled();
      expect(useAuthStore.getState()).toMatchObject({ isAuthenticated: false, isLoading: false });
    });

    it('passes Neon Auth errors through (wrong password)', async () => {
      neon.signInWithEmail.mockRejectedValueOnce(new Error('invalid'));
      await expect(
        useAuthStore.getState().login('pat@example.com', 'wrong-pass')
      ).rejects.toThrow('invalid');
      expect(userApi.getCurrentUser).not.toHaveBeenCalled();
    });
  });

  it('register keeps the invite for later, creates the Neon sign-in and emails a code', async () => {
    await useAuthStore.getState().register('pat@example.com', 'password123', {
      inviteCode: 'ABC-123',
      acceptTerms: true,
    });

    expect(neon.savePendingSignup).toHaveBeenCalledWith({
      email: 'pat@example.com',
      inviteCode: 'ABC-123',
      invitationToken: undefined,
      acceptTerms: true,
    });
    expect(neon.signUpWithEmail).toHaveBeenCalledWith('pat@example.com', 'password123');
    expect(neon.sendVerificationCode).toHaveBeenCalledWith('pat@example.com');
    expect(authApi.register).not.toHaveBeenCalled();
  });

  it('logout signs out of Neon Auth (not the cookie endpoint) and clears local state', async () => {
    useAuthStore.setState({ user: user as never, isAuthenticated: true });
    await useAuthStore.getState().logout();

    expect(neon.neonSignOut).toHaveBeenCalled();
    expect(neon.clearPendingSignup).toHaveBeenCalled();
    expect(authApi.logout).not.toHaveBeenCalled();
    expect(clearSessionState).toHaveBeenCalled();
    expect(useAuthStore.getState()).toMatchObject({ user: null, isAuthenticated: false });
  });

  describe('checkAuth', () => {
    it('is signed out without a Neon Auth session, without asking the API', async () => {
      useAuthStore.setState({ isAuthenticated: true });
      neon.getNeonAccessToken.mockResolvedValue(null);

      expect(await useAuthStore.getState().checkAuth()).toBe(false);
      expect(userApi.getCurrentUser).not.toHaveBeenCalled();
      expect(useAuthStore.getState().isAuthenticated).toBe(false);
    });

    it('loads the user with a fresh token', async () => {
      userApi.getCurrentUser.mockResolvedValue({ data: user });
      expect(await useAuthStore.getState().checkAuth()).toBe(true);
      expect(neon.getNeonAccessToken).toHaveBeenCalledWith({ forceRefresh: true });
    });

    it('keeps the current state on a network error (e.g. reload mid-check)', async () => {
      useAuthStore.setState({ user: user as never, isAuthenticated: true });
      neon.getNeonAccessToken.mockRejectedValue(new TypeError('Failed to fetch'));

      expect(await useAuthStore.getState().checkAuth()).toBe(true);
      expect(useAuthStore.getState().isAuthenticated).toBe(true);
    });

    it('keeps the current state when the API is down, signs out on 401', async () => {
      useAuthStore.setState({ user: user as never, isAuthenticated: true });
      userApi.getCurrentUser.mockRejectedValueOnce(apiError(503, 'unavailable'));
      expect(await useAuthStore.getState().checkAuth()).toBe(true);

      userApi.getCurrentUser.mockRejectedValueOnce(apiError(401, 'account_not_linked'));
      expect(await useAuthStore.getState().checkAuth()).toBe(false);
      expect(useAuthStore.getState().isAuthenticated).toBe(false);
    });
  });

  describe('completeSignIn', () => {
    it('throws not_signed_in when there is no Neon session', async () => {
      neon.getNeonAccessToken.mockResolvedValue(null);
      await expect(useAuthStore.getState().completeSignIn()).rejects.toMatchObject({
        code: 'not_signed_in',
      });
    });

    it('finishes a sign-in made outside login (email code, Google)', async () => {
      userApi.getCurrentUser.mockResolvedValue({ data: user });
      await useAuthStore.getState().completeSignIn();
      expect(useAuthStore.getState()).toMatchObject({ user, isAuthenticated: true });
    });
  });

  it('refreshAccessToken forces a new JWT and logs out when there is none', async () => {
    await useAuthStore.getState().refreshAccessToken();
    expect(neon.getNeonAccessToken).toHaveBeenCalledWith({ forceRefresh: true });

    neon.getNeonAccessToken.mockResolvedValue(null);
    await expect(useAuthStore.getState().refreshAccessToken()).rejects.toMatchObject({
      code: 'not_signed_in',
    });
    expect(neon.neonSignOut).toHaveBeenCalled();
  });
});
