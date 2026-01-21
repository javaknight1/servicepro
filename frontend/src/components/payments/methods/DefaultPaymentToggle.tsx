import { useId } from 'react';
import type { DefaultPaymentToggleProps } from '../../../types/payment';
import { cn } from '../../../utils/cn';

export function DefaultPaymentToggle({
  isDefault,
  onChange,
  disabled = false,
  size = 'md',
  className,
}: DefaultPaymentToggleProps) {
  const id = useId();

  const handleToggle = () => {
    if (!disabled) {
      onChange(!isDefault);
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault();
      handleToggle();
    }
  };

  // Size-based classes
  const sizeClasses = {
    sm: {
      toggle: 'h-5 w-9',
      circle: 'h-3.5 w-3.5',
      translateOn: 'translate-x-4',
      translateOff: 'translate-x-0.5',
      label: 'text-sm',
    },
    md: {
      toggle: 'h-6 w-11',
      circle: 'h-4 w-4',
      translateOn: 'translate-x-5',
      translateOff: 'translate-x-1',
      label: 'text-sm',
    },
    lg: {
      toggle: 'h-7 w-14',
      circle: 'h-5 w-5',
      translateOn: 'translate-x-7',
      translateOff: 'translate-x-1',
      label: 'text-base',
    },
  };

  const sizes = sizeClasses[size];

  return (
    <div className={cn('flex items-center gap-3', className)}>
      <button
        type="button"
        role="switch"
        id={id}
        aria-checked={isDefault}
        aria-label={
          isDefault
            ? 'Remove as default payment method'
            : 'Set as default payment method'
        }
        disabled={disabled}
        onClick={handleToggle}
        onKeyDown={handleKeyDown}
        className={cn(
          'relative inline-flex shrink-0 rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out',
          'focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
          sizes.toggle,
          isDefault ? 'bg-primary-600' : 'bg-neutral-200',
          disabled && 'cursor-not-allowed opacity-50'
        )}
      >
        <span
          aria-hidden="true"
          className={cn(
            'pointer-events-none inline-block transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
            sizes.circle,
            isDefault ? sizes.translateOn : sizes.translateOff
          )}
        />
      </button>
      <label
        htmlFor={id}
        className={cn(
          'cursor-pointer select-none font-medium',
          sizes.label,
          disabled ? 'cursor-not-allowed text-neutral-400' : 'text-neutral-700'
        )}
      >
        {isDefault ? 'Default payment method' : 'Set as default'}
      </label>
    </div>
  );
}
