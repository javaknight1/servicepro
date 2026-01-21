import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { tenantApi } from '@/services/tenantService';
import type {
  Tenant,
  TenantMember,
  TenantState,
  CreateTenantRequest,
  UpdateTenantRequest,
  AddMemberRequest,
} from '@/types/tenant';

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

          // If no current tenant is set, use the first one
          const { currentTenant } = get();
          if (!currentTenant && tenants.length > 0) {
            // Check if there's a saved tenant ID
            const savedTenantId = localStorage.getItem(CURRENT_TENANT_KEY);
            const savedTenant = savedTenantId
              ? tenants.find((t) => t.id === savedTenantId)
              : null;

            set({ currentTenant: savedTenant || tenants[0] });
          }
        } catch (error: any) {
          set({
            error:
              error.response?.data?.message || 'Failed to load organizations',
            isLoading: false,
          });
          throw error;
        }
      },

      setCurrentTenant: (tenant: Tenant) => {
        localStorage.setItem(CURRENT_TENANT_KEY, tenant.id);
        set({ currentTenant: tenant });
      },

      switchTenant: async (tenantId: string) => {
        const { tenants } = get();
        const tenant = tenants.find((t) => t.id === tenantId);

        if (!tenant) {
          throw new Error('Organization not found');
        }

        localStorage.setItem(CURRENT_TENANT_KEY, tenantId);
        set({ currentTenant: tenant });

        // Optionally reload other stores/data when tenant changes
        // This could trigger a page reload or data refresh
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
        } catch (error: any) {
          set({
            error:
              error.response?.data?.message || 'Failed to create organization',
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
        } catch (error: any) {
          set({
            error:
              error.response?.data?.message || 'Failed to update organization',
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
        } catch (error: any) {
          set({
            error:
              error.response?.data?.message || 'Failed to delete organization',
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
