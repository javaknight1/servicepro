import { useContext } from 'react';
import { QueryClientContext, useQuery } from '@tanstack/react-query';
import { Link } from 'react-router';
import { Clock } from 'lucide-react';
import { useTenantStore } from '@store';
import {
  membershipApi,
  tenantMembershipQueryKey,
} from '@/services/membershipService';

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

/**
 * Non-blocking app-wide banner once the no-card trial has ended and no plan
 * was chosen (D15: no lockout during the invite-only beta). Silent on errors,
 * and absent where no QueryClient is provided.
 */
export function TrialBanner() {
  const client = useContext(QueryClientContext);
  const { currentTenant } = useTenantStore();
  const tenantId = currentTenant?.id;
  if (!client || typeof tenantId !== 'string') return null;
  return <TrialEndedBanner tenantId={tenantId} />;
}
