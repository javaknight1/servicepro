import { useEffect, useState, useRef } from 'react';
import { Building2, ChevronDown, Check, Plus } from 'lucide-react';
import { Link } from 'react-router-dom';
import { useTenantStore } from '@/store';
import type { Tenant } from '@/types/tenant';

export interface TenantSwitcherProps {
  className?: string;
}

export function TenantSwitcher({ className = '' }: TenantSwitcherProps) {
  const { currentTenant, tenants, isLoading, loadTenants, switchTenant } =
    useTenantStore();

  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Load tenants on mount
  useEffect(() => {
    if (tenants.length === 0) {
      loadTenants().catch(console.error);
    }
  }, [loadTenants, tenants.length]);

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

  const handleTenantSelect = async (tenant: Tenant) => {
    if (tenant.id !== currentTenant?.id) {
      await switchTenant(tenant.id);
      // Optionally reload data for the new tenant context
      window.location.reload();
    }
    setIsOpen(false);
  };

  if (isLoading && tenants.length === 0) {
    return (
      <div
        className={`flex items-center px-3 py-2 text-sm text-neutral-500 ${className}`}
      >
        <Building2 className="h-4 w-4 mr-2 animate-pulse" />
        Loading...
      </div>
    );
  }

  if (tenants.length === 0) {
    return (
      <Link
        to="/settings/organization/new"
        className={`flex items-center px-3 py-2 text-sm text-primary-600 hover:text-primary-700 ${className}`}
      >
        <Plus className="h-4 w-4 mr-2" />
        Create Organization
      </Link>
    );
  }

  return (
    <div ref={dropdownRef} className={`relative ${className}`}>
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className="flex items-center px-3 py-2 text-sm font-medium text-neutral-700 hover:text-neutral-900 rounded-md hover:bg-neutral-100 transition-colors"
      >
        <Building2 className="h-4 w-4 mr-2 text-neutral-500" />
        <span className="max-w-[150px] truncate">
          {currentTenant?.name || 'Select Organization'}
        </span>
        <ChevronDown
          className={`h-4 w-4 ml-2 text-neutral-400 transition-transform ${
            isOpen ? 'rotate-180' : ''
          }`}
        />
      </button>

      {isOpen && (
        <div className="absolute left-0 mt-1 w-64 bg-white rounded-lg shadow-lg border border-neutral-200 py-1 z-50">
          <div className="px-3 py-2 text-xs font-semibold text-neutral-500 uppercase tracking-wider">
            Organizations
          </div>

          <div className="max-h-60 overflow-y-auto">
            {tenants.map((tenant) => (
              <button
                key={tenant.id}
                type="button"
                onClick={() => handleTenantSelect(tenant)}
                className={`w-full flex items-center px-3 py-2 text-sm hover:bg-neutral-50 ${
                  currentTenant?.id === tenant.id ? 'bg-primary-50' : ''
                }`}
              >
                <div className="flex-1 text-left">
                  <div className="font-medium text-neutral-900">
                    {tenant.name}
                  </div>
                  {tenant.role && (
                    <div className="text-xs text-neutral-500">
                      {tenant.role}
                    </div>
                  )}
                </div>
                {currentTenant?.id === tenant.id && (
                  <Check className="h-4 w-4 text-primary-600" />
                )}
              </button>
            ))}
          </div>

          <div className="border-t border-neutral-200 mt-1 pt-1">
            <Link
              to="/settings/organization/new"
              onClick={() => setIsOpen(false)}
              className="flex items-center px-3 py-2 text-sm text-primary-600 hover:bg-neutral-50"
            >
              <Plus className="h-4 w-4 mr-2" />
              Create Organization
            </Link>
          </div>
        </div>
      )}
    </div>
  );
}

export default TenantSwitcher;
