import { useState, useEffect } from 'react';
import { NavLink, useLocation } from 'react-router-dom';
import { cn } from '@utils/cn';
import { useSidebarStore } from '@store/sidebarStore';
import { useTenantStore } from '@store/tenantStore';
import {
  LayoutDashboard,
  Users,
  Briefcase,
  FileText,
  Receipt,
  BarChart3,
  Settings,
  ChevronLeft,
  ChevronRight,
  ChevronDown,
  X,
  Menu,
  UserCog,
} from 'lucide-react';

interface NavItem {
  label: string;
  path?: string;
  icon: React.ElementType;
  children?: { label: string; path: string }[];
}

// Items that are always visible
const baseNavItems: NavItem[] = [
  { label: 'Dashboard', path: '/dashboard', icon: LayoutDashboard },
];

// Items that require an organization to be selected (includes Settings -> Org Settings)
const orgNavItems: NavItem[] = [
  { label: 'Customers', path: '/customers', icon: Users },
  {
    label: 'Jobs',
    icon: Briefcase,
    children: [
      { label: 'List', path: '/jobs' },
      { label: 'Calendar', path: '/jobs/calendar' },
    ],
  },
  { label: 'Quotes', path: '/quotes', icon: FileText },
  { label: 'Invoices', path: '/invoices', icon: Receipt },
  {
    label: 'Team',
    icon: UserCog,
    children: [
      { label: 'Members', path: '/team/members' },
      { label: 'Roles & Permissions', path: '/team/roles' },
    ],
  },
  {
    label: 'Reports',
    icon: BarChart3,
    children: [
      { label: 'Revenue', path: '/reports/revenue' },
      { label: 'Customers', path: '/reports/customers' },
    ],
  },
  // Settings goes to Organization Settings (Account Settings is in profile dropdown)
  { label: 'Settings', path: '/settings/organization', icon: Settings },
];

function NavItemLink({
  item,
  isCollapsed,
}: {
  item: NavItem;
  isCollapsed: boolean;
}) {
  const location = useLocation();
  const Icon = item.icon;
  const hasChildren = item.children && item.children.length > 0;
  const isChildActive =
    hasChildren &&
    item.children!.some((child) => location.pathname === child.path);

  // Initialize expanded state based on whether a child is active
  const [isExpanded, setIsExpanded] = useState(isChildActive);

  // Keep dropdown expanded when navigating to a child route
  useEffect(() => {
    if (isChildActive) {
      setIsExpanded(true);
    }
  }, [isChildActive, location.pathname]);

  if (hasChildren) {
    return (
      <div>
        <button
          onClick={() => setIsExpanded(!isExpanded)}
          className={cn(
            'w-full flex items-center gap-3 px-3 py-2 rounded-lg text-neutral-600 hover:bg-neutral-100 hover:text-neutral-900 transition-colors',
            (isExpanded || isChildActive) && 'bg-neutral-100 text-neutral-900',
            isCollapsed && 'justify-center'
          )}
          title={isCollapsed ? item.label : undefined}
        >
          <Icon className="h-5 w-5 flex-shrink-0" />
          {!isCollapsed && (
            <>
              <span className="flex-1 text-left text-sm font-medium">
                {item.label}
              </span>
              <ChevronDown
                className={cn(
                  'h-4 w-4 transition-transform',
                  isExpanded && 'rotate-180'
                )}
              />
            </>
          )}
        </button>
        {!isCollapsed && isExpanded && item.children && (
          <div className="mt-1 ml-8 space-y-1">
            {item.children.map((child) => (
              <NavLink
                key={child.path}
                to={child.path}
                end
                className={({ isActive }) =>
                  cn(
                    'block px-3 py-2 rounded-lg text-sm text-neutral-600 hover:bg-neutral-100 hover:text-neutral-900 transition-colors',
                    isActive && 'bg-primary-50 text-primary-700 font-medium'
                  )
                }
              >
                {child.label}
              </NavLink>
            ))}
          </div>
        )}
      </div>
    );
  }

  return (
    <NavLink
      to={item.path!}
      className={({ isActive }) =>
        cn(
          'flex items-center gap-3 px-3 py-2 rounded-lg text-neutral-600 hover:bg-neutral-100 hover:text-neutral-900 transition-colors',
          isActive && 'bg-primary-50 text-primary-700 font-medium',
          isCollapsed && 'justify-center'
        )
      }
      title={isCollapsed ? item.label : undefined}
    >
      <Icon className="h-5 w-5 flex-shrink-0" />
      {!isCollapsed && (
        <span className="text-sm font-medium">{item.label}</span>
      )}
    </NavLink>
  );
}

export function Sidebar() {
  const { isCollapsed, isMobileOpen, toggleCollapsed, setMobileOpen } =
    useSidebarStore();
  const { currentTenant } = useTenantStore();

  // Build nav items based on whether user has an organization
  // Settings (Org Settings) only shows when user has an organization
  const navItems = [...baseNavItems, ...(currentTenant ? orgNavItems : [])];

  return (
    <>
      {/* Mobile menu button */}
      <button
        onClick={() => setMobileOpen(true)}
        className="lg:hidden fixed top-4 left-4 z-40 p-2 rounded-lg bg-white shadow-md border border-neutral-200"
        aria-label="Open menu"
      >
        <Menu className="h-5 w-5 text-neutral-700" />
      </button>

      {/* Mobile overlay */}
      {isMobileOpen && (
        <div
          className="lg:hidden fixed inset-0 z-40 bg-black/50"
          onClick={() => setMobileOpen(false)}
        />
      )}

      {/* Sidebar */}
      <aside
        className={cn(
          'fixed lg:static inset-y-0 left-0 z-50 flex flex-col bg-white border-r border-neutral-200 transition-all duration-300',
          isCollapsed ? 'w-16' : 'w-64',
          isMobileOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'
        )}
      >
        {/* Header */}
        <div className="h-16 flex items-center justify-between px-4 border-b border-neutral-200">
          {!isCollapsed && (
            <span className="text-lg font-semibold text-primary-600">
              ServicePro
            </span>
          )}
          <button
            onClick={() => {
              if (isMobileOpen) {
                setMobileOpen(false);
              } else {
                toggleCollapsed();
              }
            }}
            className={cn(
              'p-1.5 rounded-lg text-neutral-500 hover:bg-neutral-100 hover:text-neutral-700 transition-colors',
              isCollapsed && 'mx-auto'
            )}
            aria-label={isCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          >
            {isMobileOpen ? (
              <X className="h-5 w-5" />
            ) : isCollapsed ? (
              <ChevronRight className="h-5 w-5" />
            ) : (
              <ChevronLeft className="h-5 w-5" />
            )}
          </button>
        </div>

        {/* Navigation */}
        <nav className="flex-1 p-3 space-y-1 overflow-y-auto">
          {navItems.map((item) => (
            <NavItemLink
              key={item.label}
              item={item}
              isCollapsed={isCollapsed}
            />
          ))}
        </nav>
      </aside>
    </>
  );
}
