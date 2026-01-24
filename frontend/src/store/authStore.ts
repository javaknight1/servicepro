import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { authApi, userApi } from '@services/api';
import type { AuthState, User } from '@app-types';

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      token: null,
      refreshToken: null,
      isAuthenticated: false,
      isLoading: false,

      login: async (email: string, password: string) => {
        try {
          set({ isLoading: true });
          const response = await authApi.login({ email, password });
          const { token, refreshToken, expiresIn } = response.data;

          // Save tokens to localStorage
          localStorage.setItem('access_token', token);
          localStorage.setItem('refresh_token', refreshToken);

          // Update state
          set({
            token,
            refreshToken,
            isAuthenticated: true,
            isLoading: false,
          });

          // Fetch user profile after login
          try {
            const userResponse = await userApi.getCurrentUser();
            set({ user: userResponse.data });
          } catch (userError) {
            console.error('Failed to fetch user profile:', userError);
          }
        } catch (error) {
          set({ isLoading: false });
          throw error;
        }
      },

      register: async (email: string, password: string) => {
        try {
          set({ isLoading: true });
          await authApi.register({ email, password });
          set({ isLoading: false });
          // Registration successful - user needs to verify email
        } catch (error) {
          set({ isLoading: false });
          throw error;
        }
      },

      logout: () => {
        // Clear tokens from localStorage
        localStorage.removeItem('access_token');
        localStorage.removeItem('refresh_token');

        // Clear state
        set({
          user: null,
          token: null,
          refreshToken: null,
          isAuthenticated: false,
        });
      },

      refreshAccessToken: async () => {
        try {
          const currentRefreshToken = get().refreshToken;
          if (!currentRefreshToken) {
            throw new Error('No refresh token available');
          }

          const response = await authApi.refreshToken(currentRefreshToken);
          const { token, refreshToken } = response.data;

          // Save new tokens
          localStorage.setItem('access_token', token);
          localStorage.setItem('refresh_token', refreshToken);

          // Update state
          set({
            token,
            refreshToken,
          });
        } catch (error) {
          // Refresh failed - logout user
          get().logout();
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
        const token = get().token;
        if (!token) return;

        try {
          const response = await userApi.getCurrentUser();
          set({ user: response.data });
        } catch (error) {
          console.error('Failed to fetch current user:', error);
        }
      },
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({
        token: state.token,
        refreshToken: state.refreshToken,
        isAuthenticated: state.isAuthenticated,
        user: state.user,
      }),
    }
  )
);
