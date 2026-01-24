import api from './api';
import type {
  MembershipTiersResponse,
  TenantMembership,
  UpdateMembershipRequest,
  SubscriptionHistoryResponse,
} from '@/types/membership';

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

  // Get subscription history
  getSubscriptionHistory: (tenantId: string) =>
    api.get<SubscriptionHistoryResponse>(
      `/v1/tenants/${tenantId}/membership/history`
    ),
};

export default membershipApi;
