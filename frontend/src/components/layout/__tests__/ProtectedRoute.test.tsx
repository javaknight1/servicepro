import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { ProtectedRoute } from '../ProtectedRoute';
import { useAuthStore } from '@/store';

// Mock the auth store
vi.mock('@/store', () => ({
  useAuthStore: vi.fn(),
}));

type AuthState = ReturnType<typeof useAuthStore.getState>;

function mockAuth(state: Partial<AuthState>): void {
  vi.mocked(useAuthStore).mockImplementation((selector) =>
    selector(state as AuthState)
  );
}

function renderProtected() {
  return render(
    <MemoryRouter initialEntries={['/protected']}>
      <Routes>
        <Route path="/login" element={<div>Login Page</div>} />
        <Route element={<ProtectedRoute />}>
          <Route path="/protected" element={<div>Protected Content</div>} />
        </Route>
      </Routes>
    </MemoryRouter>
  );
}

describe('ProtectedRoute', () => {
  const mockCheckAuth = vi.fn<() => Promise<boolean>>();

  beforeEach(() => {
    vi.clearAllMocks();
    mockCheckAuth.mockResolvedValue(true);
  });

  it('should redirect to login when not authenticated', () => {
    mockAuth({ isAuthenticated: false, user: null, checkAuth: mockCheckAuth });

    renderProtected();

    expect(screen.getByText('Login Page')).toBeInTheDocument();
    expect(screen.queryByText('Protected Content')).not.toBeInTheDocument();
    expect(mockCheckAuth).not.toHaveBeenCalled();
  });

  it('should verify the session and render children when authenticated', async () => {
    mockAuth({
      isAuthenticated: true,
      user: { id: 'user-1', email: 'test@example.com' } as AuthState['user'],
      checkAuth: mockCheckAuth,
    });

    renderProtected();

    expect(await screen.findByText('Protected Content')).toBeInTheDocument();
    expect(screen.queryByText('Login Page')).not.toBeInTheDocument();
    expect(mockCheckAuth).toHaveBeenCalledTimes(1);
  });

  it('should show a loader until the session check completes', async () => {
    let resolveCheck: (value: boolean) => void = () => undefined;
    mockCheckAuth.mockReturnValue(
      new Promise<boolean>((resolve) => {
        resolveCheck = resolve;
      })
    );
    mockAuth({ isAuthenticated: true, user: null, checkAuth: mockCheckAuth });

    renderProtected();

    expect(screen.queryByText('Protected Content')).not.toBeInTheDocument();
    resolveCheck(true);
    expect(await screen.findByText('Protected Content')).toBeInTheDocument();
  });
});
