import { useState } from 'react';
import { useMutation, useQuery } from '@tanstack/react-query';
import {
  ChevronRight,
  CreditCard,
  FileSpreadsheet,
  Landmark,
} from 'lucide-react';
import { Link } from 'react-router';
import { paymentsApi, paymentsStatusQueryKey } from '@services/paymentsService';
import {
  quickbooksApi,
  quickbooksStatusQueryKey,
} from '@services/quickbooksService';
import { getErrorMessage } from '@/utils/error';
import { redirectTo } from '@/utils/redirect';
import type { Tenant } from '@/types/tenant';
import { StepFooter } from './parts';
import type { StepProps } from './steps';

function ComingSoon() {
  return (
    <span className="rounded-full bg-neutral-100 px-2 py-0.5 text-xs font-medium text-neutral-600">
      Coming soon
    </span>
  );
}

/** "$10.00" → "$10"; whole amounts without cents. */
function shortMoney(amount: string, currency: string): string {
  const n = Number(amount);
  try {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency,
      minimumFractionDigits: n % 1 === 0 ? 0 : 2,
    }).format(n);
  } catch {
    return amount;
  }
}

/**
 * Onboarding 3 — get paid (mockup 04): Stripe payouts through the T169
 * Connect endpoint, imports from Jobber / Housecall Pro / CSV (T193, the
 * import page) and QuickBooks (T192): a link to Settings → Integrations
 * when the server has an Intuit app, "Coming soon" until then (D11:
 * nothing that isn't available is offered as if it were). Connecting can fail
 * today (Connect isn't enabled on the platform yet): the error is shown and
 * the step can still be skipped or finished.
 */
export function GetPaidStep({
  tenant,
  canConnect,
  onFinish,
  onBack,
  busy,
}: Omit<StepProps, 'onContinue' | 'onSkip'> & {
  tenant: Tenant;
  canConnect: boolean;
  onFinish: () => Promise<void>;
}) {
  const [connectError, setConnectError] = useState<string | null>(null);

  const statusQuery = useQuery({
    queryKey: paymentsStatusQueryKey(tenant.id),
    queryFn: async () => (await paymentsApi.getStatus(tenant.id)).data,
    enabled: canConnect,
    retry: false,
  });
  const status = statusQuery.data;
  const fees = status?.fees;

  const connect = useMutation({
    mutationFn: async () => (await paymentsApi.connect(tenant.id)).data,
    onSuccess: (data) => redirectTo(data.url),
    onError: (err: unknown) =>
      setConnectError(
        getErrorMessage(err, 'Stripe payouts could not be started')
      ),
  });

  const connected = status?.state === 'active';

  // QuickBooks status (integrations.manage: owners and admins, like payouts).
  const quickbooksQuery = useQuery({
    queryKey: quickbooksStatusQueryKey(tenant.id),
    queryFn: async () => (await quickbooksApi.getStatus(tenant.id)).data,
    enabled: canConnect,
    retry: false,
  });
  const quickbooks = quickbooksQuery.data;
  const quickbooksReady = !!quickbooks?.configured;
  const quickbooksConnected = quickbooks?.status === 'connected';

  return (
    <div>
      <h1 className="text-3xl font-bold tracking-tight text-neutral-900 md:text-4xl">
        Get paid
      </h1>
      <p className="mt-2 text-lg text-neutral-600">
        Connect payouts so customers can pay invoices online, then bring your
        existing customers with you.
      </p>

      <section className="mt-8 rounded-2xl border-2 border-primary-600 bg-white p-5 sm:p-6">
        <div className="flex flex-wrap items-start gap-4">
          <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-neutral-200 text-2xl font-bold text-[#635bff]">
            S
          </div>
          <div className="min-w-0 flex-1">
            <h2 className="text-lg font-semibold text-neutral-900">
              Online payments and payouts
            </h2>
            <p className="mt-1 text-neutral-600">
              Stripe verifies your business and deposits payments to your bank
              account.
            </p>
          </div>
          {canConnect &&
            (connected ? (
              <span className="rounded-full bg-success-50 px-3 py-1 text-sm font-medium text-success-700">
                Connected
              </span>
            ) : (
              <button
                type="button"
                onClick={() => {
                  setConnectError(null);
                  connect.mutate();
                }}
                disabled={connect.isPending}
                className="rounded-lg bg-primary-600 px-5 py-2.5 text-[15px] font-medium text-white hover:bg-primary-700 disabled:opacity-60"
              >
                {connect.isPending
                  ? 'Opening Stripe…'
                  : 'Connect payouts with Stripe'}
              </button>
            ))}
        </div>

        {connectError && (
          <div
            role="alert"
            className="mt-4 rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900"
          >
            {connectError}. You can skip this for now and connect later in
            Settings → Payments.
          </div>
        )}
        {!canConnect && (
          <p className="mt-4 text-sm text-neutral-600">
            Only organization admins can connect payouts.
          </p>
        )}

        {fees && (
          <div className="mt-5 grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div className="rounded-xl border border-neutral-200 p-4">
              <div className="flex items-center gap-2 font-medium text-neutral-800">
                <Landmark className="h-4 w-4 text-primary-600" />
                Bank transfer (ACH)
              </div>
              <p className="mt-2 text-2xl font-semibold text-neutral-900">
                {fees.ach_percent}%{' '}
                <span className="text-sm font-normal text-neutral-500">
                  max {shortMoney(fees.ach_cap, fees.currency)}
                </span>
              </p>
            </div>
            <div className="rounded-xl border border-neutral-200 p-4">
              <div className="flex items-center gap-2 font-medium text-neutral-800">
                <CreditCard className="h-4 w-4 text-primary-600" />
                Card
              </div>
              <p className="mt-2 text-2xl font-semibold text-neutral-900">
                Stripe + {fees.card_percent}%
              </p>
              <p className="mt-1 text-sm text-neutral-600">
                Stripe&apos;s standard card rate, plus {fees.card_percent}%.
              </p>
            </div>
          </div>
        )}
      </section>

      <div className="mt-5 grid grid-cols-1 gap-5 md:grid-cols-2">
        <section className="rounded-2xl border border-neutral-200 bg-white p-5">
          <h2 className="text-lg font-semibold text-neutral-900">
            Bring your customers
          </h2>
          <p className="mt-1 text-sm text-neutral-600">
            Customers, sites, jobs and open invoices.
          </p>
          <ul className="mt-4 space-y-2">
            {[
              { icon: 'J', label: 'Import from Jobber', source: 'jobber' },
              {
                icon: 'H',
                label: 'Import from Housecall Pro',
                source: 'housecall_pro',
              },
              { icon: null, label: 'Upload a CSV', source: 'generic' },
            ].map((row) => (
              <li key={row.label}>
                <Link
                  to={`/settings/import?source=${row.source}`}
                  className="flex items-center gap-3 rounded-xl border border-neutral-200 p-3 text-neutral-800 hover:border-primary-600"
                >
                  <span className="flex h-8 w-8 items-center justify-center rounded-lg bg-neutral-100 font-medium text-neutral-600">
                    {row.icon ?? <FileSpreadsheet className="h-4 w-4" />}
                  </span>
                  <span className="flex-1">{row.label}</span>
                  <ChevronRight className="h-4 w-4 text-neutral-400" />
                </Link>
              </li>
            ))}
          </ul>
        </section>

        <section className="flex flex-col rounded-2xl border border-neutral-200 bg-white p-5">
          <div className="flex items-center gap-3">
            <span className="flex h-10 w-10 items-center justify-center rounded-xl bg-[#2ca01c] text-sm font-bold text-white">
              qb
            </span>
            <h2 className="text-lg font-semibold text-neutral-900">
              QuickBooks Online
            </h2>
            {!quickbooksReady && <ComingSoon />}
            {quickbooksConnected && (
              <span className="rounded-full bg-success-50 px-2 py-0.5 text-xs font-medium text-success-700">
                Connected
              </span>
            )}
          </div>
          <p className="mt-3 text-sm text-neutral-600">
            Send customers, invoices and payments to QuickBooks automatically.
            {!quickbooksReady && " We'll let you know when it's ready."}
          </p>
          <div className="mt-auto pt-4">
            {quickbooksReady ? (
              <Link
                to="/settings/organization?tab=integrations"
                className="inline-block rounded-lg border border-neutral-300 px-4 py-2 text-sm font-medium text-neutral-800 hover:border-primary-600"
              >
                {quickbooksConnected
                  ? 'QuickBooks settings'
                  : 'Connect QuickBooks'}
              </Link>
            ) : (
              <button
                type="button"
                disabled
                className="rounded-lg border border-neutral-300 px-4 py-2 text-sm font-medium text-neutral-400"
              >
                Connect QuickBooks
              </button>
            )}
          </div>
        </section>
      </div>

      <StepFooter
        onBack={onBack}
        onSkip={() => void onFinish()}
        skipLabel="Skip for now"
        note="You can connect these later in Settings."
        primaryLabel="Finish setup"
        onPrimary={() => void onFinish()}
        busy={busy}
      />
    </div>
  );
}
