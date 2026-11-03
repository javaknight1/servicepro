import { useState } from 'react';
import { Minus, Plus } from 'lucide-react';
import { getFieldErrors, getTenantErrorMessage } from '@/utils/error';
import {
  TRADE_OPTIONS,
  browserTimezone,
  timezoneOptions,
} from '@/utils/tenantSettings';
import type { Tenant } from '@/types/tenant';
import { ChoiceCard, FieldError, Hint, StepFooter } from './parts';
import { fieldBorder, fieldClass, type StepProps } from './steps';

const MAX_TECHNICIANS = 10000;

/** Onboarding 1 — your business (mockup 02): name, trade, team size, time
 * zone and where the business is. */
export function BusinessStep({
  tenant,
  onContinue,
  onSkip,
  busy,
}: StepProps & { tenant: Tenant }) {
  const s = tenant.settings ?? {};
  const [name, setName] = useState(tenant.name);
  const [trade, setTrade] = useState(s.trade ?? '');
  const [techs, setTechs] = useState(s.technicianCount ?? 1);
  const [timezone, setTimezone] = useState(s.timezone || browserTimezone());
  const [city, setCity] = useState(s.businessCity ?? '');
  const [state, setState] = useState(s.businessState ?? '');
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [error, setError] = useState<string | null>(null);

  const setTechCount = (n: number) =>
    setTechs(Math.min(MAX_TECHNICIANS, Math.max(0, Math.round(n) || 0)));

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (name.trim().length < 2) {
      setErrors({ name: 'Enter your business name' });
      return;
    }
    setErrors({});
    setError(null);
    try {
      await onContinue({
        name: name.trim(),
        settings: {
          trade: trade || undefined,
          technicianCount: techs,
          timezone,
          businessCity: city.trim() || undefined,
          businessState: state.trim() || undefined,
        },
      });
    } catch (err: unknown) {
      const fields = getFieldErrors(err);
      setErrors(fields);
      setError(
        Object.keys(fields).length > 0
          ? null
          : getTenantErrorMessage(err, 'Could not save your business details')
      );
    }
  };

  return (
    <form onSubmit={handleSubmit} noValidate>
      <h1 className="text-3xl font-bold tracking-tight text-neutral-900 md:text-4xl">
        Your business
      </h1>
      <p className="mt-2 text-lg text-neutral-600">
        This sets up your workspace, your schedule&apos;s time zone and the name
        customers see on invoices.
      </p>

      {error && (
        <div
          role="alert"
          className="mt-6 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700"
        >
          {error}
        </div>
      )}

      <div className="mt-8 space-y-7">
        <div>
          <label
            htmlFor="ob-name"
            className="mb-2 block font-medium text-neutral-800"
          >
            Business name
          </label>
          <input
            id="ob-name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            maxLength={200}
            className={`${fieldClass} ${fieldBorder(errors.name)}`}
          />
          <FieldError message={errors.name} />
          {!errors.name && (
            <Hint>Shown on invoices, service reports and payment emails.</Hint>
          )}
        </div>

        <fieldset>
          <legend className="mb-2 font-medium text-neutral-800">
            What kind of work do you do?
          </legend>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            {TRADE_OPTIONS.map((t) => (
              <ChoiceCard
                key={t.value}
                name="trade"
                value={t.value}
                checked={trade === t.value}
                onChange={setTrade}
                title={t.label}
                description={t.description}
              />
            ))}
          </div>
          <FieldError message={errors.trade} />
          {!errors.trade && <Hint>You can change this later.</Hint>}
        </fieldset>

        <div className="grid grid-cols-1 gap-6 sm:grid-cols-2">
          <div>
            <label
              htmlFor="ob-techs"
              className="mb-2 block font-medium text-neutral-800"
            >
              How many technicians?
            </label>
            <div className="flex">
              <button
                type="button"
                aria-label="Fewer technicians"
                onClick={() => setTechCount(techs - 1)}
                className="rounded-l-lg border border-neutral-300 bg-white px-4 text-neutral-600 hover:bg-neutral-50"
              >
                <Minus className="h-4 w-4" />
              </button>
              <input
                id="ob-techs"
                type="number"
                min={0}
                max={MAX_TECHNICIANS}
                value={techs}
                onChange={(e) => setTechCount(Number(e.target.value))}
                className={`${fieldClass} rounded-none border-x-0 text-center ${fieldBorder(errors.technicianCount)}`}
              />
              <button
                type="button"
                aria-label="More technicians"
                onClick={() => setTechCount(techs + 1)}
                className="rounded-r-lg border border-neutral-300 bg-white px-4 text-neutral-600 hover:bg-neutral-50"
              >
                <Plus className="h-4 w-4" />
              </button>
            </div>
            <FieldError message={errors.technicianCount} />
            {!errors.technicianCount && (
              <Hint>Field staff; add your office team later.</Hint>
            )}
          </div>

          <div>
            <label
              htmlFor="ob-timezone"
              className="mb-2 block font-medium text-neutral-800"
            >
              Time zone
            </label>
            <select
              id="ob-timezone"
              value={timezone}
              onChange={(e) => setTimezone(e.target.value)}
              className={`${fieldClass} ${fieldBorder(errors.timezone)}`}
            >
              {timezoneOptions(timezone).map((tz) => (
                <option key={tz.value} value={tz.value}>
                  {tz.label}
                </option>
              ))}
            </select>
            <FieldError message={errors.timezone} />
            {!errors.timezone && (
              <Hint>Due dates and reminders follow this zone.</Hint>
            )}
          </div>
        </div>

        <div className="grid grid-cols-1 gap-6 sm:grid-cols-3">
          <div className="sm:col-span-2">
            <label
              htmlFor="ob-city"
              className="mb-2 block font-medium text-neutral-800"
            >
              City
            </label>
            <input
              id="ob-city"
              value={city}
              onChange={(e) => setCity(e.target.value)}
              placeholder="Columbus"
              maxLength={100}
              className={`${fieldClass} ${fieldBorder(errors.businessCity)}`}
            />
            <FieldError message={errors.businessCity} />
          </div>
          <div>
            <label
              htmlFor="ob-state"
              className="mb-2 block font-medium text-neutral-800"
            >
              State
            </label>
            <input
              id="ob-state"
              value={state}
              onChange={(e) => setState(e.target.value)}
              placeholder="OH"
              maxLength={50}
              className={`${fieldClass} ${fieldBorder(errors.businessState)}`}
            />
            <FieldError message={errors.businessState} />
          </div>
        </div>
        <p className="-mt-4 text-sm text-neutral-500">
          Shown to your customers on the invoice pay page.
        </p>
      </div>

      <StepFooter
        onSkip={() => void onSkip()}
        primaryLabel="Continue"
        busy={busy}
      />
    </form>
  );
}
