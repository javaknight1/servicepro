import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { authApi, userApi } from '@services/api';
import { setUser as setErrorTrackingUser } from '@services/errorTracking';
import {
  identifyUser as identifyAnalyticsUser,
  resetUser as resetAnalyticsUser,
} from '@services/analytics';
import { clearSessionState } from '@services/session';
import { clearQueryCaches } from '@services/queryCacheRegistry';
import { NeonAuthError } from '@services/authErrors';
import { isNeonAuthEnabled } from '@config/auth';
import { getApiErrorInfo } from '@/utils/error';
import type { AuthState, RegisterOptions, User } from '@app-types';

/** Neon Auth flows (T158); loaded only when VITE_NEON_AUTH_URL is set. */
const loadNeonAuth = () => import('@services/neonAuth');

type SetState = (partial: Partial<AuthState>) => void;
type GetState = () => AuthState;

/** Record the signed-in user in the store, Sentry and analytics. */
function applySignedInUser(
  set: SetState,
  get: GetState,
  user: User,
  { clearCachesForSameUser }: { clearCachesForSameUser: boolean }
): void {
  const previousUserId = get().user?.id;
  if (previousUserId && previousUserId !== user.id) {
    // A different user than the one whose data may still be cached
    clearSessionState();
  } else if (clearCachesForSameUser) {
    clearQueryCaches();
  }
  set({ user, isAuthenticated: true });
  setErrorTrackingUser({ id: user.id, email: user.email });
  identifyAnalyticsUser({ id: user.id, email: user.email });
}

/**
 * GET /users/me for the Neon Auth session. A first sign-in answers 401
 * account_not_linked: POST /auth/neon/provision links the existing user with
 * the same (verified) email, or creates one with the invite code / invitation
 * and terms saved at sign-up, then the profile is fetched again.
 */
async function fetchOrProvisionUser(): Promise<User> {
  try {
    return (await userApi.getCurrentUser()).data;
  } catch (err: unknown) {
    if (getApiErrorInfo(err).code !== 'account_not_linked') throw err;
  }
  const neon = await loadNeonAuth();
  const pending = neon.readPendingSignup();
  await authApi.provisionNeonAccount({
    invite_code: pending?.inviteCode,
    invitation_token: pending?.invitationToken,
    accept_terms: pending?.acceptTerms ? true : undefined,
  });
  neon.clearPendingSignup();
  return (await userApi.getCurrentUser()).data;
}

/** Finish a Neon Auth sign-in; a refused account is signed out of Neon too. */
async function completeNeonSignIn(set: SetState, get: GetState): Promise<void> {
  const neon = await loadNeonAuth();
  const token = await neon.getNeonAccessToken({ forceRefresh: true });
  if (!token) {
    throw new NeonAuthError(
      'Your sign-in has expired. Please sign in again.',
      'not_signed_in',
      401
    );
  }
  let user: User;
  try {
    user = await fetchOrProvisionUser();
  } catch (err: unknown) {
    // No ServicePro account for this sign-in (not invited, email not
    // verified, ...): don't leave a half-signed-in Neon session behind.
    await neon.neonSignOut().catch(() => undefined);
    throw err;
  }
  applySignedInUser(set, get, user, { clearCachesForSameUser: true });
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      isAuthenticated: false,
      isLoading: false,

      login: async (email: string, password: string) => {
        if (isNeonAuthEnabled) {
          set({ isLoading: true });
          try {
            const neon = await loadNeonAuth();
            await neon.signInWithEmail(email, password);
            await completeNeonSignIn(set, get);
          } finally {
            set({ isLoading: false });
          }
          return;
        }

        try {
          set({ isLoading: true });
          // Login sets httpOnly cookies automatically
          await authApi.login({ email, password });

          // Update state (tokens are in httpOnly cookies, not accessible to JS)
          set({
            isAuthenticated: true,
            isLoading: false,
          });

          // Fetch user profile after login
          try {
            const userResponse = await userApi.getCurrentUser();
            applySignedInUser(set, get, userResponse.data, {
              clearCachesForSameUser: true,
            });
          } catch (userError) {
            console.error('Failed to fetch user profile:', userError);
          }
        } catch (error) {
          set({ isLoading: false });
          throw error;
        }
      },

      register: async (
        email: string,
        password: string,
        options: RegisterOptions = {}
      ) => {
        if (isNeonAuthEnabled) {
          set({ isLoading: true });
          try {
            const neon = await loadNeonAuth();
            // Provisioning happens once the email is verified (the API only
            // links or creates accounts for verified emails); keep what it
            // needs until then.
            neon.savePendingSignup({
              email,
              inviteCode: options.inviteCode || undefined,
              invitationToken: options.invitationToken || undefined,
              acceptTerms: options.acceptTerms === true,
            });
            await neon.signUpWithEmail(email, password);
            await neon.sendVerificationCode(email);
          } finally {
            set({ isLoading: false });
          }
          return;
        }

        try {
          set({ isLoading: true });
          await authApi.register({
            email,
            password,
            invitation_token: options.invitationToken || undefined,
            invite_code: options.inviteCode || undefined,
            accept_terms: options.acceptTerms === true,
          });
          set({ isLoading: false });
          // Registration successful - user needs to verify email
        } catch (error) {
          set({ isLoading: false });
          throw error;
        }
      },

      logout: async () => {
        if (isNeonAuthEnabled) {
          try {
            const neon = await loadNeonAuth();
            neon.clearPendingSignup();
            // Ends the Neon Auth session. An already-issued JWT stays valid
            // at the API until it expires (at most 15 minutes).
            await neon.neonSignOut();
          } catch (error) {
            console.error('Neon Auth sign-out failed:', error);
          }
        } else {
          try {
            // Call backend to clear httpOnly cookies
            await authApi.logout();
          } catch (error) {
            // Even if the API call fails, clear local state
            console.error('Logout API call failed:', error);
          }
        }

        // Clear local state, including cached server data and tenant context,
        // so the next user on this browser cannot see this user's data.
        clearSessionState();
        set({
          user: null,
          isAuthenticated: false,
        });
        setErrorTrackingUser(null);
        resetAnalyticsUser();
      },

      refreshAccessToken: async () => {
        if (isNeonAuthEnabled) {
          const neon = await loadNeonAuth();
          const token = await neon.getNeonAccessToken({ forceRefresh: true });
          if (!token) {
            await get().logout();
            throw new NeonAuthError(
              'Your sign-in has expired. Please sign in again.',
              'not_signed_in',
              401
            );
          }
          return;
        }

        try {
          // Refresh token cookie is sent automatically
          await authApi.refreshToken();
          // New access token cookie is set automatically by the backend
        } catch (error) {
          // Refresh failed - logout user
          await get().logout();
          throw error;
        }
      },

      setUser: (user: User) => {
        set({ user });
      },

      updateProfilePicture: (url: string | null) => {
        const currentUser = get().user;
        if (currentUser) {
          set({
            user: {
              ...currentUser,
              profile_picture_url: url ?? undefined,
            },
          });
        }
      },

      fetchCurrentUser: async () => {
        // Check if authenticated (cookies are sent automatically)
        if (!get().isAuthenticated) return;

        try {
          const response = await userApi.getCurrentUser();
          set({ user: response.data });
        } catch (error) {
          console.error('Failed to fetch current user:', error);
        }
      },

      checkAuth: async () => {
        const signedOut = () => {
          set({ user: null, isAuthenticated: false });
          setErrorTrackingUser(null);
          resetAnalyticsUser();
          return false;
        };

        if (isNeonAuthEnabled) {
          // Only a definite answer signs the user out: no Neon Auth session,
          // or the API refusing the token. A network error (offline, or the
          // page unloading mid-request) keeps the current state, so a reload
          // during the check doesn't persist a false "signed out".
          let token: string | null;
          try {
            const neon = await loadNeonAuth();
            token = await neon.getNeonAccessToken({ forceRefresh: true });
          } catch {
            return get().isAuthenticated;
          }
          if (!token) return signedOut();
          try {
            const response = await userApi.getCurrentUser();
            applySignedInUser(set, get, response.data, {
              clearCachesForSameUser: false,
            });
            return true;
          } catch (err: unknown) {
            const { status } = getApiErrorInfo(err);
            if (status === 401 || status === 403) return signedOut();
            return get().isAuthenticated;
          }
        }

        // Try to fetch current user to verify authentication
        try {
          const response = await userApi.getCurrentUser();
          // A session for a different user than the persisted one clears
          // that user's cached data.
          applySignedInUser(set, get, response.data, {
            clearCachesForSameUser: false,
          });
          return true;
        } catch {
          return signedOut();
        }
      },

      completeSignIn: async () => {
        if (isNeonAuthEnabled) {
          set({ isLoading: true });
          try {
            await completeNeonSignIn(set, get);
          } finally {
            set({ isLoading: false });
          }
          return;
        }
        if (!(await get().checkAuth())) {
          throw new Error('Not signed in');
        }
      },
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({
        isAuthenticated: state.isAuthenticated,
        user: state.user,
      }),
    }
  )
);
