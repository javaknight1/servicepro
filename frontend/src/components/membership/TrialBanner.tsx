import { useContext } from 'react';
import { QueryClientContext, useQuery } from '@tanstack/react-query';
import { Link } from 'react-router';
import { Clock } from 'lucide-react';
import { useTenantStore } from '@store';
import {
  membershipApi,
  tenantMembershipQueryKey,
} from '@/services/membershipService';
import { canReadPlan, useTenantPermissions } from '@/utils/tenantPermissions';

function TrialEndedBanner({ tenantId }: { tenantId: string }) {
  const { data } = useQuery({
    queryKey: tenantMembershipQueryKey(tenantId),
    queryFn: async () =>
      (await membershipApi.getTenantMembership(tenantId)).data,
    retry: false,
    staleTime: 5 * 60 * 1000,
  });

  if (!data?.trial?.ended || data.has_paid_subscription) return null;
  return (
    <div
      role="status"
      className="flex flex-wrap items-center gap-2 border-b border-amber-200 bg-amber-50 px-4 py-2 text-sm text-amber-900"
    >
      <Clock className="h-4 w-4 shrink-0" />
      <span>Your free trial has ended. Everything still works for now.</span>
      <Link
        to="/settings/organization?tab=membership"
        className="font-medium underline hover:no-underline"
      >
        Choose a plan
      </Link>
    </div>
  );
}

/** Renders the banner only for callers who may read the plan (T272). */
function PlanGatedTrialBanner() {
  const { currentTenant } = useTenantStore();
  const allowed = canReadPlan(useTenantPermissions(currentTenant));
  const tenantId = currentTenant?.id;
  if (!allowed || typeof tenantId !== 'string') return null;
  return <TrialEndedBanner tenantId={tenantId} />;
}

/**
 * Non-blocking app-wide banner once the no-card trial has ended and no plan
 * was chosen (D15: no lockout during the invite-only beta). Silent on errors,
 * absent where no QueryClient is provided, and never shown to (or requested
 * for) roles that can't read the plan, such as technicians (T272).
 */
export function TrialBanner() {
  const client = useContext(QueryClientContext);
  if (!client) return null;
  return <PlanGatedTrialBanner />;
}
