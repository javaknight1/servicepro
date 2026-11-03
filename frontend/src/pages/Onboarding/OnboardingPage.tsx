import { useState } from 'react';
import { Navigate, useNavigate } from 'react-router';
import { Check } from 'lucide-react';
import { useAuthStore, useTenantStore } from '@store';
import { getTenantErrorMessage } from '@/utils/error';
import { useTenantPermissions } from '@/utils/tenantPermissions';
import { mergeSettings } from '@/utils/tenantSettings';
import { cn } from '@utils/cn';
import type { TenantSettings } from '@/types/tenant';
import { BusinessStep } from './BusinessStep';
import { BillingStep } from './BillingStep';
import { GetPaidStep } from './GetPaidStep';
import type { StepSave } from './steps';

const STEPS = [
  { title: 'Your business', subtitle: 'Name, trade and team size' },
  { title: 'How you bill', subtitle: 'Terms, invoice style, tax' },
  { title: 'Get paid', subtitle: 'Payouts, import, accounting' },
] as const;

const LAST_STEP = STEPS.length;

function clampStep(step: number | undefined): number {
  return Math.min(LAST_STEP, Math.max(1, step ?? 1));
}

function StepRail({ current }: { current: number }) {
  return (
    <ol className="mt-8 space-y-0">
      {STEPS.map((s, i) => {
        const n = i + 1;
        const done = n < current;
        const active = n === current;
        return (
          <li key={s.title} className="relative flex gap-4 pb-8 last:pb-0">
            {n < LAST_STEP && (
              <span
                aria-hidden="true"
                className={cn(
                  'absolute left-[19px] top-11 h-[calc(100%-48px)] w-0.5',
                  done ? 'bg-primary-600' : 'bg-neutral-200'
                )}
              />
            )}
            <span
              className={cn(
                'flex h-10 w-10 shrink-0 items-center justify-center rounded-full border-2 text-[15px] font-medium',
                done && 'border-primary-600 bg-primary-600 text-white',
                active && 'border-primary-600 bg-primary-50 text-primary-700',
                !done && !active && 'border-neutral-300 text-neutral-500'
              )}
            >
              {done ? <Check className="h-5 w-5" /> : n}
            </span>
            <span aria-current={active ? 'step' : undefined}>
              <span
                className={cn(
                  'block font-medium',
                  active ? 'text-primary-700' : 'text-neutral-900'
                )}
              >
                {s.title}
              </span>
              <span className="block text-sm text-neutral-600">
                {s.subtitle}
              </span>
            </span>
          </li>
        );
      })}
    </ol>
  );
}

/**
 * Onboarding wizard (T235, mockups 02–04), shown after a new organization is
 * created. Every step saves into the tenant's settings and records where to
 * resume (`onboardingStep`); finishing or skipping the last step sets
 * `onboardingCompletedAt`. "Save and finish later" leaves it resumable from
 * the dashboard.
 */
export function OnboardingPage() {
  const navigate = useNavigate();
  const { currentTenant, updateTenant } = useTenantStore();
  const email = useAuthStore((state) => state.user?.email);
  const { canUpdateTenant, canManageBilling, canManageTaxRates } =
    useTenantPermissions(currentTenant);
  const [step, setStep] = useState(() =>
    clampStep(currentTenant?.settings?.onboardingStep)
  );
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (!currentTenant) {
    return <Navigate to="/settings/organization/new" replace />;
  }
  const tenant = currentTenant;

  const save = async (patch: Partial<TenantSettings>, name?: string) => {
    setBusy(true);
    try {
      await updateTenant(tenant.id, {
        ...(name !== undefined ? { name } : {}),
        settings: mergeSettings(tenant, patch),
      });
    } finally {
      setBusy(false);
    }
  };

  const goTo = (next: number) => {
    setStep(next);
    window.scrollTo?.(0, 0);
  };

  /** Saves the step's answers, then moves on. Errors go back to the step. */
  const continueFrom = async ({ settings, name }: StepSave) => {
    const next = Math.min(LAST_STEP, step + 1);
    await save({ ...settings, onboardingStep: next }, name);
    goTo(next);
  };

  /** Moves on without the step's answers (only the resume point is kept). */
  const skipFrom = async () => {
    const next = Math.min(LAST_STEP, step + 1);
    setError(null);
    try {
      await save({ onboardingStep: next });
    } catch (err: unknown) {
      setError(getTenantErrorMessage(err, 'Could not save your progress'));
      return;
    }
    goTo(next);
  };

  const finish = async () => {
    setError(null);
    try {
      await save({
        onboardingStep: LAST_STEP,
        onboardingCompletedAt: new Date().toISOString(),
      });
    } catch (err: unknown) {
      setError(getTenantErrorMessage(err, 'Could not finish setup'));
      return;
    }
    navigate('/dashboard');
  };

  const saveAndExit = async () => {
    setError(null);
    try {
      await save({ onboardingStep: step });
    } catch (err: unknown) {
      setError(getTenantErrorMessage(err, 'Could not save your progress'));
      return;
    }
    navigate('/dashboard');
  };

  return (
    <div className="min-h-screen bg-neutral-50">
      <header className="flex flex-wrap items-center justify-between gap-2 border-b border-neutral-200 bg-white px-4 py-4 sm:px-8">
        <span className="text-2xl font-bold tracking-tight text-primary-600">
          ServicePro
        </span>
        <span className="text-sm text-neutral-600">
          {email && (
            <span className="hidden sm:inline">Signed in as {email} · </span>
          )}
          <button
            type="button"
            onClick={() => void saveAndExit()}
            disabled={busy}
            className="font-medium text-neutral-900 hover:underline"
          >
            Save and finish later
          </button>
        </span>
      </header>

      <div className="grid grid-cols-1 lg:grid-cols-[320px_minmax(0,1fr)]">
        <aside className="hidden border-r border-neutral-200 bg-white px-8 py-10 lg:block lg:min-h-[calc(100vh-73px)]">
          <p className="text-neutral-600">Set up your workspace</p>
          <p className="text-neutral-600">
            Step {step} of {LAST_STEP}
          </p>
          <StepRail current={step} />
          <div className="mt-10 rounded-xl border border-neutral-200 bg-neutral-50 p-4">
            <p className="font-medium text-neutral-900">
              Team plan · free trial
            </p>
            <p className="mt-1 text-sm text-neutral-600">
              30 days free, no card needed. Unlimited users. Pick a plan when
              the trial ends.
            </p>
          </div>
        </aside>

        <main className="px-4 py-8 sm:px-8 lg:px-16 lg:py-12">
          <div className="mx-auto max-w-3xl">
            <p className="mb-4 text-sm text-neutral-600 lg:hidden">
              Step {step} of {LAST_STEP} · {STEPS[step - 1].title}
            </p>
            {error && (
              <div
                role="alert"
                className="mb-6 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700"
              >
                {error}
              </div>
            )}
            {!canUpdateTenant ? (
              <div className="rounded-xl border border-neutral-200 bg-white p-6">
                <p className="text-neutral-700">
                  Only organization admins can set up the workspace.
                </p>
                <button
                  type="button"
                  onClick={() => navigate('/dashboard')}
                  className="mt-4 rounded-lg bg-primary-600 px-4 py-2 text-white"
                >
                  Go to the dashboard
                </button>
              </div>
            ) : step === 1 ? (
              <BusinessStep
                tenant={tenant}
                onContinue={continueFrom}
                onSkip={skipFrom}
                busy={busy}
              />
            ) : step === 2 ? (
              <BillingStep
                tenant={tenant}
                canManageTaxRates={canManageTaxRates}
                onContinue={continueFrom}
                onSkip={skipFrom}
                onBack={() => goTo(1)}
                busy={busy}
              />
            ) : (
              <GetPaidStep
                tenant={tenant}
                canConnect={canManageBilling}
                onFinish={finish}
                onBack={() => goTo(2)}
                busy={busy}
              />
            )}
          </div>
        </main>
      </div>
    </div>
  );
}
