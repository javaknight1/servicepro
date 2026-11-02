import { Fragment } from 'react';
import { Link, useLocation } from 'react-router';
import { ChevronRight, Menu } from 'lucide-react';
import { useTenantStore } from '@store';
import { useSidebarStore } from '@store/sidebarStore';
import { TenantSwitcher } from '@components/tenants';
import { ProfileDropdown } from './Header';

/** One step of the top bar's trail; the last one is the current page. */
export interface Breadcrumb {
  label: string;
  to?: string;
}

/** Page titles by path prefix, most specific first. */
const PAGE_TITLES: [string, string][] = [
  ['/dashboard', 'Dashboard'],
  ['/customers', 'Customers & Sites'],
  ['/jobs/calendar', 'Job calendar'],
  ['/jobs', 'Jobs'],
  ['/quotes', 'Quotes'],
  ['/invoices', 'Invoices'],
  ['/reports/ar-aging', 'A/R aging'],
  ['/reports/revenue', 'Revenue'],
  ['/team', 'Team'],
  ['/settings/organization', 'Settings'],
  ['/settings', 'Account settings'],
];

// eslint-disable-next-line react-refresh/only-export-components
export function pageTitleFor(pathname: string): string {
  const hit = PAGE_TITLES.find(
    ([prefix]) => pathname === prefix || pathname.startsWith(`${prefix}/`)
  );
  return hit ? hit[1] : 'ServicePro';
}

interface AppTopBarProps {
  /** Overrides the title derived from the route. */
  breadcrumbs?: Breadcrumb[];
}

/** The signed-in top bar: page title left, organization and profile right. */
export function AppTopBar({ breadcrumbs }: AppTopBarProps) {
  const { pathname } = useLocation();
  const hasOrganizations = useTenantStore((s) => s.tenants.length > 0);
  const setMobileOpen = useSidebarStore((s) => s.setMobileOpen);
  const trail: Breadcrumb[] =
    breadcrumbs && breadcrumbs.length > 0
      ? breadcrumbs
      : [{ label: pageTitleFor(pathname) }];

  return (
    <header className="sticky top-0 z-30 flex h-16 shrink-0 items-center gap-3 border-b border-neutral-200 bg-white px-4 sm:px-6 lg:px-10">
      <button
        type="button"
        onClick={() => setMobileOpen(true)}
        className="-ml-1 rounded-lg p-2 text-neutral-600 hover:bg-neutral-100 lg:hidden"
        aria-label="Open menu"
      >
        <Menu className="h-5 w-5" />
      </button>

      <nav aria-label="Breadcrumb" className="min-w-0 flex-1">
        <ol className="flex min-w-0 items-center gap-2 text-[15px]">
          {trail.map((crumb, i) => {
            const isLast = i === trail.length - 1;
            return (
              <Fragment key={`${crumb.label}-${i}`}>
                {i > 0 && (
                  <ChevronRight
                    className="h-4 w-4 shrink-0 text-neutral-400"
                    aria-hidden="true"
                  />
                )}
                <li
                  className={
                    isLast
                      ? 'truncate font-medium text-neutral-900'
                      : 'hidden shrink-0 text-neutral-500 sm:block'
                  }
                  aria-current={isLast ? 'page' : undefined}
                >
                  {crumb.to && !isLast ? (
                    <Link to={crumb.to} className="hover:text-neutral-800">
                      {crumb.label}
                    </Link>
                  ) : (
                    crumb.label
                  )}
                </li>
              </Fragment>
            );
          })}
        </ol>
      </nav>

      <div className="flex shrink-0 items-center gap-3 sm:gap-4">
        {hasOrganizations && (
          <>
            <TenantSwitcher className="hidden sm:block" />
            <div className="hidden h-7 w-px bg-neutral-200 sm:block" />
          </>
        )}
        <ProfileDropdown />
      </div>
    </header>
  );
}
