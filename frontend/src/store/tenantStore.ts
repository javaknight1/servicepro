import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { tenantApi } from '@/services/tenantService';
import { getErrorMessage } from '@/utils/error';
import type {
  Tenant,
  TenantState,
  CreateTenantRequest,
  UpdateTenantRequest,
  AddMemberRequest,
} from '@/types/tenant';
import { clearQueryCaches } from '@/services/queryCacheRegistry';

const TENANT_STORAGE_KEY = 'tenant-storage';
const CURRENT_TENANT_KEY = 'current_tenant_id';

export const useTenantStore = create<TenantState>()(
  persist(
    (set, get) => ({
      currentTenant: null,
      tenants: [],
      isLoading: false,
      error: null,

      loadTenants: async () => {
        try {
          set({ isLoading: true, error: null });
          const response = await tenantApi.list();
          const tenants = response.data.tenants;

          set({ tenants, isLoading: false });

          // Re-validate the persisted current tenant against the freshly
          // loaded list: it may belong to a previous user on this browser.
          const { currentTenant } = get();
          const stillMember = currentTenant
            ? tenants.find((t) => t.id === currentTenant.id)
            : undefined;

          if (stillMember) {
            localStorage.setItem(CURRENT_TENANT_KEY, stillMember.id);
            set({ currentTenant: stillMember });
          } else if (tenants.length > 0) {
            // Check if there's a saved tenant ID
            const savedTenantId = localStorage.getItem(CURRENT_TENANT_KEY);
            const nextTenant =
              (savedTenantId && tenants.find((t) => t.id === savedTenantId)) ||
              tenants[0];

            if (currentTenant && currentTenant.id !== nextTenant.id) {
              clearQueryCaches();
            }
            // Keep the request header (read from storage) in sync with the UI
            localStorage.setItem(CURRENT_TENANT_KEY, nextTenant.id);
            set({ currentTenant: nextTenant });
          } else {
            localStorage.removeItem(CURRENT_TENANT_KEY);
            set({ currentTenant: null });
          }
        } catch (error) {
          set({
            error: getErrorMessage(error, 'Failed to load organizations'),
            isLoading: false,
          });
          throw error;
        }
      },

      setCurrentTenant: (tenant: Tenant) => {
        if (get().currentTenant?.id !== tenant.id) {
          // Cached server data belongs to the previous tenant
          clearQueryCaches();
        }
        localStorage.setItem(CURRENT_TENANT_KEY, tenant.id);
        set({ currentTenant: tenant });
      },

      switchTenant: async (tenantId: string) => {
        const { tenants } = get();
        const tenant = tenants.find((t) => t.id === tenantId);

        if (!tenant) {
          throw new Error('Organization not found');
        }

        if (get().currentTenant?.id !== tenantId) {
          // Cached server data (memory + persisted) belongs to the old tenant
          clearQueryCaches();
        }

        localStorage.setItem(CURRENT_TENANT_KEY, tenantId);
        set({ currentTenant: tenant });
      },

      createTenant: async (data: CreateTenantRequest) => {
        try {
          set({ isLoading: true, error: null });
          const response = await tenantApi.create(data);
          const newTenant = response.data;

          set((state) => ({
            tenants: [...state.tenants, newTenant],
            isLoading: false,
          }));

          return newTenant;
        } catch (error) {
          set({
            error: getErrorMessage(error, 'Failed to create organization'),
            isLoading: false,
          });
          throw error;
        }
      },

      updateTenant: async (id: string, data: UpdateTenantRequest) => {
        try {
          set({ isLoading: true, error: null });
          const response = await tenantApi.update(id, data);
          const updatedTenant = response.data;

          set((state) => ({
            tenants: state.tenants.map((t) =>
              t.id === id ? updatedTenant : t
            ),
            currentTenant:
              state.currentTenant?.id === id
                ? updatedTenant
                : state.currentTenant,
            isLoading: false,
          }));

          return updatedTenant;
        } catch (error) {
          set({
            error: getErrorMessage(error, 'Failed to update organization'),
            isLoading: false,
          });
          throw error;
        }
      },

      deleteTenant: async (id: string) => {
        try {
          set({ isLoading: true, error: null });
          await tenantApi.delete(id);

          const { tenants, currentTenant } = get();
          const remainingTenants = tenants.filter((t) => t.id !== id);

          set({
            tenants: remainingTenants,
            currentTenant:
              currentTenant?.id === id
                ? remainingTenants[0] || null
                : currentTenant,
            isLoading: false,
          });
        } catch (error) {
          set({
            error: getErrorMessage(error, 'Failed to delete organization'),
            isLoading: false,
          });
          throw error;
        }
      },

      getMembers: async (tenantId: string) => {
        const response = await tenantApi.getMembers(tenantId);
        return response.data.members;
      },

      addMember: async (tenantId: string, data: AddMemberRequest) => {
        const response = await tenantApi.addMember(tenantId, data);
        return response.data;
      },

      removeMember: async (tenantId: string, userId: string) => {
        await tenantApi.removeMember(tenantId, userId);
      },

      updateMemberRole: async (
        tenantId: string,
        userId: string,
        roleId: string
      ) => {
        await tenantApi.updateMemberRole(tenantId, userId, roleId);
      },

      clearError: () => {
        set({ error: null });
      },

      reset: () => {
        localStorage.removeItem(CURRENT_TENANT_KEY);
        set({
          currentTenant: null,
          tenants: [],
          isLoading: false,
          error: null,
        });
      },
    }),
    {
      name: TENANT_STORAGE_KEY,
      partialize: (state) => ({
        currentTenant: state.currentTenant,
        tenants: state.tenants,
      }),
    }
  )
);
