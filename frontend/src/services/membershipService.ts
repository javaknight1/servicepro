import api from './api';
import type {
  MembershipTiersResponse,
  TenantMembership,
  UpdateMembershipRequest,
  SubscriptionHistoryResponse,
  PreviewSubscriptionChangeRequest,
  SubscriptionChangePreview,
  PlanCheckoutRequest,
  PlanCheckoutResponse,
} from '@/types/membership';

/** React Query keys for plans and a tenant's membership. */
export const membershipTiersQueryKey = ['membership-tiers'] as const;
export const tenantMembershipQueryKey = (tenantId: string | null | undefined) =>
  ['tenant-membership', tenantId ?? null] as const;

// Membership API endpoints
export const membershipApi = {
  // Get all available membership tiers (public)
  getAllTiers: () => api.get<MembershipTiersResponse>('/v1/membership-tiers'),

  // Get tenant's current membership
  getTenantMembership: (tenantId: string) =>
    api.get<TenantMembership>(`/v1/tenants/${tenantId}/membership`),

  // Update tenant's membership tier
  updateMembership: (tenantId: string, data: UpdateMembershipRequest) =>
    api.put<TenantMembership>(`/v1/tenants/${tenantId}/membership`, data),

  // Start a Stripe Checkout session for a plan (T210); redirect to its url.
  startCheckout: (tenantId: string, data: PlanCheckoutRequest) =>
    api.post<PlanCheckoutResponse>(
      `/v1/tenants/${tenantId}/membership/checkout`,
      data
    ),

  // Get subscription history
  getSubscriptionHistory: (tenantId: string) =>
    api.get<SubscriptionHistoryResponse>(
      `/v1/tenants/${tenantId}/membership/history`
    ),

  // Preview subscription change
  previewSubscriptionChange: (
    tenantId: string,
    data: PreviewSubscriptionChangeRequest
  ) =>
    api.post<SubscriptionChangePreview>(
      `/v1/tenants/${tenantId}/membership/preview`,
      data
    ),
};

export default membershipApi;
