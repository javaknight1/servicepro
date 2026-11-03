import { cn } from '@utils/cn';
import type { TeamRoleOption } from './teamRoles';

interface RolePillsProps {
  /** Accessible name of the group, e.g. "Role". */
  label: string;
  options: TeamRoleOption[];
  /** Selected role ID ('' for none). */
  value: string;
  onChange: (roleId: string) => void;
}

/** Owner / Office manager / Technician as a row of pills (mockup 13). */
export function RolePills({ label, options, value, onChange }: RolePillsProps) {
  const selected = options.find((o) => o.id === value);
  return (
    <div>
      <div
        role="radiogroup"
        aria-label={label}
        className="flex flex-wrap gap-2"
      >
        {options.map((option) => {
          const checked = option.id === value;
          return (
            <button
              key={option.id}
              type="button"
              role="radio"
              aria-checked={checked}
              onClick={() => onChange(option.id)}
              className={cn(
                'rounded-full border px-4 py-1.5 text-sm font-medium transition-colors',
                checked
                  ? 'border-primary-600 bg-primary-50 text-primary-700'
                  : 'border-neutral-300 bg-white text-neutral-700 hover:bg-neutral-50'
              )}
            >
              {option.label}
            </button>
          );
        })}
      </div>
      {selected && (
        <p className="mt-2 text-sm text-neutral-600">{selected.summary}</p>
      )}
    </div>
  );
}
