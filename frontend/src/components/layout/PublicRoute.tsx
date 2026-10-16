import { Navigate, Outlet, useSearchParams } from 'react-router-dom';
import { useAuthStore } from '@store';
import { getSafeReturnUrl } from '@/utils/urlSanitizer';

/**
 * PublicRoute - redirects to dashboard (or a safe same-origin `returnUrl`,
 * e.g. an invitation-accept link) if already authenticated.
 * Used for login, register pages
 */
export function PublicRoute() {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const [searchParams] = useSearchParams();

  if (isAuthenticated) {
    const returnUrl = getSafeReturnUrl(searchParams.get('returnUrl'));
    return <Navigate to={returnUrl ?? '/dashboard'} replace />;
  }

  return <Outlet />;
}
