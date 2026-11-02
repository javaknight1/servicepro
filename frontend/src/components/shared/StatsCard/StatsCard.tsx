import { ReactNode } from 'react';
import { cn } from '@utils/cn';

export interface StatsCardProps {
  label: string;
  value: string | number;
  icon?: ReactNode;
  /** Line under the value. */
  change?: string;
  /** Colour of that line: green, red or grey. */
  changeType?: 'positive' | 'negative' | 'neutral';
  className?: string;
}

/** KPI card from the MVP mockups: label, big value, coloured subline, icon tile. */
export function StatsCard({
  label,
  value,
  icon,
  change,
  changeType = 'neutral',
  className,
}: StatsCardProps) {
  const changeColors = {
    positive: 'text-success-600',
    negative: 'text-error-600',
    neutral: 'text-neutral-500',
  };

  return (
    <div
      className={cn(
        'rounded-2xl border border-neutral-200 bg-white p-6',
        className
      )}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <p className="text-[15px] leading-snug text-neutral-600">{label}</p>
          <p className="mt-1.5 text-3xl font-bold tracking-tight text-neutral-900 tabular-nums">
            {value}
          </p>
        </div>
        {icon && (
          <div className="mt-1 flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-primary-50 text-primary-600">
            {icon}
          </div>
        )}
      </div>
      {change && (
        <p className={cn('mt-1.5 text-sm', changeColors[changeType])}>
          {change}
        </p>
      )}
    </div>
  );
}
