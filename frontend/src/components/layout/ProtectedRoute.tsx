import { useEffect } from 'react';
import { Navigate, Outlet } from 'react-router-dom';
import { useAuthStore } from '@store';

export function ProtectedRoute() {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const user = useAuthStore((state) => state.user);
  const fetchCurrentUser = useAuthStore((state) => state.fetchCurrentUser);

  useEffect(() => {
    // If authenticated but no user data, fetch it
    if (isAuthenticated && !user) {
      fetchCurrentUser();
    }
  }, [isAuthenticated, user, fetchCurrentUser]);

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  return <Outlet />;
}
