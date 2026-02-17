import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { authApi, userApi } from '@services/api';
import { setUser as setErrorTrackingUser } from '@services/errorTracking';
import {
  identifyUser as identifyAnalyticsUser,
  resetUser as resetAnalyticsUser,
} from '@services/analytics';
import type { AuthState, User } from '@app-types';

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      isAuthenticated: false,
      isLoading: false,

      login: async (email: string, password: string) => {
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
            set({ user: userResponse.data });
            setErrorTrackingUser({
              id: userResponse.data.id,
              email: userResponse.data.email,
            });
            identifyAnalyticsUser({
              id: userResponse.data.id,
              email: userResponse.data.email,
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
        invitationToken?: string
      ) => {
        try {
          set({ isLoading: true });
          await authApi.register({
            email,
            password,
            invitation_token: invitationToken,
          });
          set({ isLoading: false });
          // Registration successful - user needs to verify email
        } catch (error) {
          set({ isLoading: false });
          throw error;
        }
      },

      logout: async () => {
        try {
          // Call backend to clear httpOnly cookies
          await authApi.logout();
        } catch (error) {
          // Even if the API call fails, clear local state
          console.error('Logout API call failed:', error);
        }

        // Clear local state
        set({
          user: null,
          isAuthenticated: false,
        });
        setErrorTrackingUser(null);
        resetAnalyticsUser();
      },

      refreshAccessToken: async () => {
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
        // Try to fetch current user to verify authentication
        try {
          const response = await userApi.getCurrentUser();
          set({ user: response.data, isAuthenticated: true });
          setErrorTrackingUser({
            id: response.data.id,
            email: response.data.email,
          });
          identifyAnalyticsUser({
            id: response.data.id,
            email: response.data.email,
          });
          return true;
        } catch {
          set({ user: null, isAuthenticated: false });
          setErrorTrackingUser(null);
          resetAnalyticsUser();
          return false;
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
