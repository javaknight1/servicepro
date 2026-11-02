import { ReactNode } from 'react';
import { Link } from 'react-router';
import { ArrowRight } from 'lucide-react';
import { cn } from '@utils/cn';

export interface ListCardLinkAction {
  label: string;
  to: string;
}

export interface ListCardProps {
  title: ReactNode;
  /** Shown in grey after the title, e.g. "(9)". */
  count?: number;
  /** Right side of the header: a link ("View all →") or any node. */
  action?: ListCardLinkAction | ReactNode;
  /** Pinned under the rows (summary + button). */
  footer?: ReactNode;
  children: ReactNode;
  className?: string;
  /** Classes for the body wrapper (e.g. padding around the rows). */
  bodyClassName?: string;
}

function isLinkAction(
  action: ListCardProps['action']
): action is ListCardLinkAction {
  return (
    typeof action === 'object' &&
    action !== null &&
    'to' in action &&
    'label' in action
  );
}

/** White panel with a header, rows and an optional footer (mockup 05 panels). */
export function ListCard({
  title,
  count,
  action,
  footer,
  children,
  className,
  bodyClassName,
}: ListCardProps) {
  return (
    <section
      className={cn(
        'flex flex-col overflow-hidden rounded-2xl border border-neutral-200 bg-white',
        className
      )}
    >
      <header className="flex items-center justify-between gap-4 px-6 pt-5 pb-3">
        <h2 className="text-lg font-semibold text-neutral-900 md:text-xl">
          {title}
          {count !== undefined && (
            <span className="ml-1.5 font-normal text-neutral-500">
              ({count})
            </span>
          )}
        </h2>
        {isLinkAction(action) ? (
          <Link
            to={action.to}
            className="inline-flex shrink-0 items-center gap-1 text-sm font-medium text-primary-600 hover:text-primary-700"
          >
            {action.label}
            <ArrowRight className="h-4 w-4" />
          </Link>
        ) : (
          action
        )}
      </header>
      <div className={cn('flex-1', bodyClassName)}>{children}</div>
      {footer && (
        <footer className="flex items-center justify-between gap-4 border-t border-neutral-200 px-6 py-4">
          {footer}
        </footer>
      )}
    </section>
  );
}

export interface ListCardRowProps {
  /** Orange mono reference before the title, e.g. "J-3317". */
  reference?: ReactNode;
  title: ReactNode;
  subtitle?: ReactNode;
  /** Leading element, e.g. an avatar. */
  leading?: ReactNode;
  /** Right-aligned value, e.g. an amount. */
  value?: ReactNode;
  /** Extra right-side element after the row (button, chevron). */
  trailing?: ReactNode;
  /** Makes the row a link. */
  to?: string;
  className?: string;
}

/** One row in a ListCard: reference + title, grey subline, value right. */
export function ListCardRow({
  reference,
  title,
  subtitle,
  leading,
  value,
  trailing,
  to,
  className,
}: ListCardRowProps) {
  const body = (
    <>
      {leading}
      <div className="min-w-0 flex-1">
        <p className="flex min-w-0 items-baseline gap-2 text-[15px] text-neutral-900">
          {reference && (
            <span className="shrink-0 font-mono text-sm text-primary-600">
              {reference}
            </span>
          )}
          <span className="truncate font-medium">{title}</span>
        </p>
        {subtitle && (
          <p className="mt-0.5 truncate text-sm text-neutral-500">{subtitle}</p>
        )}
      </div>
      {value !== undefined && value !== null && (
        <div className="shrink-0 text-right text-[15px] font-semibold text-neutral-900 tabular-nums">
          {value}
        </div>
      )}
    </>
  );

  const rowClass =
    'flex min-w-0 flex-1 items-center gap-4 rounded-xl px-4 py-3 transition-colors';
  return (
    <li className={cn('flex items-center gap-2', className)}>
      {to ? (
        <Link to={to} className={cn(rowClass, 'hover:bg-neutral-50')}>
          {body}
        </Link>
      ) : (
        <div className={rowClass}>{body}</div>
      )}
      {trailing}
    </li>
  );
}
