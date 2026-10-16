import { AlertCircle } from 'lucide-react';
import { cn } from '@utils/cn';
import { Button } from '../Button';

export interface ErrorStateProps {
  title?: string;
  message?: string;
  onRetry?: () => void;
  retryLabel?: string;
  className?: string;
}

/**
 * Shown when data fails to load, so an API error is never mistaken for an
 * empty list.
 */
export function ErrorState({
  title = 'Something went wrong',
  message,
  onRetry,
  retryLabel = 'Try again',
  className,
}: ErrorStateProps) {
  return (
    <div
      role="alert"
      className={cn(
        'flex flex-col items-center justify-center py-12 px-4 text-center',
        className
      )}
    >
      <div className="mb-4 p-4 bg-red-50 rounded-full text-red-500">
        <AlertCircle className="h-12 w-12" aria-hidden="true" />
      </div>
      <h3 className="text-lg font-semibold text-neutral-900">{title}</h3>
      {message && (
        <p className="mt-2 text-sm text-neutral-600 max-w-sm">{message}</p>
      )}
      {onRetry && (
        <Button variant="primary" size="md" onClick={onRetry} className="mt-4">
          {retryLabel}
        </Button>
      )}
    </div>
  );
}
