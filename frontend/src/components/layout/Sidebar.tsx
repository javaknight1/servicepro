import { useState, useEffect } from 'react';
import { NavLink, useLocation } from 'react-router';
import { cn } from '@utils/cn';
import { useSidebarStore } from '@store/sidebarStore';
import { useTenantStore } from '@store/tenantStore';
import { useMembershipStore } from '@store/membershipStore';
import type { TenantMembership } from '@/types/membership';
import { TenantSwitcher } from '@components/tenants';
import {
  LayoutGrid,
  Building2,
  Briefcase,
  FileText,
  Receipt,
  BarChart3,
  Settings,
  ChevronLeft,
  ChevronRight,
  X,
  UserCog,
} from 'lucide-react';

interface NavChild {
  label: string;
  path: string;
}

interface NavItem {
  label: string;
  icon: React.ElementType;
  path?: string;
  /** Path prefixes that also make this item active (e.g. detail pages). */
  match?: string[];
  children?: NavChild[];
}

/*
 * Order follows the MVP mockup (docs/product/mockups/servicepro-05-dashboard.png):
 * Dashboard, Customers & Sites, Agreements, Schedule, Jobs, Quotes, Invoices,
 * Payments, Reports, Team, Settings. Agreements and Schedule (T167) and
 * Payments (T169/T170) have no pages yet, so they are left out until their
 * tasks ship rather than linking to a 404.
 */
const baseNavItems: NavItem[] = [
  { label: 'Dashboard', path: '/dashboard', icon: LayoutGrid },
];

// Items that need an organization to be selected.
const orgNavItems: NavItem[] = [
  {
    label: 'Customers & Sites',
    path: '/customers',
    icon: Building2,
    match: ['/customers'],
  },
  { label: 'Jobs', path: '/jobs', icon: Briefcase, match: ['/jobs'] },
  { label: 'Quotes', path: '/quotes', icon: FileText, match: ['/quotes'] },
  { label: 'Invoices', path: '/invoices', icon: Receipt, match: ['/invoices'] },
  {
    label: 'Reports',
    icon: BarChart3,
    children: [
      { label: 'A/R aging', path: '/reports/ar-aging' },
      { label: 'Revenue', path: '/reports/revenue' },
    ],
  },
  { label: 'Team', path: '/team/members', icon: UserCog, match: ['/team'] },
  // Organization settings; account settings live in the profile menu.
  {
    label: 'Settings',
    path: '/settings/organization',
    icon: Settings,
    match: ['/settings/organization'],
  },
];

const itemBase =
  'flex items-center gap-3 rounded-xl px-3 py-2.5 text-[15px] text-neutral-700 transition-colors hover:bg-neutral-100 hover:text-neutral-900';
const itemActive =
  'bg-primary-50 text-primary-600 hover:bg-primary-50 hover:text-primary-600';

function isPathActive(pathname: string, item: NavItem): boolean {
  const prefixes = item.match ?? (item.path ? [item.path] : []);
  return prefixes.some((p) => pathname === p || pathname.startsWith(`${p}/`));
}

function NavItemLink({
  item,
  isCollapsed,
  onNavigate,
}: {
  item: NavItem;
  isCollapsed: boolean;
  onNavigate: () => void;
}) {
  const location = useLocation();
  const Icon = item.icon;
  const children = item.children ?? [];
  const isChildActive = children.some(
    (child) => location.pathname === child.path
  );
  const [isExpanded, setIsExpanded] = useState(isChildActive);

  // Keep the group open when navigating to one of its pages.
  useEffect(() => {
    if (isChildActive) setIsExpanded(true);
  }, [isChildActive]);

  if (children.length > 0) {
    return (
      <div>
        <button
          type="button"
          onClick={() => setIsExpanded(!isExpanded)}
          aria-expanded={isExpanded}
          className={cn(
            itemBase,
            'w-full',
            isChildActive && !isExpanded && itemActive,
            isCollapsed && 'justify-center px-0'
          )}
          title={isCollapsed ? item.label : undefined}
        >
          <Icon className="h-5 w-5 shrink-0" strokeWidth={1.75} />
          {!isCollapsed && (
            <>
              <span className="flex-1 text-left">{item.label}</span>
              <ChevronRight
                className={cn(
                  'h-4 w-4 text-neutral-500 transition-transform',
                  isExpanded && 'rotate-90'
                )}
              />
            </>
          )}
        </button>
        {!isCollapsed && isExpanded && (
          <div className="mt-1 ml-11 space-y-0.5">
            {children.map((child) => (
              <NavLink
                key={child.path}
                to={child.path}
                end
                onClick={onNavigate}
                className={({ isActive }) =>
                  cn(
                    'block rounded-lg px-3 py-2 text-sm text-neutral-600 transition-colors hover:bg-neutral-100 hover:text-neutral-900',
                    isActive && 'bg-primary-50 font-medium text-primary-600'
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

  const active = isPathActive(location.pathname, item);
  return (
    <NavLink
      to={item.path ?? '/'}
      onClick={onNavigate}
      aria-current={active ? 'page' : undefined}
      className={cn(
        itemBase,
        active && itemActive,
        isCollapsed && 'justify-center px-0'
      )}
      title={isCollapsed ? item.label : undefined}
    >
      <Icon className="h-5 w-5 shrink-0" strokeWidth={1.75} />
      {!isCollapsed && <span>{item.label}</span>}
    </NavLink>
  );
}

/** "Team plan · unlimited users", from the tenant's membership tier. */
function planBadgeParts(
  membership: TenantMembership | null
): { plan: string; detail: string | null } | null {
  const tier = membership?.tier;
  const name =
    tier?.display_name || membership?.subscription?.tier_display_name;
  if (!name) return null;
  const plan = /plan$/i.test(name) ? name : `${name} plan`;
  const users = tier?.features?.team_members;
  let detail: string | null = null;
  if (users === -1) detail = 'unlimited users';
  else if (typeof users === 'number' && users > 0)
    detail = `up to ${users} ${users === 1 ? 'user' : 'users'}`;
  return { plan, detail };
}

function PlanBadge() {
  const currentTenant = useTenantStore((s) => s.currentTenant);
  const membership = useMembershipStore((s) => s.currentMembership);
  const loadMembership = useMembershipStore((s) => s.loadMembership);
  const tenantId = currentTenant?.id;

  useEffect(() => {
    if (!tenantId) return;
    if (membership?.subscription?.tenant_id === tenantId) return;
    // Not every role may read the plan; the badge is simply left out then.
    loadMembership(tenantId).catch(() => undefined);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tenantId]);

  const parts = planBadgeParts(membership);
  if (!tenantId || !parts) return null;
  return (
    <div className="border-t border-neutral-200 p-3">
      <div className="rounded-xl bg-neutral-100 px-4 py-3 text-sm text-neutral-600">
        <span className="font-medium text-neutral-900">{parts.plan}</span>
        {parts.detail && <> · {parts.detail}</>}
      </div>
    </div>
  );
}

export function Sidebar() {
  const { isCollapsed, isMobileOpen, toggleCollapsed, setMobileOpen } =
    useSidebarStore();
  const { currentTenant, tenants } = useTenantStore();
  const hasOrganizations = tenants.length > 0;

  const navItems = [...baseNavItems, ...(currentTenant ? orgNavItems : [])];
  // The drawer always shows labels, even when the desktop rail is collapsed.
  const collapsed = isCollapsed && !isMobileOpen;
  const closeDrawer = () => {
    if (isMobileOpen) setMobileOpen(false);
  };

  return (
    <>
      {/* Mobile overlay */}
      {isMobileOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/40 lg:hidden"
          onClick={() => setMobileOpen(false)}
          aria-hidden="true"
        />
      )}

      <aside
        aria-label="Main navigation"
        className={cn(
          'fixed inset-y-0 left-0 z-50 flex flex-col border-r border-neutral-200 bg-white transition-all duration-300 lg:sticky lg:top-0 lg:h-screen',
          collapsed ? 'w-64 lg:w-[72px]' : 'w-64',
          isMobileOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'
        )}
      >
        <div
          className={cn(
            'flex h-16 shrink-0 items-center border-b border-neutral-200 px-4',
            collapsed ? 'justify-center' : 'justify-between'
          )}
        >
          {!collapsed && (
            <NavLink
              to="/dashboard"
              onClick={closeDrawer}
              className="pl-1 text-xl font-bold tracking-tight text-primary-600"
            >
              ServicePro
            </NavLink>
          )}
          <button
            type="button"
            onClick={() =>
              isMobileOpen ? setMobileOpen(false) : toggleCollapsed()
            }
            className="rounded-lg p-1.5 text-neutral-500 transition-colors hover:bg-neutral-100 hover:text-neutral-800"
            aria-label={
              isMobileOpen
                ? 'Close menu'
                : collapsed
                  ? 'Expand sidebar'
                  : 'Collapse sidebar'
            }
          >
            {isMobileOpen ? (
              <X className="h-5 w-5" />
            ) : collapsed ? (
              <ChevronRight className="h-5 w-5" />
            ) : (
              <ChevronLeft className="h-5 w-5" />
            )}
          </button>
        </div>

        {/* On phones the top bar has no room for the organization switcher. */}
        {isMobileOpen && hasOrganizations && (
          <div className="border-b border-neutral-200 px-2 py-2 sm:hidden">
            <TenantSwitcher />
          </div>
        )}

        <nav className="flex-1 space-y-1 overflow-y-auto p-3">
          {navItems.map((item) => (
            <NavItemLink
              key={item.label}
              item={item}
              isCollapsed={collapsed}
              onNavigate={closeDrawer}
            />
          ))}
        </nav>

        {!collapsed && <PlanBadge />}
      </aside>
    </>
  );
}
