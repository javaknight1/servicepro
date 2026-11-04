/**
 * LoginPage on a Neon Auth build (T158): email-verification hand-off,
 * invite-only refusal, Google sign-in and its callback.
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

const { mockLogin, mockCompleteSignIn, neon } = vi.hoisted(() => ({
  mockLogin: vi.fn(),
  mockCompleteSignIn: vi.fn(),
  neon: { sendVerificationCode: vi.fn(), signInWithGoogle: vi.fn() },
}));

vi.mock('@store', () => {
  const pick = (state: object) => (selector?: (s: object) => unknown) =>
    typeof selector === 'function' ? selector(state) : state;
  return {
    useAuthStore: vi.fn(
      pick({ login: mockLogin, completeSignIn: mockCompleteSignIn })
    ),
    useTenantStore: vi.fn(
      pick({ currentTenant: null, tenants: [], setCurrentTenant: vi.fn(), fetchTenants: vi.fn() })
    ),
  };
});
vi.mock('@services/neonAuth', () => neon);

const mockNavigate = vi.fn();
vi.mock('react-router', async () => {
  const actual = await vi.importActual('react-router');
  return { ...actual, useNavigate: () => mockNavigate };
});

import { LoginPage } from '../LoginPage';

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

function renderAt(path = '/login') {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <LoginPage />
    </MemoryRouter>
  );
}

async function submit(email = 'pat@example.com', password = 'password123') {
  const user = userEvent.setup();
  await user.type(screen.getByLabelText(/email/i), email);
  await user.type(screen.getByLabelText(/^password/i), password);
  await user.click(screen.getByRole('button', { name: /^sign in$/i }));
}

describe('LoginPage (Neon Auth)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('signs in and goes to the dashboard', async () => {
    mockLogin.mockResolvedValue(undefined);
    renderAt();
    await submit();
    await waitFor(() => expect(mockNavigate).toHaveBeenCalledWith('/dashboard'));
  });

  it('shows our message for a wrong password', async () => {
    mockLogin.mockRejectedValue(
      new NeonAuthError('Invalid email or password', 'invalid_credentials', 401)
    );
    renderAt();
    await submit();
    expect(await screen.findByText('Invalid email or password.')).toBeInTheDocument();
  });

  it('sends a verification code and asks for it when the email is unverified', async () => {
    mockLogin.mockRejectedValue(apiError(403, 'email_not_verified'));
    neon.sendVerificationCode.mockResolvedValue(undefined);
    renderAt('/login?returnUrl=%2Fjobs');
    await submit();
    await waitFor(() =>
      expect(mockNavigate).toHaveBeenCalledWith(
        '/verify-email?email=pat%40example.com&returnUrl=%2Fjobs'
      )
    );
    expect(neon.sendVerificationCode).toHaveBeenCalledWith('pat@example.com');
  });

  it('explains invite-only when there is no ServicePro account', async () => {
    mockLogin.mockRejectedValue(apiError(403, 'registration_closed'));
    renderAt();
    await submit();
    expect(
      await screen.findByText(/no ServicePro account for this email yet/i)
    ).toBeInTheDocument();
  });

  it('points existing users at setting up their new sign-in', () => {
    renderAt();
    expect(
      screen.getByRole('link', { name: /set up your sign-in/i })
    ).toHaveAttribute('href', '/register');
  });

  it('starts Google sign-in with /login?oauth=1 as the callback', async () => {
    neon.signInWithGoogle.mockReturnValue(new Promise(() => undefined));
    renderAt();
    await userEvent.setup().click(
      screen.getByRole('button', { name: /continue with google/i })
    );
    await waitFor(() => expect(neon.signInWithGoogle).toHaveBeenCalled());
    const [callback, errorCallback] = neon.signInWithGoogle.mock.calls[0];
    expect(new URL(callback).pathname).toBe('/login');
    expect(new URL(callback).searchParams.get('oauth')).toBe('1');
    expect(new URL(errorCallback).searchParams.get('oauth')).toBe('error');
  });

  it('finishes the sign-in when Google sends the user back', async () => {
    mockCompleteSignIn.mockResolvedValue(undefined);
    renderAt('/login?oauth=1&returnUrl=%2Fjobs');
    expect(screen.getByRole('status')).toHaveTextContent(/signing you in/i);
    await waitFor(() => expect(mockNavigate).toHaveBeenCalledWith('/jobs'));
    expect(mockCompleteSignIn).toHaveBeenCalledTimes(1);
  });

  it('shows why a Google sign-in was refused', async () => {
    mockCompleteSignIn.mockRejectedValue(apiError(403, 'registration_closed'));
    renderAt('/login?oauth=1');
    expect(
      await screen.findByText(/no ServicePro account for this email yet/i)
    ).toBeInTheDocument();
    expect(mockNavigate).not.toHaveBeenCalled();
  });

  it('reports a cancelled Google sign-in', async () => {
    renderAt('/login?oauth=error');
    expect(
      await screen.findByText(/google sign-in was cancelled or failed/i)
    ).toBeInTheDocument();
    expect(mockCompleteSignIn).not.toHaveBeenCalled();
  });
});
