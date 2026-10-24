import { ReactNode } from 'react';
import { Sidebar } from './Sidebar';
import { Header } from './Header';
import { BuildStamp } from './BuildStamp';
import { cn } from '@utils/cn';
import { useSidebarStore } from '@store/sidebarStore';

interface DashboardLayoutProps {
  children: ReactNode;
}

export function DashboardLayout({ children }: DashboardLayoutProps) {
  const { isCollapsed } = useSidebarStore();

  return (
    <div className="min-h-screen flex bg-neutral-50">
      <Sidebar />
      <div
        className={cn(
          'flex-1 flex flex-col min-w-0 transition-all duration-300',
          isCollapsed ? 'lg:ml-0' : 'lg:ml-0'
        )}
      >
        <Header />
        <main className="flex-1 overflow-auto">{children}</main>
        <footer className="border-t border-neutral-200 bg-white px-4 py-1.5">
          <BuildStamp />
        </footer>
      </div>
    </div>
  );
}
