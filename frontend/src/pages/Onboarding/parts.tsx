import type { ReactNode } from 'react';
import { cn } from '@utils/cn';

export function FieldError({ message }: { message?: string }) {
  if (!message) return null;
  return <p className="mt-1.5 text-sm text-error-600">{message}</p>;
}

export function Hint({ children }: { children: ReactNode }) {
  return <p className="mt-1.5 text-sm text-neutral-500">{children}</p>;
}

/** A radio option drawn as a card (mockups 02 and 03). */
export function ChoiceCard({
  name,
  value,
  checked,
  onChange,
  title,
  description,
  badge,
}: {
  name: string;
  value: string;
  checked: boolean;
  onChange: (value: string) => void;
  title: string;
  description: string;
  badge?: string;
}) {
  return (
    <label
      className={cn(
        'flex cursor-pointer items-start gap-3 rounded-xl border bg-white p-4 transition-colors',
        checked
          ? 'border-primary-600 ring-1 ring-primary-600'
          : 'border-neutral-200 hover:border-neutral-300'
      )}
    >
      <input
        type="radio"
        name={name}
        value={value}
        checked={checked}
        onChange={() => onChange(value)}
        className="mt-1 h-4 w-4 accent-primary-600"
      />
      <span>
        <span className="flex items-center gap-2 font-medium text-neutral-900">
          {title}
          {badge && (
            <span className="rounded-full bg-primary-50 px-2 py-0.5 text-xs font-medium text-primary-700">
              {badge}
            </span>
          )}
        </span>
        <span className="mt-0.5 block text-sm text-neutral-600">
          {description}
        </span>
      </span>
    </label>
  );
}

/** An on/off switch row (mockup 03's billing rules). */
export function ToggleRow({
  id,
  title,
  description,
  checked,
  onChange,
}: {
  id: string;
  title: string;
  description: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
}) {
  return (
    <div className="flex items-center justify-between gap-4 p-4">
      <div>
        <label htmlFor={id} className="font-medium text-neutral-900">
          {title}
        </label>
        <p className="mt-0.5 text-sm text-neutral-600">{description}</p>
      </div>
      <label className="relative inline-flex shrink-0 cursor-pointer items-center">
        <input
          id={id}
          type="checkbox"
          role="switch"
          checked={checked}
          onChange={(e) => onChange(e.target.checked)}
          className="peer sr-only"
        />
        <span
          aria-hidden="true"
          className="h-6 w-11 rounded-full bg-neutral-200 transition-colors after:absolute after:left-[2px] after:top-[2px] after:h-5 after:w-5 after:rounded-full after:bg-white after:transition-all peer-checked:bg-primary-600 peer-checked:after:translate-x-full peer-focus-visible:ring-2 peer-focus-visible:ring-primary-500"
        />
      </label>
    </div>
  );
}

/** Back / skip / continue row at the bottom of each step. */
export function StepFooter({
  onBack,
  onSkip,
  skipLabel = 'Skip this step',
  primaryLabel,
  busy,
  note,
  onPrimary,
}: {
  onBack?: () => void;
  onSkip: () => void;
  skipLabel?: string;
  primaryLabel: string;
  busy: boolean;
  note?: string;
  /** Click handler for a non-submit primary button; submits otherwise. */
  onPrimary?: () => void;
}) {
  return (
    <div className="mt-8 flex flex-wrap items-center gap-3 border-t border-neutral-200 pt-6">
      {onBack && (
        <button
          type="button"
          onClick={onBack}
          className="inline-flex items-center gap-2 rounded-lg px-3 py-2 text-[15px] font-medium text-neutral-700 hover:bg-neutral-100"
        >
          <span aria-hidden="true">←</span> Back
        </button>
      )}
      <div className="ml-auto flex flex-wrap items-center gap-3">
        {note && <span className="text-sm text-neutral-500">{note}</span>}
        <button
          type="button"
          onClick={onSkip}
          disabled={busy}
          className="rounded-lg px-3 py-2 text-[15px] font-medium text-neutral-700 hover:bg-neutral-100 disabled:opacity-50"
        >
          {skipLabel}
        </button>
        <button
          type={onPrimary ? 'button' : 'submit'}
          onClick={onPrimary}
          disabled={busy}
          className="rounded-lg bg-primary-600 px-5 py-2.5 text-[15px] font-medium text-white hover:bg-primary-700 disabled:opacity-60"
        >
          {busy ? 'Saving…' : primaryLabel}
        </button>
      </div>
    </div>
  );
}
