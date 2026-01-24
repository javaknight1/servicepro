import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import { LoginPage } from '../LoginPage';

// Mock the auth store
const mockLogin = vi.fn();
vi.mock('@store', () => ({
  useAuthStore: vi.fn((selector) => {
    if (typeof selector === 'function') {
      return selector({ login: mockLogin });
    }
    return { login: mockLogin };
  }),
  useTenantStore: vi.fn((selector) => {
    const state = {
      currentTenant: null,
      tenants: [],
      setCurrentTenant: vi.fn(),
      fetchTenants: vi.fn(),
    };
    if (typeof selector === 'function') {
      return selector(state);
    }
    return state;
  }),
}));

// Mock useNavigate
const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

// Helper to render with router context
const renderWithRouter = (ui: React.ReactElement) => {
  return render(<BrowserRouter>{ui}</BrowserRouter>);
};

describe('LoginPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // ==========================================================================
  // Rendering Tests
  // ==========================================================================
  describe('Rendering', () => {
    it('renders login form', () => {
      renderWithRouter(<LoginPage />);

      expect(
        screen.getByRole('heading', { name: /sign in/i })
      ).toBeInTheDocument();
      expect(screen.getByLabelText(/email/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/password/i)).toBeInTheDocument();
      expect(
        screen.getByRole('button', { name: /sign in/i })
      ).toBeInTheDocument();
    });

    it('renders link to register page', () => {
      renderWithRouter(<LoginPage />);
      expect(
        screen.getByRole('link', { name: /create a new account/i })
      ).toHaveAttribute('href', '/register');
    });

    it('renders forgot password link', () => {
      renderWithRouter(<LoginPage />);
      expect(
        screen.getByRole('link', { name: /forgot your password/i })
      ).toHaveAttribute('href', '/forgot-password');
    });
  });

  // ==========================================================================
  // Form Validation Tests
  // ==========================================================================
  describe('Form Validation', () => {
    it('shows error for empty email', async () => {
      const user = userEvent.setup();
      renderWithRouter(<LoginPage />);

      await user.click(screen.getByRole('button', { name: /sign in/i }));

      await waitFor(() => {
        expect(screen.getByText(/invalid email/i)).toBeInTheDocument();
      });
    });

    // Note: HTML5 type="email" validation blocks invalid formats before form submission,
    // so custom validation errors cannot be tested for these cases in browser tests.
    it.skip('shows error for invalid email format', async () => {
      const user = userEvent.setup();
      renderWithRouter(<LoginPage />);

      await user.type(screen.getByLabelText(/email/i), 'notanemail');
      await user.click(screen.getByRole('button', { name: /sign in/i }));

      await waitFor(() => {
        expect(screen.getByText(/invalid email/i)).toBeInTheDocument();
      });
    });

    it('shows error for short password', async () => {
      const user = userEvent.setup();
      renderWithRouter(<LoginPage />);

      await user.type(screen.getByLabelText(/email/i), 'test@example.com');
      await user.type(screen.getByLabelText(/password/i), 'short');
      await user.click(screen.getByRole('button', { name: /sign in/i }));

      await waitFor(() => {
        expect(
          screen.getByText(/password must be at least 8 characters/i)
        ).toBeInTheDocument();
      });
    });

    it('shows error for empty password', async () => {
      const user = userEvent.setup();
      renderWithRouter(<LoginPage />);

      await user.type(screen.getByLabelText(/email/i), 'test@example.com');
      await user.click(screen.getByRole('button', { name: /sign in/i }));

      await waitFor(() => {
        expect(
          screen.getByText(/password must be at least 8 characters/i)
        ).toBeInTheDocument();
      });
    });

    it('does not show errors for valid inputs', async () => {
      const user = userEvent.setup();
      mockLogin.mockResolvedValueOnce({});
      renderWithRouter(<LoginPage />);

      await user.type(screen.getByLabelText(/email/i), 'test@example.com');
      await user.type(screen.getByLabelText(/password/i), 'password123');
      await user.click(screen.getByRole('button', { name: /sign in/i }));

      await waitFor(() => {
        expect(screen.queryByText(/invalid email/i)).not.toBeInTheDocument();
        expect(
          screen.queryByText(/password must be at least/i)
        ).not.toBeInTheDocument();
      });
    });
  });

  // ==========================================================================
  // Email Validation Tests - Table Driven
  // ==========================================================================
  // Note: Some invalid email formats are blocked by HTML5 browser validation
  // before our custom validation can show errors. Only emails that pass HTML5
  // validation but fail our custom validation can be tested here.
  describe('Email Validation', () => {
    // These emails pass HTML5 validation but should fail our custom validation
    const invalidEmails = [
      { email: '', error: 'Invalid email address' },
      { email: 'missing@domain', error: 'Invalid email address' },
    ];

    // These are blocked by HTML5 validation, cannot be tested in browser
    // { email: 'plaintext', error: 'Invalid email address' },
    // { email: '@nodomain.com', error: 'Invalid email address' },
    // { email: 'spaces in@email.com', error: 'Invalid email address' },

    invalidEmails.forEach(({ email, error }) => {
      it(`shows error for invalid email: "${email || '(empty)'}"`, async () => {
        const user = userEvent.setup();
        renderWithRouter(<LoginPage />);

        if (email) {
          await user.type(screen.getByLabelText(/email/i), email);
        }
        await user.click(screen.getByRole('button', { name: /sign in/i }));

        await waitFor(() => {
          expect(screen.getByText(new RegExp(error, 'i'))).toBeInTheDocument();
        });
      });
    });

    const validEmails = [
      'user@example.com',
      'user.name@example.co.uk',
      'user+tag@example.org',
    ];

    validEmails.forEach((email) => {
      it(`accepts valid email: "${email}"`, async () => {
        const user = userEvent.setup();
        mockLogin.mockResolvedValueOnce({});
        renderWithRouter(<LoginPage />);

        await user.type(screen.getByLabelText(/email/i), email);
        await user.type(screen.getByLabelText(/password/i), 'password123');
        await user.click(screen.getByRole('button', { name: /sign in/i }));

        await waitFor(() => {
          expect(screen.queryByText(/invalid email/i)).not.toBeInTheDocument();
        });
      });
    });
  });

  // ==========================================================================
  // Password Validation Tests - Table Driven
  // ==========================================================================
  describe('Password Validation', () => {
    const invalidPasswords = [
      { password: '', length: 0 },
      { password: '1234567', length: 7 },
      { password: 'short', length: 5 },
    ];

    invalidPasswords.forEach(({ password, length }) => {
      it(`shows error for password with ${length} characters`, async () => {
        const user = userEvent.setup();
        renderWithRouter(<LoginPage />);

        await user.type(screen.getByLabelText(/email/i), 'test@example.com');
        if (password) {
          await user.type(screen.getByLabelText(/password/i), password);
        }
        await user.click(screen.getByRole('button', { name: /sign in/i }));

        await waitFor(() => {
          expect(
            screen.getByText(/password must be at least 8 characters/i)
          ).toBeInTheDocument();
        });
      });
    });

    const validPasswords = ['12345678', 'longerpassword', 'Password123!'];

    validPasswords.forEach((password) => {
      it(`accepts valid password with ${password.length} characters`, async () => {
        const user = userEvent.setup();
        mockLogin.mockResolvedValueOnce({});
        renderWithRouter(<LoginPage />);

        await user.type(screen.getByLabelText(/email/i), 'test@example.com');
        await user.type(screen.getByLabelText(/password/i), password);
        await user.click(screen.getByRole('button', { name: /sign in/i }));

        await waitFor(() => {
          expect(
            screen.queryByText(/password must be at least/i)
          ).not.toBeInTheDocument();
        });
      });
    });
  });

  // ==========================================================================
  // Form Submission Tests
  // ==========================================================================
  describe('Form Submission', () => {
    it('calls login with correct credentials', async () => {
      const user = userEvent.setup();
      mockLogin.mockResolvedValueOnce({});
      renderWithRouter(<LoginPage />);

      await user.type(screen.getByLabelText(/email/i), 'test@example.com');
      await user.type(screen.getByLabelText(/password/i), 'password123');
      await user.click(screen.getByRole('button', { name: /sign in/i }));

      await waitFor(() => {
        expect(mockLogin).toHaveBeenCalledWith(
          'test@example.com',
          'password123'
        );
      });
    });

    it('navigates to dashboard on successful login', async () => {
      const user = userEvent.setup();
      mockLogin.mockResolvedValueOnce({});
      renderWithRouter(<LoginPage />);

      await user.type(screen.getByLabelText(/email/i), 'test@example.com');
      await user.type(screen.getByLabelText(/password/i), 'password123');
      await user.click(screen.getByRole('button', { name: /sign in/i }));

      await waitFor(() => {
        expect(mockNavigate).toHaveBeenCalledWith('/dashboard');
      });
    });

    it('does not call login when form is invalid', async () => {
      const user = userEvent.setup();
      renderWithRouter(<LoginPage />);

      await user.type(screen.getByLabelText(/email/i), 'invalid');
      await user.click(screen.getByRole('button', { name: /sign in/i }));

      await waitFor(() => {
        expect(mockLogin).not.toHaveBeenCalled();
      });
    });
  });

  // ==========================================================================
  // Error Handling Tests
  // ==========================================================================
  describe('Error Handling', () => {
    it('displays error message on login failure', async () => {
      const user = userEvent.setup();
      mockLogin.mockRejectedValueOnce({
        response: {
          data: {
            message: 'Invalid credentials',
          },
        },
      });
      renderWithRouter(<LoginPage />);

      await user.type(screen.getByLabelText(/email/i), 'test@example.com');
      await user.type(screen.getByLabelText(/password/i), 'password123');
      await user.click(screen.getByRole('button', { name: /sign in/i }));

      await waitFor(() => {
        expect(screen.getByText(/invalid credentials/i)).toBeInTheDocument();
      });
    });

    it('displays default error message when no message in response', async () => {
      const user = userEvent.setup();
      mockLogin.mockRejectedValueOnce({});
      renderWithRouter(<LoginPage />);

      await user.type(screen.getByLabelText(/email/i), 'test@example.com');
      await user.type(screen.getByLabelText(/password/i), 'password123');
      await user.click(screen.getByRole('button', { name: /sign in/i }));

      await waitFor(() => {
        expect(screen.getByText(/failed to login/i)).toBeInTheDocument();
      });
    });

    it('clears error on new submission attempt', async () => {
      const user = userEvent.setup();
      mockLogin.mockRejectedValueOnce({
        response: { data: { message: 'First error' } },
      });
      mockLogin.mockResolvedValueOnce({});
      renderWithRouter(<LoginPage />);

      // First submission - fails
      await user.type(screen.getByLabelText(/email/i), 'test@example.com');
      await user.type(screen.getByLabelText(/password/i), 'password123');
      await user.click(screen.getByRole('button', { name: /sign in/i }));

      await waitFor(() => {
        expect(screen.getByText(/first error/i)).toBeInTheDocument();
      });

      // Second submission - succeeds
      await user.click(screen.getByRole('button', { name: /sign in/i }));

      await waitFor(() => {
        expect(screen.queryByText(/first error/i)).not.toBeInTheDocument();
      });
    });
  });

  // ==========================================================================
  // Loading State Tests
  // ==========================================================================
  describe('Loading State', () => {
    it('shows loading state during submission', async () => {
      const user = userEvent.setup();
      mockLogin.mockImplementation(
        () => new Promise((resolve) => setTimeout(resolve, 100))
      );
      renderWithRouter(<LoginPage />);

      await user.type(screen.getByLabelText(/email/i), 'test@example.com');
      await user.type(screen.getByLabelText(/password/i), 'password123');
      await user.click(screen.getByRole('button', { name: /sign in/i }));

      // Button should be disabled and show loading
      expect(screen.getByRole('button', { name: /sign in/i })).toBeDisabled();
    });

    it('enables button after submission completes', async () => {
      const user = userEvent.setup();
      mockLogin.mockResolvedValueOnce({});
      renderWithRouter(<LoginPage />);

      await user.type(screen.getByLabelText(/email/i), 'test@example.com');
      await user.type(screen.getByLabelText(/password/i), 'password123');
      await user.click(screen.getByRole('button', { name: /sign in/i }));

      await waitFor(() => {
        expect(
          screen.getByRole('button', { name: /sign in/i })
        ).not.toBeDisabled();
      });
    });
  });

  // ==========================================================================
  // Accessibility Tests
  // ==========================================================================
  describe('Accessibility', () => {
    it('has accessible form labels', () => {
      renderWithRouter(<LoginPage />);

      expect(screen.getByLabelText(/email/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/password/i)).toBeInTheDocument();
    });

    it('form inputs have correct types', () => {
      renderWithRouter(<LoginPage />);

      expect(screen.getByLabelText(/email/i)).toHaveAttribute('type', 'email');
      expect(screen.getByLabelText(/password/i)).toHaveAttribute(
        'type',
        'password'
      );
    });

    it('form inputs have autocomplete attributes', () => {
      renderWithRouter(<LoginPage />);

      expect(screen.getByLabelText(/email/i)).toHaveAttribute(
        'autocomplete',
        'email'
      );
      expect(screen.getByLabelText(/password/i)).toHaveAttribute(
        'autocomplete',
        'current-password'
      );
    });

    it('error messages are displayed near inputs', async () => {
      const user = userEvent.setup();
      renderWithRouter(<LoginPage />);

      await user.click(screen.getByRole('button', { name: /sign in/i }));

      await waitFor(() => {
        // Both error messages should be visible
        expect(screen.getByText(/invalid email/i)).toBeInTheDocument();
        expect(
          screen.getByText(/password must be at least/i)
        ).toBeInTheDocument();
      });
    });
  });

  // ==========================================================================
  // Keyboard Navigation Tests
  // ==========================================================================
  describe('Keyboard Navigation', () => {
    it('supports tab navigation through form', async () => {
      const user = userEvent.setup();
      renderWithRouter(<LoginPage />);

      // Focus the email input directly and then test tab navigation
      const emailInput = screen.getByLabelText(/email/i);
      emailInput.focus();
      expect(emailInput).toHaveFocus();

      await user.tab();
      expect(screen.getByLabelText(/password/i)).toHaveFocus();
    });

    it('submits form on Enter in password field', async () => {
      const user = userEvent.setup();
      mockLogin.mockResolvedValueOnce({});
      renderWithRouter(<LoginPage />);

      await user.type(screen.getByLabelText(/email/i), 'test@example.com');
      await user.type(screen.getByLabelText(/password/i), 'password123');
      await user.keyboard('{Enter}');

      await waitFor(() => {
        expect(mockLogin).toHaveBeenCalled();
      });
    });
  });
});
