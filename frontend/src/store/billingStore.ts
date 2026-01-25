import { create } from 'zustand';
import { billingApi } from '@/services';
import type {
  SavedPaymentMethod,
  BillingEvent,
  BillingConfig,
} from '@/types/payment';

interface BillingState {
  // Config
  config: BillingConfig | null;
  isStripeEnabled: boolean;

  // Payment Methods
  paymentMethods: SavedPaymentMethod[];
  isLoadingMethods: boolean;
  methodsError: string | null;

  // Billing History
  billingEvents: BillingEvent[];
  isLoadingHistory: boolean;
  historyError: string | null;
  historyTotal: number;
  historyPage: number;
  historyPageSize: number;

  // Actions
  loadConfig: () => Promise<void>;
  loadPaymentMethods: (tenantId: string) => Promise<void>;
  addPaymentMethod: (
    tenantId: string,
    paymentMethodId: string,
    setAsDefault?: boolean
  ) => Promise<SavedPaymentMethod>;
  setDefaultPaymentMethod: (
    tenantId: string,
    paymentMethodId: string
  ) => Promise<void>;
  deletePaymentMethod: (
    tenantId: string,
    paymentMethodId: string
  ) => Promise<void>;
  loadBillingHistory: (tenantId: string, page?: number) => Promise<void>;
  clearError: () => void;
}

export const useBillingStore = create<BillingState>((set, get) => ({
  // Initial state
  config: null,
  isStripeEnabled: false,

  paymentMethods: [],
  isLoadingMethods: false,
  methodsError: null,

  billingEvents: [],
  isLoadingHistory: false,
  historyError: null,
  historyTotal: 0,
  historyPage: 1,
  historyPageSize: 20,

  // Actions
  loadConfig: async () => {
    try {
      const response = await billingApi.getConfig();
      set({
        config: response.data,
        isStripeEnabled: response.data.stripe_enabled,
      });
    } catch (error) {
      console.error('[BillingStore] Failed to load config:', error);
      set({ isStripeEnabled: false });
    }
  },

  loadPaymentMethods: async (tenantId: string) => {
    set({ isLoadingMethods: true, methodsError: null });
    try {
      const response = await billingApi.getPaymentMethods(tenantId);
      set({
        paymentMethods: response.data.payment_methods,
        isLoadingMethods: false,
      });
    } catch (error) {
      const message =
        error instanceof Error
          ? error.message
          : 'Failed to load payment methods';
      set({ methodsError: message, isLoadingMethods: false });
    }
  },

  addPaymentMethod: async (
    tenantId: string,
    paymentMethodId: string,
    setAsDefault = false
  ) => {
    try {
      const response = await billingApi.addPaymentMethod(tenantId, {
        payment_method_id: paymentMethodId,
        set_as_default: setAsDefault,
      });

      // Reload payment methods to get updated list
      await get().loadPaymentMethods(tenantId);

      return response.data;
    } catch (error) {
      const message =
        error instanceof Error ? error.message : 'Failed to add payment method';
      set({ methodsError: message });
      throw error;
    }
  },

  setDefaultPaymentMethod: async (
    tenantId: string,
    paymentMethodId: string
  ) => {
    try {
      await billingApi.setDefaultPaymentMethod(tenantId, paymentMethodId);

      // Update local state
      set((state) => ({
        paymentMethods: state.paymentMethods.map((pm) => ({
          ...pm,
          is_default: pm.id === paymentMethodId,
        })),
      }));
    } catch (error) {
      const message =
        error instanceof Error
          ? error.message
          : 'Failed to set default payment method';
      set({ methodsError: message });
      throw error;
    }
  },

  deletePaymentMethod: async (tenantId: string, paymentMethodId: string) => {
    try {
      await billingApi.deletePaymentMethod(tenantId, paymentMethodId);

      // Remove from local state
      set((state) => ({
        paymentMethods: state.paymentMethods.filter(
          (pm) => pm.id !== paymentMethodId
        ),
      }));
    } catch (error) {
      const message =
        error instanceof Error
          ? error.message
          : 'Failed to delete payment method';
      set({ methodsError: message });
      throw error;
    }
  },

  loadBillingHistory: async (tenantId: string, page = 1) => {
    const { historyPageSize } = get();
    set({ isLoadingHistory: true, historyError: null });

    try {
      const response = await billingApi.getBillingHistory(
        tenantId,
        page,
        historyPageSize
      );
      set({
        billingEvents: response.data.events,
        historyTotal: response.data.total,
        historyPage: response.data.page,
        isLoadingHistory: false,
      });
    } catch (error) {
      const message =
        error instanceof Error
          ? error.message
          : 'Failed to load billing history';
      set({ historyError: message, isLoadingHistory: false });
    }
  },

  clearError: () => {
    set({ methodsError: null, historyError: null });
  },
}));

export default useBillingStore;
