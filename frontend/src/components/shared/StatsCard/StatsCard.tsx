import { ReactNode } from 'react';
import { cn } from '@utils/cn';
import { TrendingUp, TrendingDown, Minus } from 'lucide-react';

export interface StatsCardProps {
  label: string;
  value: string | number;
  icon?: ReactNode;
  change?: string;
  changeType?: 'positive' | 'negative' | 'neutral';
  className?: string;
}

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
    neutral: 'text-neutral-600',
  };

  const ChangeIcon =
    changeType === 'positive'
      ? TrendingUp
      : changeType === 'negative'
        ? TrendingDown
        : Minus;

  return (
    <div
      className={cn(
        'bg-white rounded-xl border border-neutral-200 p-6 shadow-sm',
        className
      )}
    >
      <div className="flex items-center justify-between">
        <div className="min-w-0 flex-1">
          <p className="text-sm font-medium text-neutral-600 truncate">
            {label}
          </p>
          <p className="text-2xl font-bold text-neutral-900 mt-1">{value}</p>
          {change && (
            <div
              className={cn(
                'flex items-center gap-1 mt-1',
                changeColors[changeType]
              )}
            >
              <ChangeIcon className="h-4 w-4" />
              <span className="text-sm">{change}</span>
            </div>
          )}
        </div>
        {icon && (
          <div className="p-3 bg-primary-100 rounded-lg flex-shrink-0 ml-4">
            {icon}
          </div>
        )}
      </div>
    </div>
  );
}
