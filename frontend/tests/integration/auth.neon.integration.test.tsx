/**
 * Authentication flows on a Neon Auth build (VITE_NEON_AUTH_URL set, T158):
 * - Register (invite-only rules kept) → email code → account linked/created
 * - Verify email by code, resend, refusal after verification
 * - Password reset by code
 * The cookie-auth flows stay in auth.integration.test.tsx.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router';
import { AxiosError, type AxiosResponse } from 'axios';
import { NeonAuthError } from '@services/authErrors';

vi.mock('@config/auth', () => ({
  NEON_AUTH_URL: 'https://auth.example.test/neondb/auth',
  isNeonAuthEnabled: true,
  isGoogleSignInEnabled: true,
}));

const { mockRegister, mockCompleteSignIn, neon } = vi.hoisted(() => ({
  mockRegister: vi.fn(),
  mockCompleteSignIn: vi.fn(),
  neon: {
    verifyEmailCode: vi.fn(),
    sendVerificationCode: vi.fn(),
    requestPasswordResetCode: vi.fn(),
    resetPasswordWithCode: vi.fn(),
    savePendingSignup: vi.fn(),
    signInWithGoogle: vi.fn(),
  },
}));

vi.mock('@store', () => {
  const pick = (state: object) => (selector?: (s: object) => unknown) =>
    typeof selector === 'function' ? selector(state) : state;
  return {
    useAuthStore: vi.fn(
      pick({
        user: null,
        isAuthenticated: false,
        register: mockRegister,
        completeSignIn: mockCompleteSignIn,
        login: vi.fn(),
      })
    ),
    useTenantStore: vi.fn(pick({ currentTenant: null, tenants: [] })),
  };
});
vi.mock('@services/neonAuth', () => neon);
vi.mock('@services/tenantService', () => ({
  invitationApi: { getByToken: vi.fn(), accept: vi.fn() },
}));

const mockNavigate = vi.fn();
vi.mock('react-router', async () => {
  const actual = await vi.importActual('react-router');
  return { ...actual, useNavigate: () => mockNavigate };
});

import { invitationApi } from '@services/tenantService';
import { RegisterPage } from '../../src/pages/Register/RegisterPage';
import { NeonVerifyEmailPage } from '../../src/pages/VerifyEmail/NeonVerifyEmailPage';
import { NeonForgotPasswordPage } from '../../src/pages/ForgotPassword/NeonForgotPasswordPage';
import { NeonResetPasswordPage } from '../../src/pages/ResetPassword/NeonResetPasswordPage';

function apiError(status: number, code: string): AxiosError {
  const response = {
    status,
    data: { error: code, message: 'server message' },
    statusText: '',
    headers: {},
    config: {},
  } as unknown as AxiosResponse;
  return new AxiosError(code, String(status), undefined, null, response);
}

function renderAt(ui: React.ReactElement, path: string) {
  return render(<MemoryRouter initialEntries={[path]}>{ui}</MemoryRouter>);
}

describe('Neon Auth sign-in flows', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Register', () => {
    async function fillRegister(code?: string) {
      const user = userEvent.setup();
      await user.type(screen.getByLabelText(/email address/i), 'pat@example.com');
      if (code) await user.type(screen.getByLabelText(/invite code/i), code);
      await user.type(screen.getByLabelText(/^password/i), 'password123');
      await user.type(screen.getByLabelText(/confirm password/i), 'password123');
      await user.click(screen.getByRole('checkbox'));
      await user.click(screen.getByRole('button', { name: /create account/i }));
    }

    it('creates the Neon sign-in with the invite and terms, then asks for the email code', async () => {
      mockRegister.mockResolvedValue(undefined);
      renderAt(<RegisterPage />, '/register');
      await fillRegister('ABC-123');

      await waitFor(() =>
        expect(mockNavigate).toHaveBeenCalledWith('/verify-email?email=pat%40example.com')
      );
      expect(mockRegister).toHaveBeenCalledWith('pat@example.com', 'password123', {
        invitationToken: undefined,
        inviteCode: 'ABC-123',
        acceptTerms: true,
      });
      expect(screen.queryByText(/verification link/i)).not.toBeInTheDocument();
    });

    it('tells existing users they can leave the invite code empty', () => {
      renderAt(<RegisterPage />, '/register');
      expect(screen.getByText(/already use servicepro\? leave this empty/i)).toBeInTheDocument();
    });

    it('says when the email already has a sign-in', async () => {
      mockRegister.mockRejectedValue(
        new NeonAuthError('User already exists', 'user_already_exists', 422)
      );
      renderAt(<RegisterPage />, '/register');
      await fillRegister();
      expect(
        await screen.findByText(/already have a sign-in for this email/i)
      ).toBeInTheDocument();
      expect(mockNavigate).not.toHaveBeenCalled();
    });

    it('passes a tenant invitation token through to provisioning', async () => {
      vi.mocked(invitationApi.getByToken).mockResolvedValue({
        data: { email: 'pat@example.com', tenant_name: 'Acme', role_name: 'Technician' },
      } as never);
      mockRegister.mockResolvedValue(undefined);
      renderAt(<RegisterPage />, '/register?invitation=inv-token');
      await screen.findByText('Acme');
      const user = userEvent.setup();
      await user.type(screen.getByLabelText(/^password/i), 'password123');
      await user.type(screen.getByLabelText(/confirm password/i), 'password123');
      await user.click(screen.getByRole('checkbox'));
      await user.click(screen.getByRole('button', { name: /create account & join/i }));

      await waitFor(() => expect(mockRegister).toHaveBeenCalled());
      expect(mockRegister.mock.calls[0][2]).toMatchObject({ invitationToken: 'inv-token' });
    });

    it('Google sign-up needs the terms ticked and keeps the invite for after the redirect', async () => {
      neon.signInWithGoogle.mockReturnValue(new Promise(() => undefined));
      renderAt(<RegisterPage />, '/register?invite=XYZ-9');
      const user = userEvent.setup();
      const google = screen.getByRole('button', { name: /continue with google/i });

      await user.click(google);
      expect(
        await screen.findByText(/agree to the terms of service and privacy policy/i)
      ).toBeInTheDocument();
      expect(neon.signInWithGoogle).not.toHaveBeenCalled();

      await user.click(screen.getByRole('checkbox'));
      await user.click(google);
      await waitFor(() => expect(neon.signInWithGoogle).toHaveBeenCalled());
      expect(neon.savePendingSignup).toHaveBeenCalledWith({
        inviteCode: 'XYZ-9',
        invitationToken: undefined,
        acceptTerms: true,
      });
    });
  });

  describe('Verify email', () => {
    const path = '/verify-email?email=pat%40example.com';

    async function enterCode(code = '123456') {
      const user = userEvent.setup();
      await user.type(screen.getByLabelText(/verification code/i), code);
      await user.click(screen.getByRole('button', { name: /verify and continue/i }));
    }

    it('verifies the code, finishes sign-in and opens the app', async () => {
      neon.verifyEmailCode.mockResolvedValue(undefined);
      mockCompleteSignIn.mockResolvedValue(undefined);
      renderAt(<NeonVerifyEmailPage />, path);
      expect(screen.getByLabelText(/email address/i)).toHaveValue('pat@example.com');

      await enterCode();
      await waitFor(() => expect(mockNavigate).toHaveBeenCalledWith('/dashboard'));
      expect(neon.verifyEmailCode).toHaveBeenCalledWith('pat@example.com', '123456');
      expect(mockCompleteSignIn).toHaveBeenCalled();
    });

    it('shows a clear message for a wrong or expired code', async () => {
      neon.verifyEmailCode.mockRejectedValue(new NeonAuthError('Invalid OTP', 'validation_failed', 400));
      renderAt(<NeonVerifyEmailPage />, path);
      await enterCode('000000');
      expect(await screen.findByText(/wrong or has expired/i)).toBeInTheDocument();
      expect(mockCompleteSignIn).not.toHaveBeenCalled();
    });

    it('explains invite-only when the verified email has no invitation', async () => {
      neon.verifyEmailCode.mockResolvedValue(undefined);
      mockCompleteSignIn.mockRejectedValue(apiError(403, 'registration_closed'));
      renderAt(<NeonVerifyEmailPage />, path);
      await enterCode();
      expect(
        await screen.findByText(/no ServicePro account for this email yet/i)
      ).toBeInTheDocument();
      expect(mockNavigate).not.toHaveBeenCalled();
    });

    it('sends a new code', async () => {
      neon.sendVerificationCode.mockResolvedValue(undefined);
      renderAt(<NeonVerifyEmailPage />, path);
      await userEvent.setup().click(screen.getByRole('button', { name: /send a new code/i }));
      expect(await screen.findByText(/we sent a new code/i)).toBeInTheDocument();
      expect(neon.sendVerificationCode).toHaveBeenCalledWith('pat@example.com');
    });

    it('honours a safe returnUrl', async () => {
      neon.verifyEmailCode.mockResolvedValue(undefined);
      mockCompleteSignIn.mockResolvedValue(undefined);
      renderAt(<NeonVerifyEmailPage />, path + '&returnUrl=%2Finvitations%2Faccept%3Ftoken%3Dt');
      await enterCode();
      await waitFor(() =>
        expect(mockNavigate).toHaveBeenCalledWith('/invitations/accept?token=t')
      );
    });
  });

  describe('Password reset', () => {
    it('emails a reset code and moves on to entering it', async () => {
      neon.requestPasswordResetCode.mockResolvedValue(undefined);
      renderAt(<NeonForgotPasswordPage />, '/forgot-password');
      const user = userEvent.setup();
      await user.type(screen.getByLabelText(/email address/i), 'pat@example.com');
      await user.click(screen.getByRole('button', { name: /send reset code/i }));
      await waitFor(() =>
        expect(mockNavigate).toHaveBeenCalledWith('/reset-password?email=pat%40example.com')
      );
    });

    async function fillReset(code = '123456') {
      const user = userEvent.setup();
      await user.type(screen.getByLabelText(/reset code/i), code);
      await user.type(screen.getByLabelText(/^new password/i), 'new-password1');
      await user.type(screen.getByLabelText(/confirm new password/i), 'new-password1');
      await user.click(screen.getByRole('button', { name: /reset password/i }));
    }

    it('resets the password with the code', async () => {
      neon.resetPasswordWithCode.mockResolvedValue(undefined);
      renderAt(<NeonResetPasswordPage />, '/reset-password?email=pat%40example.com');
      await fillReset();
      expect(await screen.findByText(/password reset successful/i)).toBeInTheDocument();
      expect(neon.resetPasswordWithCode).toHaveBeenCalledWith(
        'pat@example.com',
        '123456',
        'new-password1'
      );
    });

    it('explains a wrong code', async () => {
      neon.resetPasswordWithCode.mockRejectedValue(
        new NeonAuthError('Invalid OTP', 'validation_failed', 400)
      );
      renderAt(<NeonResetPasswordPage />, '/reset-password?email=pat%40example.com');
      await fillReset('000000');
      expect(await screen.findByText(/wrong or has expired/i)).toBeInTheDocument();
    });
  });
});
