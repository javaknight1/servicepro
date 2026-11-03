import type { TenantSettings } from '@/types/tenant';

/** What a step saves: settings to merge and, optionally, a new name. */
export interface StepSave {
  settings: Partial<TenantSettings>;
  name?: string;
}

/** Props every wizard step gets from the page. */
export interface StepProps {
  /** Save the step and move to the next one; rejects with the API error. */
  onContinue: (save: StepSave) => Promise<void>;
  /** Move on without saving this step's answers. */
  onSkip: () => Promise<void>;
  onBack?: () => void;
  /** True while a save is in flight. */
  busy: boolean;
}

export const fieldClass =
  'w-full rounded-lg border bg-white px-3 py-2.5 text-[15px] text-neutral-900 focus:outline-hidden focus:ring-2 focus:ring-primary-500 focus:border-primary-500';

export function fieldBorder(error?: string) {
  return error ? 'border-error-500' : 'border-neutral-300';
}
