import { useState, useRef, useEffect } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { LogOut, Settings, User } from 'lucide-react';
import { useAuthStore, useTenantStore } from '@store';
import { Button } from '@components/shared';
import { TenantSwitcher } from '@components/tenants';

function ProfileDropdown() {
  const navigate = useNavigate();
  const { user, logout } = useAuthStore();
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Close dropdown when clicking outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(event.target as Node)
      ) {
        setIsOpen(false);
      }
    }
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleSettingsClick = () => {
    setIsOpen(false);
    navigate('/settings');
  };

  const handleLogout = () => {
    setIsOpen(false);
    logout();
  };

  // Get initials for avatar
  const getInitials = () => {
    if (user?.email) {
      return user.email.charAt(0).toUpperCase();
    }
    return 'U';
  };

  return (
    <div ref={dropdownRef} className="relative">
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center justify-center h-9 w-9 rounded-full bg-primary-100 text-primary-700 font-medium text-sm hover:bg-primary-200 transition-colors focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2"
        title={user?.email || 'Account'}
      >
        {getInitials()}
      </button>

      {isOpen && (
        <div className="absolute right-0 mt-2 w-56 bg-white rounded-lg shadow-lg border border-neutral-200 py-1 z-50">
          {user?.email && (
            <div className="px-4 py-3 border-b border-neutral-200">
              <p className="text-sm font-medium text-neutral-900 truncate">
                {user.email}
              </p>
            </div>
          )}

          <button
            onClick={handleSettingsClick}
            className="w-full flex items-center px-4 py-2 text-sm text-neutral-700 hover:bg-neutral-50"
          >
            <Settings className="h-4 w-4 mr-3 text-neutral-400" />
            Account Settings
          </button>

          <div className="border-t border-neutral-200 my-1" />

          <button
            onClick={handleLogout}
            className="w-full flex items-center px-4 py-2 text-sm text-red-600 hover:bg-red-50"
          >
            <LogOut className="h-4 w-4 mr-3" />
            Log out
          </button>
        </div>
      )}
    </div>
  );
}

export function Header() {
  const { isAuthenticated } = useAuthStore();
  const { tenants } = useTenantStore();

  const hasOrganizations = tenants.length > 0;

  return (
    <header className="bg-white border-b border-neutral-200">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex justify-between items-center h-16">
          <div className="flex items-center">
            <Link
              to={isAuthenticated ? '/dashboard' : '/'}
              className="text-2xl font-bold text-primary-600"
            >
              ServicePro
            </Link>
          </div>

          <nav className="flex items-center space-x-4">
            {isAuthenticated ? (
              <>
                {hasOrganizations && (
                  <>
                    <TenantSwitcher />
                    <div className="h-6 w-px bg-neutral-200" />
                  </>
                )}
                <ProfileDropdown />
              </>
            ) : (
              <>
                <Link
                  to="/login"
                  className="text-neutral-700 hover:text-neutral-900 px-3 py-2 rounded-md text-sm font-medium"
                >
                  Login
                </Link>
                <Link to="/register">
                  <Button variant="primary" size="sm">
                    Get Started
                  </Button>
                </Link>
              </>
            )}
          </nav>
        </div>
      </div>
    </header>
  );
}
