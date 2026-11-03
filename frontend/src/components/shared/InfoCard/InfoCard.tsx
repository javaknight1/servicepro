import { ReactNode } from 'react';
import { cn } from '@utils/cn';

export interface InfoCardProps {
  /** Small orange icon before the label. */
  icon?: ReactNode;
  label: string;
  /** The main fact, in bold. */
  value: ReactNode;
  /** A grey line under the value. */
  detail?: ReactNode;
  className?: string;
}

/** A labelled fact card (mockup 06's billing rules row). */
export function InfoCard({
  icon,
  label,
  value,
  detail,
  className,
}: InfoCardProps) {
  return (
    <div
      className={cn(
        'min-w-0 rounded-2xl border border-neutral-200 bg-white px-5 py-4',
        className
      )}
    >
      <p className="flex items-center gap-2 text-[15px] text-neutral-600">
        {icon && (
          <span className="text-primary-600 [&>svg]:h-4 [&>svg]:w-4">
            {icon}
          </span>
        )}
        {label}
      </p>
      <p className="mt-2 truncate text-base font-semibold text-neutral-900">
        {value}
      </p>
      {detail && (
        <p className="mt-0.5 truncate text-sm text-neutral-500">{detail}</p>
      )}
    </div>
  );
}
