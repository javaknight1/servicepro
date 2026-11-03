import { useEffect, useRef, useState } from 'react';
import { useSearchParams } from 'react-router';
import { useMutation, useQuery } from '@tanstack/react-query';
import {
  Building2,
  CreditCard,
  ExternalLink,
  Landmark,
  ShieldCheck,
} from 'lucide-react';
import { DashboardLayout } from '@components/layout';
import {
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  ErrorState,
} from '@components/shared';
import type { BadgeVariant } from '@components/shared';
import { useTenantStore } from '@store';
import { paymentsApi, paymentsStatusQueryKey } from '@services/paymentsService';
import { getErrorMessage } from '@/utils/error';
import { redirectTo } from '@/utils/redirect';
import { useTenantPermissions } from '@/utils/tenantPermissions';
import type {
  PaymentFees,
  PaymentsState,
  PaymentsStatus,
} from '@/types/paymentsConnect';

/** The invoice the fee explainer uses ("$10 on a $5,000 bank payment"). */
const EXAMPLE_INVOICE = 5000;

const STATE_BADGE: Record<
  PaymentsState,
  { label: string; variant: BadgeVariant }
> = {
  not_connected: { label: 'Not connected', variant: 'neutral' },
  onboarding: { label: 'Setup in progress', variant: 'warning' },
  action_required: { label: 'Action needed', variant: 'warning' },
  restricted: { label: 'Restricted', variant: 'error' },
  active: { label: 'Connected', variant: 'success' },
};

function money(amount: number, currency = 'USD'): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency,
    minimumFractionDigits: amount % 1 === 0 ? 0 : 2,
  }).format(amount);
}

/** ServicePro's bank fee on an amount: ach_percent, capped at ach_cap. */
function bankFeeFor(amount: number, fees: PaymentFees): number {
  const fee = (amount * Number(fees.ach_percent)) / 100;
  return Math.min(Math.round(fee * 100) / 100, Number(fees.ach_cap));
}

function capabilityLabel(status: string): {
  label: string;
  variant: BadgeVariant;
} {
  switch (status) {
    case 'active':
      return { label: 'Active', variant: 'success' };
    case 'pending':
      return { label: 'Pending', variant: 'warning' };
    case 'restricted':
    case 'rejected':
      return { label: 'Not available', variant: 'error' };
    case 'unsupported':
      return { label: 'Not supported', variant: 'neutral' };
    default:
      return { label: 'Not set up', variant: 'neutral' };
  }
}

function connectLabel(state: PaymentsState): string {
  switch (state) {
    case 'not_connected':
      return 'Connect payouts with Stripe';
    case 'active':
      return 'Review details on Stripe';
    default:
      return 'Continue setup with Stripe';
  }
}

function StatusRow({ label, status }: { label: string; status: string }) {
  const { label: text, variant } = capabilityLabel(status);
  return (
    <div className="flex items-center justify-between py-2">
      <span className="text-sm text-neutral-600">{label}</span>
      <Badge variant={variant} size="sm">
        {text}
      </Badge>
    </div>
  );
}

function FeeExplainer({ fees }: { fees: PaymentFees }) {
  const exampleFee = bankFeeFor(EXAMPLE_INVOICE, fees);
  const uncapped = (EXAMPLE_INVOICE * Number(fees.ach_percent)) / 100;
  return (
    <Card variant="elevated" padding="lg">
      <CardHeader>
        <CardTitle>What you pay to get paid</CardTitle>
        <CardDescription>
          ServicePro&apos;s fee is taken from each online payment. It is never
          added to your customer&apos;s invoice.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div className="rounded-lg border border-neutral-200 p-4">
            <div className="flex items-center gap-2 text-sm font-medium text-neutral-700">
              <Landmark className="h-4 w-4 text-primary-600" />
              Bank transfer (ACH)
            </div>
            <p className="mt-2 text-2xl font-semibold text-neutral-900">
              {fees.ach_percent}%{' '}
              <span className="text-sm font-normal text-neutral-500">
                max {money(Number(fees.ach_cap), fees.currency)}
              </span>
            </p>
            <p className="mt-1 text-sm text-neutral-600">
              A {money(EXAMPLE_INVOICE, fees.currency)} invoice paid by bank
              costs you {money(exampleFee, fees.currency)}.
            </p>
          </div>
          <div className="rounded-lg border border-neutral-200 p-4">
            <div className="flex items-center gap-2 text-sm font-medium text-neutral-700">
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
        {uncapped > exampleFee && (
          <p className="mt-4 text-sm text-neutral-500">
            Without the cap, {fees.ach_percent}% of{' '}
            {money(EXAMPLE_INVOICE, fees.currency)} would be{' '}
            {money(uncapped, fees.currency)}. Stripe&apos;s own processing fees
            are billed to you by Stripe.
          </p>
        )}
      </CardContent>
    </Card>
  );
}

function StripeAccountCard({
  status,
  canManage,
  onConnect,
  isConnecting,
  connectError,
}: {
  status: PaymentsStatus;
  canManage: boolean;
  onConnect: () => void;
  isConnecting: boolean;
  connectError: string | null;
}) {
  const badge = STATE_BADGE[status.state] ?? STATE_BADGE.not_connected;
  return (
    <Card variant="elevated" padding="lg">
      <CardHeader>
        <div className="flex flex-wrap items-start justify-between gap-4">
          <div className="flex items-start gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-50">
              <Building2 className="h-5 w-5 text-primary-600" />
            </div>
            <div>
              <CardTitle>Stripe payouts</CardTitle>
              <CardDescription>
                Stripe verifies your business and deposits payments to your bank
                account. Takes about 5 minutes.
              </CardDescription>
            </div>
          </div>
          <Badge variant={badge.variant}>{badge.label}</Badge>
        </div>
      </CardHeader>
      <CardContent>
        {status.connected && (
          <div className="divide-y divide-neutral-100">
            <StatusRow
              label="Card payments"
              status={status.card_payments_status}
            />
            <StatusRow
              label="Bank payments (ACH)"
              status={status.ach_payments_status}
            />
            <StatusRow
              label="Payouts to your bank"
              status={status.payouts_status}
            />
          </div>
        )}

        {status.state === 'action_required' && (
          <p className="mt-4 text-sm text-warning-700">
            Stripe needs more information to keep payments and payouts running.
          </p>
        )}
        {status.state === 'restricted' && (
          <p className="mt-4 text-sm text-error-700">
            Stripe has paused payments or payouts on this account. Finish the
            steps on Stripe to restore them.
          </p>
        )}

        {status.state === 'active' && !status.online_payments_enabled && (
          <p className="mt-4 text-sm text-neutral-600">
            Online invoice payment isn&apos;t switched on yet, so invoices
            don&apos;t show a Pay button. Your account is ready for when it is.
          </p>
        )}
        {status.pay_links_active && (
          <p className="mt-4 flex items-center gap-2 text-sm text-success-700">
            <ShieldCheck className="h-4 w-4" />
            Customers can pay your invoices online by bank or card.
          </p>
        )}

        {connectError && (
          <p role="alert" className="mt-4 text-sm text-error-700">
            {connectError}
          </p>
        )}

        <div className="mt-6 flex flex-wrap items-center gap-3">
          {canManage ? (
            <Button onClick={onConnect} isLoading={isConnecting}>
              {connectLabel(status.state)}
            </Button>
          ) : (
            <p className="text-sm text-neutral-600">
              Only owners and admins can connect or change payouts.
            </p>
          )}
          {status.state === 'active' && (
            <a
              href="https://dashboard.stripe.com"
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-1 text-sm font-medium text-primary-700 hover:underline"
            >
              Open Stripe Dashboard
              <ExternalLink className="h-4 w-4" />
            </a>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

export function PaymentsSettingsPage() {
  const { currentTenant } = useTenantStore();
  const tenantId = currentTenant?.id;
  const { canManageBilling, isLoading: permissionsLoading } =
    useTenantPermissions(currentTenant);
  const [searchParams, setSearchParams] = useSearchParams();
  const onboarding = searchParams.get('onboarding');
  const [connectError, setConnectError] = useState<string | null>(null);

  const statusQuery = useQuery({
    queryKey: paymentsStatusQueryKey(tenantId),
    queryFn: async () => (await paymentsApi.getStatus(tenantId as string)).data,
    enabled: !!tenantId && canManageBilling,
  });

  const connect = useMutation({
    mutationFn: async () =>
      (await paymentsApi.connect(tenantId as string)).data,
    onSuccess: (data) => redirectTo(data.url),
    onError: (error: unknown) =>
      setConnectError(
        getErrorMessage(error, "Couldn't start Stripe setup. Please try again.")
      ),
  });

  // Stripe sends the user to ?onboarding=refresh when a link expired or was
  // already used: make a fresh one and go straight back (once).
  const refreshed = useRef(false);
  const { mutate: startConnect } = connect;
  useEffect(() => {
    if (onboarding !== 'refresh' || !tenantId || !canManageBilling) return;
    if (refreshed.current) return;
    refreshed.current = true;
    setSearchParams({}, { replace: true });
    startConnect();
  }, [onboarding, tenantId, canManageBilling, setSearchParams, startConnect]);

  const handleConnect = () => {
    setConnectError(null);
    connect.mutate();
  };

  const renderBody = () => {
    if (!tenantId) {
      return (
        <Card variant="elevated" padding="lg">
          <p className="text-neutral-600">Select an organization first.</p>
        </Card>
      );
    }
    if (!permissionsLoading && !canManageBilling) {
      return (
        <Card variant="elevated" padding="lg">
          <p className="text-neutral-600">
            Only owners and admins can see payouts and connect Stripe.
          </p>
        </Card>
      );
    }
    if (statusQuery.isError) {
      return (
        <ErrorState
          title="Couldn't load payments"
          message={getErrorMessage(statusQuery.error)}
          onRetry={() => statusQuery.refetch()}
        />
      );
    }
    if (!statusQuery.data) {
      return (
        <Card variant="elevated" padding="lg">
          <p className="text-neutral-500">Loading payments…</p>
        </Card>
      );
    }
    return (
      <div className="space-y-6">
        {onboarding === 'return' && statusQuery.data.state !== 'active' && (
          <div
            role="status"
            className="rounded-lg border border-primary-200 bg-primary-50 p-4 text-sm text-primary-800"
          >
            Thanks — Stripe is checking your details. This page updates when
            they&apos;re done; you can also continue setup any time.
          </div>
        )}
        <StripeAccountCard
          status={statusQuery.data}
          canManage={canManageBilling}
          onConnect={handleConnect}
          isConnecting={connect.isPending}
          connectError={connectError}
        />
        <FeeExplainer fees={statusQuery.data.fees} />
      </div>
    );
  };

  return (
    <DashboardLayout>
      <div className="mx-auto max-w-5xl px-4 py-8 sm:px-6 lg:px-8">
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-neutral-900">Payments</h1>
          <p className="mt-2 text-neutral-600">
            Online payments from your invoices, deposited to your bank by
            Stripe.
          </p>
        </div>
        {renderBody()}
      </div>
    </DashboardLayout>
  );
}
