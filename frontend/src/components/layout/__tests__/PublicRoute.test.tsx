import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { PublicRoute } from '../PublicRoute';
import { useAuthStore } from '@/store';

// Mock the auth store
vi.mock('@/store', () => ({
  useAuthStore: vi.fn(),
}));

describe('PublicRoute', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should redirect to dashboard when authenticated', () => {
    vi.mocked(useAuthStore).mockImplementation((selector) => {
      const state = {
        isAuthenticated: true,
      };
      return selector(state as ReturnType<typeof useAuthStore.getState>);
    });

    render(
      <MemoryRouter initialEntries={['/login']}>
        <Routes>
          <Route path="/dashboard" element={<div>Dashboard Page</div>} />
          <Route element={<PublicRoute />}>
            <Route path="/login" element={<div>Login Page</div>} />
          </Route>
        </Routes>
      </MemoryRouter>
    );

    expect(screen.getByText('Dashboard Page')).toBeInTheDocument();
    expect(screen.queryByText('Login Page')).not.toBeInTheDocument();
  });

  it('should render children when not authenticated', () => {
    vi.mocked(useAuthStore).mockImplementation((selector) => {
      const state = {
        isAuthenticated: false,
      };
      return selector(state as ReturnType<typeof useAuthStore.getState>);
    });

    render(
      <MemoryRouter initialEntries={['/login']}>
        <Routes>
          <Route path="/dashboard" element={<div>Dashboard Page</div>} />
          <Route element={<PublicRoute />}>
            <Route path="/login" element={<div>Login Page</div>} />
          </Route>
        </Routes>
      </MemoryRouter>
    );

    expect(screen.getByText('Login Page')).toBeInTheDocument();
    expect(screen.queryByText('Dashboard Page')).not.toBeInTheDocument();
  });
});
