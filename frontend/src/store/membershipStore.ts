import { create } from 'zustand';
import { AxiosError } from 'axios';
import { membershipApi } from '@/services/membershipService';
import type { MembershipState, BillingCycle } from '@/types/membership';

interface ApiErrorResponse {
  message?: string;
}

export const useMembershipStore = create<MembershipState>((set) => ({
  tiers: [],
  currentMembership: null,
  isLoading: false,
  error: null,

  loadTiers: async () => {
    try {
      set({ isLoading: true, error: null });
      const response = await membershipApi.getAllTiers();
      set({ tiers: response.data.tiers, isLoading: false });
    } catch (error) {
      const axiosError = error as AxiosError<ApiErrorResponse>;
      set({
        error:
          axiosError.response?.data?.message ||
          'Failed to load membership tiers',
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
    } catch (error) {
      const axiosError = error as AxiosError<ApiErrorResponse>;
      // If no subscription found, set null but don't treat as error
      if (axiosError.response?.status === 404) {
        set({ currentMembership: null, isLoading: false });
        return;
      }
      set({
        error:
          axiosError.response?.data?.message || 'Failed to load membership',
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
    } catch (error) {
      const axiosError = error as AxiosError<ApiErrorResponse>;
      set({
        error:
          axiosError.response?.data?.message || 'Failed to update membership',
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
