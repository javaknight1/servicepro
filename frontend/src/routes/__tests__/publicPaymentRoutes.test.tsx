import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { AppRoutes, PUBLIC_PAYMENT_PATHS, isProtectedRoute } from '../index';
import { useAuthStore } from '@store';

describe('public Stripe Checkout return routes', () => {
  beforeEach(() => {
    useAuthStore.setState({ isAuthenticated: false, user: null });
  });

  it('renders payment success for an unauthenticated payer', async () => {
    render(
      <MemoryRouter
        initialEntries={[`${PUBLIC_PAYMENT_PATHS.success}?session_id=cs_123`]}
      >
        <AppRoutes />
      </MemoryRouter>
    );

    expect(
      await screen.findByText(/payment successful/i, {}, { timeout: 3000 })
    ).toBeInTheDocument();
  });

  it('renders payment cancelled for an unauthenticated payer', async () => {
    render(
      <MemoryRouter initialEntries={[PUBLIC_PAYMENT_PATHS.cancelled]}>
        <AppRoutes />
      </MemoryRouter>
    );

    expect(
      await screen.findByText(/payment cancelled/i, {}, { timeout: 3000 })
    ).toBeInTheDocument();
  });

  it('does not treat the payment return pages as protected', () => {
    expect(isProtectedRoute(PUBLIC_PAYMENT_PATHS.success)).toBe(false);
    expect(isProtectedRoute(PUBLIC_PAYMENT_PATHS.cancelled)).toBe(false);
    expect(isProtectedRoute('/invoices/abc')).toBe(true);
  });
});
