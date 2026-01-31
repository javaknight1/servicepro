/* eslint-disable react-refresh/only-export-components */
import { cn } from '@utils/cn';

export type BadgeVariant = 'success' | 'warning' | 'error' | 'info' | 'neutral';

export interface BadgeProps {
  children: React.ReactNode;
  variant?: BadgeVariant;
  size?: 'sm' | 'md';
  className?: string;
}

export function Badge({
  children,
  variant = 'neutral',
  size = 'md',
  className,
}: BadgeProps) {
  const variants = {
    success: 'bg-success-100 text-success-700 border-success-200',
    warning: 'bg-warning-100 text-warning-700 border-warning-200',
    error: 'bg-error-100 text-error-700 border-error-200',
    info: 'bg-primary-100 text-primary-700 border-primary-200',
    neutral: 'bg-neutral-100 text-neutral-700 border-neutral-200',
  };

  const sizes = {
    sm: 'px-2 py-0.5 text-xs',
    md: 'px-2.5 py-1 text-sm',
  };

  return (
    <span
      className={cn(
        'inline-flex items-center font-medium rounded-full border',
        variants[variant],
        sizes[size],
        className
      )}
    >
      {children}
    </span>
  );
}

export function getStatusBadgeVariant(status: string): BadgeVariant {
  const statusMap: Record<string, BadgeVariant> = {
    // Generic statuses
    active: 'success',
    inactive: 'neutral',
    pending: 'warning',
    completed: 'success',
    cancelled: 'error',

    // Quote statuses
    draft: 'neutral',
    sent: 'info',
    viewed: 'info',
    accepted: 'success',
    declined: 'error',
    expired: 'warning',

    // Job statuses
    scheduled: 'info',
    in_progress: 'warning',

    // Invoice statuses
    paid: 'success',
    partially_paid: 'warning',
    overdue: 'error',

    // Customer statuses
    prospect: 'info',

    // Priorities
    low: 'neutral',
    medium: 'info',
    high: 'warning',
    urgent: 'error',
  };

  return statusMap[status.toLowerCase()] || 'neutral';
}
