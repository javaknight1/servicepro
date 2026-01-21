import { Navigate, Outlet } from 'react-router-dom';
import { useAuthStore } from '@store';

/**
 * PublicRoute - redirects to dashboard if already authenticated
 * Used for login, register pages
 */
export function PublicRoute() {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);

  if (isAuthenticated) {
    return <Navigate to="/dashboard" replace />;
  }

  return <Outlet />;
}
