import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { ProtectedRoute } from '../ProtectedRoute';
import { useAuthStore } from '@/store';

// Mock the auth store
vi.mock('@/store', () => ({
  useAuthStore: vi.fn(),
}));

describe('ProtectedRoute', () => {
  const mockFetchCurrentUser = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should redirect to login when not authenticated', () => {
    vi.mocked(useAuthStore).mockImplementation((selector) => {
      const state = {
        isAuthenticated: false,
        user: null,
        fetchCurrentUser: mockFetchCurrentUser,
      };
      return selector(state as ReturnType<typeof useAuthStore.getState>);
    });

    render(
      <MemoryRouter initialEntries={['/protected']}>
        <Routes>
          <Route path="/login" element={<div>Login Page</div>} />
          <Route element={<ProtectedRoute />}>
            <Route path="/protected" element={<div>Protected Content</div>} />
          </Route>
        </Routes>
      </MemoryRouter>
    );

    expect(screen.getByText('Login Page')).toBeInTheDocument();
    expect(screen.queryByText('Protected Content')).not.toBeInTheDocument();
  });

  it('should render children when authenticated', () => {
    vi.mocked(useAuthStore).mockImplementation((selector) => {
      const state = {
        isAuthenticated: true,
        user: { id: 'user-1', email: 'test@example.com' },
        fetchCurrentUser: mockFetchCurrentUser,
      };
      return selector(state as ReturnType<typeof useAuthStore.getState>);
    });

    render(
      <MemoryRouter initialEntries={['/protected']}>
        <Routes>
          <Route path="/login" element={<div>Login Page</div>} />
          <Route element={<ProtectedRoute />}>
            <Route path="/protected" element={<div>Protected Content</div>} />
          </Route>
        </Routes>
      </MemoryRouter>
    );

    expect(screen.getByText('Protected Content')).toBeInTheDocument();
    expect(screen.queryByText('Login Page')).not.toBeInTheDocument();
  });

  it('should fetch current user when authenticated but no user data', () => {
    vi.mocked(useAuthStore).mockImplementation((selector) => {
      const state = {
        isAuthenticated: true,
        user: null,
        fetchCurrentUser: mockFetchCurrentUser,
      };
      return selector(state as ReturnType<typeof useAuthStore.getState>);
    });

    render(
      <MemoryRouter initialEntries={['/protected']}>
        <Routes>
          <Route element={<ProtectedRoute />}>
            <Route path="/protected" element={<div>Protected Content</div>} />
          </Route>
        </Routes>
      </MemoryRouter>
    );

    expect(mockFetchCurrentUser).toHaveBeenCalled();
  });

  it('should not fetch current user when already has user data', () => {
    vi.mocked(useAuthStore).mockImplementation((selector) => {
      const state = {
        isAuthenticated: true,
        user: { id: 'user-1', email: 'test@example.com' },
        fetchCurrentUser: mockFetchCurrentUser,
      };
      return selector(state as ReturnType<typeof useAuthStore.getState>);
    });

    render(
      <MemoryRouter initialEntries={['/protected']}>
        <Routes>
          <Route element={<ProtectedRoute />}>
            <Route path="/protected" element={<div>Protected Content</div>} />
          </Route>
        </Routes>
      </MemoryRouter>
    );

    expect(mockFetchCurrentUser).not.toHaveBeenCalled();
  });
});
