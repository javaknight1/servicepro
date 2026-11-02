import { ReactNode } from 'react';
import { cn } from '@utils/cn';

export interface PageHeaderProps {
  title: ReactNode;
  /** One line under the title (date, counts, a short explanation). */
  subtitle?: ReactNode;
  /** Shown before the title, e.g. an initials tile. */
  leading?: ReactNode;
  /** Shown right after the title, e.g. a status badge. */
  badge?: ReactNode;
  /** Buttons on the right: secondary (outline) first, primary last. */
  actions?: ReactNode;
  className?: string;
}

/** Page title block from the MVP mockups: big title, grey subline, actions right. */
export function PageHeader({
  title,
  subtitle,
  leading,
  badge,
  actions,
  className,
}: PageHeaderProps) {
  return (
    <div
      className={cn(
        'mb-6 flex flex-col gap-4 sm:flex-row sm:items-end sm:justify-between',
        className
      )}
    >
      <div className="flex min-w-0 items-center gap-4">
        {leading}
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-3">
            <h1 className="text-2xl font-bold tracking-tight text-neutral-900 md:text-3xl">
              {title}
            </h1>
            {badge}
          </div>
          {subtitle && (
            <p className="mt-1 text-base text-neutral-600">{subtitle}</p>
          )}
        </div>
      </div>
      {actions && (
        <div className="flex shrink-0 flex-wrap items-center gap-3">
          {actions}
        </div>
      )}
    </div>
  );
}
