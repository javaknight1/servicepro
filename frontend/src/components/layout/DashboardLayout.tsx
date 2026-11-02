import { ReactNode } from 'react';
import { Sidebar } from './Sidebar';
import { AppTopBar, type Breadcrumb } from './AppTopBar';
import { BuildStamp } from './BuildStamp';

interface DashboardLayoutProps {
  children: ReactNode;
  /** Top-bar trail; defaults to the page title for the current route. */
  breadcrumbs?: Breadcrumb[];
}

/** Signed-in app shell (mockup 05): sidebar, top bar, page content. */
export function DashboardLayout({
  children,
  breadcrumbs,
}: DashboardLayoutProps) {
  return (
    <div className="flex min-h-screen bg-neutral-50">
      <Sidebar />
      <div className="flex min-w-0 flex-1 flex-col">
        <AppTopBar breadcrumbs={breadcrumbs} />
        <main className="flex-1">{children}</main>
        <footer className="border-t border-neutral-200 bg-white px-4 py-1.5">
          <BuildStamp />
        </footer>
      </div>
    </div>
  );
}
