/**
 * authStore on a cookie-auth build (VITE_NEON_AUTH_URL unset): the API's own
 * sign-in, unchanged by T158. Neon Auth must never be touched.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('@config/auth', () => ({
  NEON_AUTH_URL: '',
  isNeonAuthEnabled: false,
  isGoogleSignInEnabled: false,
}));

const neonModule = vi.hoisted(() => vi.fn());
vi.mock('@services/neonAuth', () => {
  neonModule();
  return {};
});

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

describe('authStore with cookie sign-in', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useAuthStore.setState({ user: null, isAuthenticated: false, isLoading: false });
  });

  it('login posts credentials to /auth/login and loads the profile', async () => {
    authApi.login.mockResolvedValue({ data: {} });
    userApi.getCurrentUser.mockResolvedValue({ data: user });

    await useAuthStore.getState().login('pat@example.com', 'password123');

    expect(authApi.login).toHaveBeenCalledWith({
      email: 'pat@example.com',
      password: 'password123',
    });
    expect(useAuthStore.getState()).toMatchObject({ user, isAuthenticated: true });
  });

  it('a different user than the persisted one clears the old session data', async () => {
    useAuthStore.setState({ user: { id: 'other', email: 'o@example.com' } as never });
    authApi.login.mockResolvedValue({ data: {} });
    userApi.getCurrentUser.mockResolvedValue({ data: user });

    await useAuthStore.getState().login('pat@example.com', 'password123');
    expect(clearSessionState).toHaveBeenCalled();
  });

  it('register posts to /auth/register with invite and terms', async () => {
    authApi.register.mockResolvedValue({ data: {} });
    await useAuthStore.getState().register('pat@example.com', 'password123', {
      inviteCode: 'ABC',
      acceptTerms: true,
    });
    expect(authApi.register).toHaveBeenCalledWith({
      email: 'pat@example.com',
      password: 'password123',
      invitation_token: undefined,
      invite_code: 'ABC',
      accept_terms: true,
    });
  });

  it('logout calls /auth/logout and clears local state', async () => {
    useAuthStore.setState({ user: user as never, isAuthenticated: true });
    authApi.logout.mockResolvedValue({});
    await useAuthStore.getState().logout();
    expect(authApi.logout).toHaveBeenCalled();
    expect(useAuthStore.getState()).toMatchObject({ user: null, isAuthenticated: false });
  });

  it('checkAuth signs out on any failure, as before', async () => {
    useAuthStore.setState({ isAuthenticated: true });
    userApi.getCurrentUser.mockRejectedValue(new Error('network'));
    expect(await useAuthStore.getState().checkAuth()).toBe(false);
    expect(useAuthStore.getState().isAuthenticated).toBe(false);
  });

  it('refreshAccessToken uses the refresh cookie endpoint', async () => {
    authApi.refreshToken.mockResolvedValue({});
    await useAuthStore.getState().refreshAccessToken();
    expect(authApi.refreshToken).toHaveBeenCalled();
  });

  it('never loads the Neon Auth module', async () => {
    authApi.login.mockResolvedValue({ data: {} });
    userApi.getCurrentUser.mockResolvedValue({ data: user });
    await useAuthStore.getState().login('pat@example.com', 'password123');
    await useAuthStore.getState().checkAuth();
    await useAuthStore.getState().logout();
    expect(neonModule).not.toHaveBeenCalled();
  });
});
