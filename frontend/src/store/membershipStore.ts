import { create } from 'zustand';
import { membershipApi } from '@/services/membershipService';
import type {
  MembershipTier,
  TenantMembership,
  MembershipState,
  BillingCycle,
} from '@/types/membership';

export const useMembershipStore = create<MembershipState>((set, get) => ({
  tiers: [],
  currentMembership: null,
  isLoading: false,
  error: null,

  loadTiers: async () => {
    try {
      set({ isLoading: true, error: null });
      const response = await membershipApi.getAllTiers();
      set({ tiers: response.data.tiers, isLoading: false });
    } catch (error: any) {
      set({
        error:
          error.response?.data?.message || 'Failed to load membership tiers',
        isLoading: false,
      });
      throw error;
    }
  },

  loadMembership: async (tenantId: string) => {
    try {
      set({ isLoading: true, error: null });
      const response = await membershipApi.getTenantMembership(tenantId);
      set({ currentMembership: response.data, isLoading: false });
    } catch (error: any) {
      // If no subscription found, set null but don't treat as error
      if (error.response?.status === 404) {
        set({ currentMembership: null, isLoading: false });
        return;
      }
      set({
        error: error.response?.data?.message || 'Failed to load membership',
        isLoading: false,
      });
      throw error;
    }
  },

  updateMembership: async (
    tenantId: string,
    tierId: string,
    billingCycle: BillingCycle
  ) => {
    try {
      set({ isLoading: true, error: null });
      const response = await membershipApi.updateMembership(tenantId, {
        tier_id: tierId,
        billing_cycle: billingCycle,
      });
      set({ currentMembership: response.data, isLoading: false });
    } catch (error: any) {
      set({
        error: error.response?.data?.message || 'Failed to update membership',
        isLoading: false,
      });
      throw error;
    }
  },

  clearError: () => {
    set({ error: null });
  },

  reset: () => {
    set({
      tiers: [],
      currentMembership: null,
      isLoading: false,
      error: null,
    });
  },
}));
