import { useEffect, useState } from 'react';
import { Navigate, Outlet } from 'react-router-dom';
import { useAuthStore } from '@store';
import { Loader2 } from 'lucide-react';

export function ProtectedRoute() {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const user = useAuthStore((state) => state.user);
  const checkAuth = useAuthStore((state) => state.checkAuth);
  const [isVerifying, setIsVerifying] = useState(!user);

  useEffect(() => {
    // Verify the session is still valid with the backend
    if (isAuthenticated) {
      setIsVerifying(true);
      checkAuth().finally(() => setIsVerifying(false));
    }
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }

  if (isVerifying) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <Loader2 className="h-8 w-8 animate-spin text-primary-500" />
      </div>
    );
  }

  return <Outlet />;
}
