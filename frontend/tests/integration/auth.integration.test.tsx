/**
 * Authentication Flow Integration Tests
 *
 * Tests complete authentication workflows including:
 * - Login → Dashboard navigation
 * - Registration → Email verification flow
 * - Password reset workflow
 * - Session management
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import React from 'react';

// Mock modules before imports
vi.mock('@store', () => ({
  useAuthStore: vi.fn(),
  useTenantStore: vi.fn(),
}));

vi.mock('@/services/api', () => ({
  authApi: {
    login: vi.fn(),
    register: vi.fn(),
    logout: vi.fn(),
    requestPasswordReset: vi.fn(),
    confirmPasswordReset: vi.fn(),
    verifyEmail: vi.fn(),
  },
  userApi: {
    getCurrentUser: vi.fn(),
  },
  default: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}));

vi.mock('@services/tenantService', () => ({
  invitationApi: {
    getByToken: vi.fn(),
    accept: vi.fn(),
  },
}));

// Import after mocking
import { useAuthStore, useTenantStore } from '@store';
import { authApi } from '@/services/api';
import { invitationApi } from '@services/tenantService';
import { LoginPage } from '../../src/pages/Login/LoginPage';
import { RegisterPage } from '../../src/pages/Register/RegisterPage';
import { ForgotPasswordPage } from '../../src/pages/ForgotPassword/ForgotPasswordPage';

// =============================================================================
// Test Setup
// =============================================================================

const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

const mockTenant = {
  id: 'tenant-123',
  name: 'Test Organization',
  slug: 'test-org',
  is_active: true,
};

// Minimal store mocks for the registration page
function mockRegisterStores(register: ReturnType<typeof vi.fn>) {
  vi.mocked(useAuthStore).mockImplementation((selector) => {
    const state = {
      user: null,
      isAuthenticated: false,
      isLoading: false,
      error: null,
      login: vi.fn(),
      logout: vi.fn(),
      register,
    };
    if (typeof selector === 'function') {
      return selector(state as never);
    }
    return state;
  });
  vi.mocked(useTenantStore).mockImplementation((selector) => {
    const state = { currentTenant: null, tenants: [] };
    if (typeof selector === 'function') {
      return selector(state as never);
    }
    return state;
  });
}

async function fillRegistration(
  user: ReturnType<typeof userEvent.setup>,
  email = 'newuser@example.com'
) {
  const emailInput = screen.getByLabelText(/email/i);
  if (!(emailInput as HTMLInputElement).disabled) {
    await user.type(emailInput, email);
  }
  await user.type(screen.getByLabelText(/^password$/i), 'SecurePass123!');
  await user.type(screen.getByLabelText(/confirm password/i), 'SecurePass123!');
}

// =============================================================================
// Login Flow Integration Tests
// =============================================================================

describe('Authentication Flow Integration', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Login → Dashboard Flow', () => {
    it('should complete full login flow and navigate to dashboard', async () => {
      const user = userEvent.setup({ delay: null });
      const mockLogin = vi.fn().mockResolvedValue(undefined);

      vi.mocked(useAuthStore).mockImplementation((selector) => {
        const state = {
          user: null,
          isAuthenticated: false,
          isLoading: false,
          error: null,
          login: mockLogin,
          logout: vi.fn(),
          loadUser: vi.fn(),
          clearError: vi.fn(),
        };
        if (typeof selector === 'function') {
          return selector(state);
        }
        return state;
      });

      vi.mocked(useTenantStore).mockImplementation((selector) => {
        const state = {
          currentTenant: mockTenant,
          tenants: [mockTenant],
          isLoading: false,
          error: null,
          loadTenants: vi.fn(),
          setCurrentTenant: vi.fn(),
        };
        if (typeof selector === 'function') {
          return selector(state);
        }
        return state;
      });

      render(
        <MemoryRouter initialEntries={['/login']}>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route path="/dashboard" element={<div>Dashboard</div>} />
          </Routes>
        </MemoryRouter>
      );

      // Fill in login form
      await user.type(screen.getByLabelText(/email/i), 'test@example.com');
      await user.type(screen.getByLabelText(/password/i), 'password123');
      await user.click(screen.getByRole('button', { name: /sign in/i }));

      // Verify login was called with correct credentials
      await waitFor(() => {
        expect(mockLogin).toHaveBeenCalledWith(
          'test@example.com',
          'password123'
        );
      });

      // Verify navigation to dashboard
      await waitFor(() => {
        expect(mockNavigate).toHaveBeenCalledWith('/dashboard');
      });
    });

    it('should display error and stay on login page on failed login', async () => {
      const user = userEvent.setup({ delay: null });
      const mockLogin = vi.fn().mockRejectedValue({
        response: { data: { message: 'Invalid credentials' } },
      });

      vi.mocked(useAuthStore).mockImplementation((selector) => {
        const state = {
          user: null,
          isAuthenticated: false,
          isLoading: false,
          error: null,
          login: mockLogin,
          logout: vi.fn(),
          loadUser: vi.fn(),
          clearError: vi.fn(),
        };
        if (typeof selector === 'function') {
          return selector(state);
        }
        return state;
      });

      vi.mocked(useTenantStore).mockImplementation((selector) => {
        const state = {
          currentTenant: null,
          tenants: [],
          isLoading: false,
          error: null,
          loadTenants: vi.fn(),
          setCurrentTenant: vi.fn(),
        };
        if (typeof selector === 'function') {
          return selector(state);
        }
        return state;
      });

      render(
        <MemoryRouter initialEntries={['/login']}>
          <LoginPage />
        </MemoryRouter>
      );

      await user.type(screen.getByLabelText(/email/i), 'test@example.com');
      await user.type(screen.getByLabelText(/password/i), 'wrongpassword');
      await user.click(screen.getByRole('button', { name: /sign in/i }));

      // Should display error message
      await waitFor(() => {
        expect(screen.getByText(/invalid credentials/i)).toBeInTheDocument();
      });

      // Should not navigate away
      expect(mockNavigate).not.toHaveBeenCalled();
    });

    it('should show a generic retry message when logins are throttled (429)', async () => {
      const user = userEvent.setup({ delay: null });
      const mockLogin = vi.fn().mockRejectedValue({
        response: {
          status: 429,
          data: {
            error: 'rate_limit_exceeded',
            message: 'Too many login attempts.',
          },
        },
      });
      vi.mocked(useAuthStore).mockImplementation((selector) => {
        const state = { user: null, isAuthenticated: false, login: mockLogin };
        if (typeof selector === 'function') {
          return selector(state as never);
        }
        return state;
      });

      render(
        <MemoryRouter initialEntries={['/login']}>
          <LoginPage />
        </MemoryRouter>
      );

      await user.type(screen.getByLabelText(/email/i), 'test@example.com');
      await user.type(screen.getByLabelText(/password/i), 'password123');
      await user.click(screen.getByRole('button', { name: /sign in/i }));

      expect(
        await screen.findByText('Too many attempts, try again shortly.')
      ).toBeInTheDocument();
      expect(mockNavigate).not.toHaveBeenCalled();
    });

    it('should navigate to register page from login', async () => {
      const user = userEvent.setup({ delay: null });

      vi.mocked(useAuthStore).mockImplementation((selector) => {
        const state = {
          user: null,
          isAuthenticated: false,
          isLoading: false,
          error: null,
          login: vi.fn(),
          logout: vi.fn(),
          loadUser: vi.fn(),
          clearError: vi.fn(),
        };
        if (typeof selector === 'function') {
          return selector(state);
        }
        return state;
      });

      vi.mocked(useTenantStore).mockImplementation((selector) => {
        const state = {
          currentTenant: null,
          tenants: [],
          isLoading: false,
          error: null,
          loadTenants: vi.fn(),
          setCurrentTenant: vi.fn(),
        };
        if (typeof selector === 'function') {
          return selector(state);
        }
        return state;
      });

      render(
        <MemoryRouter initialEntries={['/login']}>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route path="/register" element={<div>Register Page</div>} />
          </Routes>
        </MemoryRouter>
      );

      const registerLink = screen.getByRole('link', {
        name: /create a new account/i,
      });
      expect(registerLink).toHaveAttribute('href', '/register');

      await user.click(registerLink);

      await waitFor(() => {
        expect(screen.getByText('Register Page')).toBeInTheDocument();
      });
    });

    it('should navigate to forgot password from login', async () => {
      const user = userEvent.setup({ delay: null });

      vi.mocked(useAuthStore).mockImplementation((selector) => {
        const state = {
          user: null,
          isAuthenticated: false,
          isLoading: false,
          error: null,
          login: vi.fn(),
          logout: vi.fn(),
          loadUser: vi.fn(),
          clearError: vi.fn(),
        };
        if (typeof selector === 'function') {
          return selector(state);
        }
        return state;
      });

      vi.mocked(useTenantStore).mockImplementation((selector) => {
        const state = {
          currentTenant: null,
          tenants: [],
          isLoading: false,
          error: null,
          loadTenants: vi.fn(),
          setCurrentTenant: vi.fn(),
        };
        if (typeof selector === 'function') {
          return selector(state);
        }
        return state;
      });

      render(
        <MemoryRouter initialEntries={['/login']}>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route
              path="/forgot-password"
              element={<div>Forgot Password Page</div>}
            />
          </Routes>
        </MemoryRouter>
      );

      const forgotPasswordLink = screen.getByRole('link', {
        name: /forgot your password/i,
      });
      expect(forgotPasswordLink).toHaveAttribute('href', '/forgot-password');

      await user.click(forgotPasswordLink);

      await waitFor(() => {
        expect(screen.getByText('Forgot Password Page')).toBeInTheDocument();
      });
    });
  });

  describe('Registration Flow', () => {
    it('should complete registration and show success message', async () => {
      const user = userEvent.setup({ delay: null });
      const mockRegister = vi.fn().mockResolvedValue(undefined);

      vi.mocked(useAuthStore).mockImplementation((selector) => {
        const state = {
          user: null,
          isAuthenticated: false,
          isLoading: false,
          error: null,
          login: vi.fn(),
          logout: vi.fn(),
          loadUser: vi.fn(),
          clearError: vi.fn(),
          register: mockRegister,
        };
        if (typeof selector === 'function') {
          return selector(state);
        }
        return state;
      });

      vi.mocked(useTenantStore).mockImplementation((selector) => {
        const state = {
          currentTenant: null,
          tenants: [],
          isLoading: false,
          error: null,
          loadTenants: vi.fn(),
          setCurrentTenant: vi.fn(),
        };
        if (typeof selector === 'function') {
          return selector(state);
        }
        return state;
      });

      render(
        <MemoryRouter initialEntries={['/register']}>
          <RegisterPage />
        </MemoryRouter>
      );

      // Fill registration form
      await user.type(screen.getByLabelText(/email/i), 'newuser@example.com');
      await user.type(screen.getByLabelText(/^password$/i), 'SecurePass123!');
      await user.type(
        screen.getByLabelText(/confirm password/i),
        'SecurePass123!'
      );

      // Check terms checkbox if present
      const termsCheckbox = screen.queryByRole('checkbox');
      if (termsCheckbox) {
        await user.click(termsCheckbox);
      }

      await user.click(
        screen.getByRole('button', { name: /create account|sign up|register/i })
      );

      // Verify register was called
      await waitFor(() => {
        expect(mockRegister).toHaveBeenCalled();
      });
    });

    it('should show validation errors for invalid registration data', async () => {
      const user = userEvent.setup({ delay: null });

      vi.mocked(useAuthStore).mockImplementation((selector) => {
        const state = {
          user: null,
          isAuthenticated: false,
          isLoading: false,
          error: null,
          login: vi.fn(),
          logout: vi.fn(),
          loadUser: vi.fn(),
          clearError: vi.fn(),
          register: vi.fn(),
        };
        if (typeof selector === 'function') {
          return selector(state);
        }
        return state;
      });

      vi.mocked(useTenantStore).mockImplementation((selector) => {
        const state = {
          currentTenant: null,
          tenants: [],
          isLoading: false,
          error: null,
          loadTenants: vi.fn(),
          setCurrentTenant: vi.fn(),
        };
        if (typeof selector === 'function') {
          return selector(state);
        }
        return state;
      });

      render(
        <MemoryRouter initialEntries={['/register']}>
          <RegisterPage />
        </MemoryRouter>
      );

      // Try to submit empty form
      await user.click(
        screen.getByRole('button', { name: /create account|sign up|register/i })
      );

      // Should show validation errors
      await waitFor(() => {
        // Check for any validation error message
        const errorMessages = screen.getAllByText(/required|invalid|must be/i);
        expect(errorMessages.length).toBeGreaterThan(0);
      });
    });

    it('should show error when passwords do not match', async () => {
      const user = userEvent.setup({ delay: null });

      vi.mocked(useAuthStore).mockImplementation((selector) => {
        const state = {
          user: null,
          isAuthenticated: false,
          isLoading: false,
          error: null,
          login: vi.fn(),
          logout: vi.fn(),
          loadUser: vi.fn(),
          clearError: vi.fn(),
          register: vi.fn(),
        };
        if (typeof selector === 'function') {
          return selector(state);
        }
        return state;
      });

      vi.mocked(useTenantStore).mockImplementation((selector) => {
        const state = {
          currentTenant: null,
          tenants: [],
          isLoading: false,
          error: null,
          loadTenants: vi.fn(),
          setCurrentTenant: vi.fn(),
        };
        if (typeof selector === 'function') {
          return selector(state);
        }
        return state;
      });

      render(
        <MemoryRouter initialEntries={['/register']}>
          <RegisterPage />
        </MemoryRouter>
      );

      await user.type(screen.getByLabelText(/email/i), 'test@example.com');
      await user.type(screen.getByLabelText(/^password$/i), 'SecurePass123!');
      await user.type(
        screen.getByLabelText(/confirm password/i),
        'DifferentPass456!'
      );

      await user.click(
        screen.getByRole('button', { name: /create account|sign up|register/i })
      );

      await waitFor(() => {
        expect(
          screen.getByText(/passwords? (do not|don't) match/i)
        ).toBeInTheDocument();
      });
    });
  });

  describe('Invite-only registration', () => {
    const registrationClosed = {
      response: {
        status: 403,
        data: {
          error: 'registration_closed',
          message:
            'Registration is by invitation only. A valid invite code or invitation is required.',
        },
      },
    };

    it('prefills the invite code from ?invite= and sends it as invite_code', async () => {
      const user = userEvent.setup({ delay: null });
      const mockRegister = vi.fn().mockResolvedValue(undefined);
      mockRegisterStores(mockRegister);

      render(
        <MemoryRouter initialEntries={['/register?invite=EARLY-BIRD-42']}>
          <RegisterPage />
        </MemoryRouter>
      );

      expect(screen.getByLabelText(/invite code/i)).toHaveValue(
        'EARLY-BIRD-42'
      );
      await fillRegistration(user);
      await user.click(screen.getByRole('button', { name: /create account/i }));

      await waitFor(() => {
        expect(mockRegister).toHaveBeenCalledWith(
          'newuser@example.com',
          'SecurePass123!',
          { invitationToken: undefined, inviteCode: 'EARLY-BIRD-42' }
        );
      });
      expect(await screen.findByText(/check your email/i)).toBeInTheDocument();
    });

    it('also accepts ?code= links', () => {
      mockRegisterStores(vi.fn());
      render(
        <MemoryRouter initialEntries={['/register?code=ABC123']}>
          <RegisterPage />
        </MemoryRouter>
      );
      expect(screen.getByLabelText(/invite code/i)).toHaveValue('ABC123');
    });

    it('explains that ServicePro is invite-only when no code was given', async () => {
      const user = userEvent.setup({ delay: null });
      mockRegisterStores(vi.fn().mockRejectedValue(registrationClosed));

      render(
        <MemoryRouter initialEntries={['/register']}>
          <RegisterPage />
        </MemoryRouter>
      );
      await fillRegistration(user);
      await user.click(screen.getByRole('button', { name: /create account/i }));

      expect(
        await screen.findByText(
          /ServicePro is invite-only right now\. Enter your invite code/i
        )
      ).toBeInTheDocument();
    });

    it('says the invite code is invalid when the API rejects it', async () => {
      const user = userEvent.setup({ delay: null });
      mockRegisterStores(vi.fn().mockRejectedValue(registrationClosed));

      render(
        <MemoryRouter initialEntries={['/register']}>
          <RegisterPage />
        </MemoryRouter>
      );
      await fillRegistration(user);
      await user.type(screen.getByLabelText(/invite code/i), 'NOPE');
      await user.click(screen.getByRole('button', { name: /create account/i }));

      expect(
        await screen.findByText(/that invite code isn't valid/i)
      ).toBeInTheDocument();
    });

    it('shows a throttling message on 429', async () => {
      const user = userEvent.setup({ delay: null });
      mockRegisterStores(
        vi.fn().mockRejectedValue({ response: { status: 429, data: {} } })
      );

      render(
        <MemoryRouter initialEntries={['/register?invite=X']}>
          <RegisterPage />
        </MemoryRouter>
      );
      await fillRegistration(user);
      await user.click(screen.getByRole('button', { name: /create account/i }));

      expect(
        await screen.findByText('Too many attempts, try again shortly.')
      ).toBeInTheDocument();
    });

    it('registers through a tenant invitation link without an invite code', async () => {
      const user = userEvent.setup({ delay: null });
      const mockRegister = vi.fn().mockResolvedValue(undefined);
      mockRegisterStores(mockRegister);
      vi.mocked(invitationApi.getByToken).mockResolvedValue({
        data: {
          id: 'inv-1',
          tenant_id: 'tenant-123',
          tenant_name: 'Acme HVAC',
          email: 'invitee@example.com',
          role_id: 'role-1',
          role_name: 'user',
          status: 'pending',
          invited_by_name: 'Owner',
          expires_at: '2099-01-01T00:00:00Z',
          created_at: '2026-10-01T00:00:00Z',
          user_exists: false,
          user_registered: false,
        },
      } as Awaited<ReturnType<typeof invitationApi.getByToken>>);

      render(
        <MemoryRouter
          initialEntries={[
            '/register?invitation=tok-123&email=invitee@example.com',
          ]}
        >
          <RegisterPage />
        </MemoryRouter>
      );

      expect(await screen.findByText('Acme HVAC')).toBeInTheDocument();
      // The invitation is enough; no invite-code field is shown.
      expect(screen.queryByLabelText(/invite code/i)).not.toBeInTheDocument();

      await fillRegistration(user);
      await user.click(
        screen.getByRole('button', { name: /create account & join/i })
      );

      await waitFor(() => {
        expect(mockRegister).toHaveBeenCalledWith(
          'invitee@example.com',
          'SecurePass123!',
          { invitationToken: 'tok-123', inviteCode: undefined }
        );
      });
    });

    it('explains a rejected tenant invitation', async () => {
      const user = userEvent.setup({ delay: null });
      mockRegisterStores(vi.fn().mockRejectedValue(registrationClosed));
      vi.mocked(invitationApi.getByToken).mockResolvedValue({
        data: {
          id: 'inv-1',
          tenant_id: 'tenant-123',
          tenant_name: 'Acme HVAC',
          email: 'invitee@example.com',
          role_id: 'role-1',
          role_name: 'user',
          status: 'pending',
          invited_by_name: 'Owner',
          expires_at: '2099-01-01T00:00:00Z',
          created_at: '2026-10-01T00:00:00Z',
          user_exists: false,
          user_registered: false,
        },
      } as Awaited<ReturnType<typeof invitationApi.getByToken>>);

      render(
        <MemoryRouter initialEntries={['/register?invitation=tok-123']}>
          <RegisterPage />
        </MemoryRouter>
      );
      expect(await screen.findByText('Acme HVAC')).toBeInTheDocument();
      await fillRegistration(user);
      await user.click(
        screen.getByRole('button', { name: /create account & join/i })
      );

      expect(
        await screen.findByText(/this invitation is no longer valid/i)
      ).toBeInTheDocument();
    });
  });

  describe('Password Reset Flow', () => {
    it('should send password reset request successfully', async () => {
      const user = userEvent.setup({ delay: null });
      const mockRequestReset = vi.fn().mockResolvedValue(undefined);

      vi.mocked(useAuthStore).mockImplementation((selector) => {
        const state = {
          user: null,
          isAuthenticated: false,
          isLoading: false,
          error: null,
          login: vi.fn(),
          logout: vi.fn(),
          loadUser: vi.fn(),
          clearError: vi.fn(),
          requestPasswordReset: mockRequestReset,
        };
        if (typeof selector === 'function') {
          return selector(state);
        }
        return state;
      });

      vi.mocked(useTenantStore).mockImplementation((selector) => {
        const state = {
          currentTenant: null,
          tenants: [],
          isLoading: false,
          error: null,
          loadTenants: vi.fn(),
          setCurrentTenant: vi.fn(),
        };
        if (typeof selector === 'function') {
          return selector(state);
        }
        return state;
      });

      render(
        <MemoryRouter initialEntries={['/forgot-password']}>
          <ForgotPasswordPage />
        </MemoryRouter>
      );

      await user.type(screen.getByLabelText(/email/i), 'test@example.com');
      await user.click(
        screen.getByRole('button', { name: /send|reset|submit/i })
      );

      // The page calls the API directly (not via the auth store)
      await waitFor(() => {
        expect(authApi.requestPasswordReset).toHaveBeenCalledWith({
          email: 'test@example.com',
        });
      });
    });

    it('should show success message after password reset request', async () => {
      const user = userEvent.setup({ delay: null });
      const mockRequestReset = vi.fn().mockResolvedValue(undefined);

      vi.mocked(useAuthStore).mockImplementation((selector) => {
        const state = {
          user: null,
          isAuthenticated: false,
          isLoading: false,
          error: null,
          login: vi.fn(),
          logout: vi.fn(),
          loadUser: vi.fn(),
          clearError: vi.fn(),
          requestPasswordReset: mockRequestReset,
        };
        if (typeof selector === 'function') {
          return selector(state);
        }
        return state;
      });

      vi.mocked(useTenantStore).mockImplementation((selector) => {
        const state = {
          currentTenant: null,
          tenants: [],
          isLoading: false,
          error: null,
          loadTenants: vi.fn(),
          setCurrentTenant: vi.fn(),
        };
        if (typeof selector === 'function') {
          return selector(state);
        }
        return state;
      });

      render(
        <MemoryRouter initialEntries={['/forgot-password']}>
          <ForgotPasswordPage />
        </MemoryRouter>
      );

      await user.type(screen.getByLabelText(/email/i), 'test@example.com');
      await user.click(
        screen.getByRole('button', { name: /send|reset|submit/i })
      );

      // Should show success message about email being sent
      expect(
        await screen.findByText(
          /check your email|email sent|instructions sent/i
        )
      ).toBeInTheDocument();
    });
  });

  describe('Session Management', () => {
    it('should handle session expiration gracefully', async () => {
      vi.mocked(useAuthStore).mockImplementation((selector) => {
        const state = {
          user: null,
          isAuthenticated: false,
          isLoading: false,
          error: 'Session expired',
          login: vi.fn(),
          logout: vi.fn(),
          loadUser: vi.fn(),
          clearError: vi.fn(),
        };
        if (typeof selector === 'function') {
          return selector(state);
        }
        return state;
      });

      vi.mocked(useTenantStore).mockImplementation((selector) => {
        const state = {
          currentTenant: null,
          tenants: [],
          isLoading: false,
          error: null,
          loadTenants: vi.fn(),
          setCurrentTenant: vi.fn(),
        };
        if (typeof selector === 'function') {
          return selector(state);
        }
        return state;
      });

      render(
        <MemoryRouter initialEntries={['/login']}>
          <LoginPage />
        </MemoryRouter>
      );

      // Should be on login page and able to re-authenticate
      expect(
        screen.getByRole('heading', { name: /sign in/i })
      ).toBeInTheDocument();
    });
  });
});
