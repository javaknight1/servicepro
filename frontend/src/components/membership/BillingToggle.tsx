import type { BillingCycle } from '@/types/membership';

interface BillingToggleProps {
  value: BillingCycle;
  onChange: (cycle: BillingCycle) => void;
}

export function BillingToggle({ value, onChange }: BillingToggleProps) {
  return (
    <div className="flex flex-col items-center space-y-2">
      <div className="inline-flex items-center bg-neutral-100 rounded-lg p-1">
        <button
          type="button"
          onClick={() => onChange('monthly')}
          className={`px-4 py-2 text-sm font-medium rounded-md transition-colors ${
            value === 'monthly'
              ? 'bg-white text-neutral-900 shadow-sm'
              : 'text-neutral-600 hover:text-neutral-900'
          }`}
        >
          Monthly
        </button>
        <button
          type="button"
          onClick={() => onChange('annual')}
          className={`px-4 py-2 text-sm font-medium rounded-md transition-colors ${
            value === 'annual'
              ? 'bg-white text-neutral-900 shadow-sm'
              : 'text-neutral-600 hover:text-neutral-900'
          }`}
        >
          Annual
          <span className="ml-1.5 text-xs text-green-600 font-semibold">
            Save 17%
          </span>
        </button>
      </div>
      {value === 'annual' && (
        <p className="text-xs text-neutral-500">Billed annually</p>
      )}
    </div>
  );
}

export default BillingToggle;
