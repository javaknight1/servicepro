import { useState } from 'react';
import { useMutation, useQuery } from '@tanstack/react-query';
import { Link } from 'react-router';
import { Clock, Sparkles } from 'lucide-react';
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@components/shared';
import {
  membershipApi,
  membershipTiersQueryKey,
  tenantMembershipQueryKey,
} from '@/services/membershipService';
import type {
  BillingCycle,
  MembershipTier,
  TenantMembership,
} from '@/types/membership';
import { getErrorMessage } from '@/utils/error';
import { redirectTo } from '@/utils/redirect';
import { BillingToggle } from './BillingToggle';
import { TierCard } from './TierCard';

/** The plans sold today (D15), in display order. */
const PLAN_ORDER = ['solo', 'team', 'pro'] as const;

interface PlansSectionProps {
  tenantId: string;
  /** membership.update: may choose a plan. */
  canChoose: boolean;
}

function TrialStatus({ membership }: { membership: TenantMembership }) {
  const trial = membership.trial;
  if (!trial) return null;
  const endsOn = new Date(trial.ends_at).toLocaleDateString();
  if (trial.ended) {
    return (
      <div
        role="status"
        className="flex items-start gap-3 rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900"
      >
        <Clock className="mt-0.5 h-4 w-4 shrink-0" />
        <p>
          Your free trial of Team ended on {endsOn}. Everything keeps working
          during the beta; choose a plan below to keep going after it.
        </p>
      </div>
    );
  }
  return (
    <div
      role="status"
      className="flex items-start gap-3 rounded-lg border border-primary-200 bg-primary-50 p-4 text-sm text-primary-900"
    >
      <Clock className="mt-0.5 h-4 w-4 shrink-0" />
      <p>
        <span className="font-semibold">
          {trial.days_left} {trial.days_left === 1 ? 'day' : 'days'} left
        </span>{' '}
        in your free Team trial (ends {endsOn}). No card needed until you choose
        a plan.
      </p>
    </div>
  );
}

/**
 * Plan picker (T210, D15): Solo / Team / Pro with a monthly-annual toggle,
 * trial days left, and "Choose plan" -> Stripe Checkout. The plan changes
 * once Stripe confirms the subscription (webhook), not on the way back.
 */
export function PlansSection({ tenantId, canChoose }: PlansSectionProps) {
  const [cycle, setCycle] = useState<BillingCycle>('monthly');

  const tiersQuery = useQuery({
    queryKey: membershipTiersQueryKey,
    queryFn: async () => (await membershipApi.getAllTiers()).data.tiers,
  });
  const membershipQuery = useQuery({
    queryKey: tenantMembershipQueryKey(tenantId),
    queryFn: async () =>
      (await membershipApi.getTenantMembership(tenantId)).data,
  });

  const checkout = useMutation({
    mutationFn: async (tier: MembershipTier) =>
      (
        await membershipApi.startCheckout(tenantId, {
          tier_id: tier.id,
          billing_cycle: cycle,
        })
      ).data,
    onSuccess: (data) => redirectTo(data.url),
  });

  const membership = membershipQuery.data;
  const plans = (tiersQuery.data ?? [])
    .filter((t) => (PLAN_ORDER as readonly string[]).includes(t.name))
    .sort(
      (a, b) =>
        PLAN_ORDER.indexOf(a.name as (typeof PLAN_ORDER)[number]) -
        PLAN_ORDER.indexOf(b.name as (typeof PLAN_ORDER)[number])
    );

  const hasPaidPlan = membership?.has_paid_subscription ?? false;
  const checkoutAvailable = membership?.checkout_available ?? false;
  const founding = membership?.founding_offer_available ?? false;
  const currentTierId = hasPaidPlan ? membership?.tier?.id : undefined;

  return (
    <Card variant="elevated" padding="lg">
      <CardHeader>
        <CardTitle>Plans</CardTitle>
        <CardDescription>
          One flat price, month to month, no contract.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-6">
          {membership && <TrialStatus membership={membership} />}

          {founding && cycle === 'monthly' && !hasPaidPlan && (
            <div className="flex items-start gap-3 rounded-lg border border-green-200 bg-green-50 p-4 text-sm text-green-900">
              <Sparkles className="mt-0.5 h-4 w-4 shrink-0" />
              <p>
                Founding offer: Team is $59/mo for life for our first customers.
                It&apos;s applied automatically at checkout.
              </p>
            </div>
          )}

          <div className="flex justify-center">
            <BillingToggle value={cycle} onChange={setCycle} />
          </div>

          {tiersQuery.isLoading ? (
            <p className="text-center text-sm text-neutral-500">
              Loading plans…
            </p>
          ) : tiersQuery.isError ? (
            <p className="text-center text-sm text-red-600">
              {getErrorMessage(tiersQuery.error)}
            </p>
          ) : (
            <div className="grid grid-cols-1 gap-6 pt-2 md:grid-cols-3">
              {plans.map((tier) => (
                <TierCard
                  key={tier.id}
                  tier={tier}
                  billingCycle={cycle}
                  isCurrentTier={tier.id === currentTierId}
                  highlighted={tier.name === 'team'}
                  ribbon={tier.name === 'team' ? 'Most teams' : undefined}
                  foundingOffer={founding && tier.name === 'team'}
                  showSelectButton={canChoose && !hasPaidPlan}
                  selectLabel="Choose plan"
                  disabled={!checkoutAvailable || checkout.isPending}
                  onSelect={(t) => checkout.mutate(t)}
                />
              ))}
            </div>
          )}

          {checkout.isError && (
            <p role="alert" className="text-sm text-red-600">
              {getErrorMessage(checkout.error)}
            </p>
          )}
          {canChoose && !hasPaidPlan && membership && !checkoutAvailable && (
            <p className="text-sm text-neutral-500">
              Online plan billing isn&apos;t switched on yet. Your trial keeps
              running in the meantime.
            </p>
          )}
          {!canChoose && !hasPaidPlan && (
            <p className="text-sm text-neutral-500">
              Ask an organization owner or admin to choose a plan.
            </p>
          )}
          {hasPaidPlan && canChoose && (
            <p className="text-sm text-neutral-600">
              To switch plans or billing cycle,{' '}
              <Link
                to="/settings/organization/membership/change"
                className="font-medium text-primary-600 hover:underline"
              >
                change your plan
              </Link>
              .
            </p>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
