import { describe, it, expect, beforeEach, vi } from 'vitest';
import { act } from '@testing-library/react';
import { useTenantStore } from '../tenantStore';
import type { Tenant } from '@/types/tenant';

// Mock tenantApi
vi.mock('@/services/tenantService', () => ({
  tenantApi: {
    list: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    getMembers: vi.fn(),
    addMember: vi.fn(),
    removeMember: vi.fn(),
    updateMemberRole: vi.fn(),
  },
}));

// Import after mocking
import { tenantApi } from '@/services/tenantService';

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: vi.fn((key: string) => store[key] || null),
    setItem: vi.fn((key: string, value: string) => {
      store[key] = value;
    }),
    removeItem: vi.fn((key: string) => {
      delete store[key];
    }),
    clear: vi.fn(() => {
      store = {};
    }),
  };
})();

Object.defineProperty(window, 'localStorage', {
  value: localStorageMock,
});

const mockTenant1: Tenant = {
  id: 'tenant-1',
  name: 'Test Org 1',
  slug: 'test-org-1',
  is_active: true,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
};

const mockTenant2: Tenant = {
  id: 'tenant-2',
  name: 'Test Org 2',
  slug: 'test-org-2',
  is_active: true,
  created_at: '2024-01-02T00:00:00Z',
  updated_at: '2024-01-02T00:00:00Z',
};

describe('tenantStore', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorageMock.clear();

    // Reset the store state before each test
    act(() => {
      useTenantStore.setState({
        currentTenant: null,
        tenants: [],
        isLoading: false,
        error: null,
      });
    });
  });

  describe('initial state', () => {
    it('should have correct initial values', () => {
      const state = useTenantStore.getState();

      expect(state.currentTenant).toBeNull();
      expect(state.tenants).toEqual([]);
      expect(state.isLoading).toBe(false);
      expect(state.error).toBeNull();
    });
  });

  describe('loadTenants', () => {
    it('should load tenants and set current tenant to first one', async () => {
      vi.mocked(tenantApi.list).mockResolvedValue({
        data: { tenants: [mockTenant1, mockTenant2] },
      });

      await act(async () => {
        await useTenantStore.getState().loadTenants();
      });

      const state = useTenantStore.getState();
      expect(state.tenants).toHaveLength(2);
      expect(state.currentTenant).toEqual(mockTenant1);
      expect(state.isLoading).toBe(false);
      expect(state.error).toBeNull();
    });

    it('should use saved tenant if available', async () => {
      localStorageMock.setItem('current_tenant_id', 'tenant-2');

      vi.mocked(tenantApi.list).mockResolvedValue({
        data: { tenants: [mockTenant1, mockTenant2] },
      });

      await act(async () => {
        await useTenantStore.getState().loadTenants();
      });

      const state = useTenantStore.getState();
      expect(state.currentTenant).toEqual(mockTenant2);
    });

    it('should set loading state during fetch', async () => {
      let resolvePromise: (value: unknown) => void;
      const promise = new Promise((resolve) => {
        resolvePromise = resolve;
      });

      vi.mocked(tenantApi.list).mockReturnValue(promise as never);

      act(() => {
        useTenantStore.getState().loadTenants();
      });

      expect(useTenantStore.getState().isLoading).toBe(true);

      await act(async () => {
        resolvePromise!({ data: { tenants: [] } });
        await promise;
      });

      expect(useTenantStore.getState().isLoading).toBe(false);
    });

    it('should set error on failure', async () => {
      vi.mocked(tenantApi.list).mockRejectedValue(new Error('Network error'));

      await expect(
        act(async () => {
          await useTenantStore.getState().loadTenants();
        })
      ).rejects.toThrow();

      const state = useTenantStore.getState();
      expect(state.error).toBe('Network error');
      expect(state.isLoading).toBe(false);
    });
  });

  describe('setCurrentTenant', () => {
    it('should set current tenant and save to localStorage', () => {
      act(() => {
        useTenantStore.getState().setCurrentTenant(mockTenant1);
      });

      expect(useTenantStore.getState().currentTenant).toEqual(mockTenant1);
      expect(localStorageMock.setItem).toHaveBeenCalledWith(
        'current_tenant_id',
        'tenant-1'
      );
    });
  });

  describe('switchTenant', () => {
    beforeEach(() => {
      act(() => {
        useTenantStore.setState({
          tenants: [mockTenant1, mockTenant2],
          currentTenant: mockTenant1,
        });
      });
    });

    it('should switch to existing tenant', async () => {
      await act(async () => {
        await useTenantStore.getState().switchTenant('tenant-2');
      });

      expect(useTenantStore.getState().currentTenant).toEqual(mockTenant2);
    });

    it('should throw error for non-existent tenant', async () => {
      await expect(
        act(async () => {
          await useTenantStore.getState().switchTenant('non-existent');
        })
      ).rejects.toThrow('Organization not found');
    });
  });

  describe('createTenant', () => {
    it('should create tenant and add to list', async () => {
      vi.mocked(tenantApi.create).mockResolvedValue({ data: mockTenant1 });

      await act(async () => {
        await useTenantStore.getState().createTenant({
          name: 'Test Org 1',
          slug: 'test-org-1',
        });
      });

      const state = useTenantStore.getState();
      expect(state.tenants).toContainEqual(mockTenant1);
      expect(state.isLoading).toBe(false);
    });

    it('should set error on failure', async () => {
      vi.mocked(tenantApi.create).mockRejectedValue(new Error('Create failed'));

      await expect(
        act(async () => {
          await useTenantStore.getState().createTenant({
            name: 'Test',
            slug: 'test',
          });
        })
      ).rejects.toThrow();

      expect(useTenantStore.getState().error).toBe('Create failed');
    });
  });

  describe('updateTenant', () => {
    beforeEach(() => {
      act(() => {
        useTenantStore.setState({
          tenants: [mockTenant1, mockTenant2],
          currentTenant: mockTenant1,
        });
      });
    });

    it('should update tenant in list', async () => {
      const updatedTenant = { ...mockTenant1, name: 'Updated Name' };
      vi.mocked(tenantApi.update).mockResolvedValue({ data: updatedTenant });

      await act(async () => {
        await useTenantStore.getState().updateTenant('tenant-1', {
          name: 'Updated Name',
        });
      });

      const state = useTenantStore.getState();
      expect(state.tenants.find((t) => t.id === 'tenant-1')?.name).toBe(
        'Updated Name'
      );
    });

    it('should update currentTenant if it was updated', async () => {
      const updatedTenant = { ...mockTenant1, name: 'Updated Name' };
      vi.mocked(tenantApi.update).mockResolvedValue({ data: updatedTenant });

      await act(async () => {
        await useTenantStore.getState().updateTenant('tenant-1', {
          name: 'Updated Name',
        });
      });

      expect(useTenantStore.getState().currentTenant?.name).toBe(
        'Updated Name'
      );
    });

    it('should not update currentTenant when updating a different tenant', async () => {
      const updatedTenant2 = { ...mockTenant2, name: 'Updated Tenant 2' };
      vi.mocked(tenantApi.update).mockResolvedValue({ data: updatedTenant2 });

      await act(async () => {
        await useTenantStore.getState().updateTenant('tenant-2', {
          name: 'Updated Tenant 2',
        });
      });

      // currentTenant should still be mockTenant1
      expect(useTenantStore.getState().currentTenant).toEqual(mockTenant1);
      // But tenant-2 should be updated in the list
      expect(
        useTenantStore.getState().tenants.find((t) => t.id === 'tenant-2')?.name
      ).toBe('Updated Tenant 2');
    });

    it('should set error on update failure', async () => {
      vi.mocked(tenantApi.update).mockRejectedValue(new Error('Update failed'));

      await expect(
        act(async () => {
          await useTenantStore.getState().updateTenant('tenant-1', {
            name: 'Updated Name',
          });
        })
      ).rejects.toThrow();

      const state = useTenantStore.getState();
      expect(state.error).toBe('Update failed');
      expect(state.isLoading).toBe(false);
    });
  });

  describe('deleteTenant', () => {
    beforeEach(() => {
      act(() => {
        useTenantStore.setState({
          tenants: [mockTenant1, mockTenant2],
          currentTenant: mockTenant1,
        });
      });
    });

    it('should remove tenant from list', async () => {
      vi.mocked(tenantApi.delete).mockResolvedValue(undefined);

      await act(async () => {
        await useTenantStore.getState().deleteTenant('tenant-2');
      });

      const state = useTenantStore.getState();
      expect(state.tenants).toHaveLength(1);
      expect(state.tenants).not.toContainEqual(mockTenant2);
    });

    it('should switch currentTenant if deleted', async () => {
      vi.mocked(tenantApi.delete).mockResolvedValue(undefined);

      await act(async () => {
        await useTenantStore.getState().deleteTenant('tenant-1');
      });

      expect(useTenantStore.getState().currentTenant).toEqual(mockTenant2);
    });

    it('should not change currentTenant when deleting a different tenant', async () => {
      vi.mocked(tenantApi.delete).mockResolvedValue(undefined);

      await act(async () => {
        await useTenantStore.getState().deleteTenant('tenant-2');
      });

      // currentTenant should still be mockTenant1
      expect(useTenantStore.getState().currentTenant).toEqual(mockTenant1);
      // tenant-2 should be removed from list
      expect(useTenantStore.getState().tenants).toHaveLength(1);
    });

    it('should set currentTenant to null when deleting the last tenant', async () => {
      // Set up state with only one tenant
      act(() => {
        useTenantStore.setState({
          tenants: [mockTenant1],
          currentTenant: mockTenant1,
        });
      });

      vi.mocked(tenantApi.delete).mockResolvedValue(undefined);

      await act(async () => {
        await useTenantStore.getState().deleteTenant('tenant-1');
      });

      // currentTenant should be null when no tenants remain
      expect(useTenantStore.getState().currentTenant).toBeNull();
      expect(useTenantStore.getState().tenants).toHaveLength(0);
    });

    it('should set error on delete failure', async () => {
      vi.mocked(tenantApi.delete).mockRejectedValue(new Error('Delete failed'));

      await expect(
        act(async () => {
          await useTenantStore.getState().deleteTenant('tenant-1');
        })
      ).rejects.toThrow();

      const state = useTenantStore.getState();
      expect(state.error).toBe('Delete failed');
      expect(state.isLoading).toBe(false);
    });
  });

  describe('member management', () => {
    it('should get members for a tenant', async () => {
      const mockMembers = [
        { user_id: 'user-1', email: 'user1@example.com' },
        { user_id: 'user-2', email: 'user2@example.com' },
      ];
      vi.mocked(tenantApi.getMembers).mockResolvedValue({
        data: { members: mockMembers },
      });

      let members;
      await act(async () => {
        members = await useTenantStore.getState().getMembers('tenant-1');
      });

      expect(members).toEqual(mockMembers);
      expect(tenantApi.getMembers).toHaveBeenCalledWith('tenant-1');
    });

    it('should add member to tenant', async () => {
      const newMember = { user_id: 'user-1', email: 'user1@example.com' };
      vi.mocked(tenantApi.addMember).mockResolvedValue({ data: newMember });

      let result;
      await act(async () => {
        result = await useTenantStore.getState().addMember('tenant-1', {
          email: 'user1@example.com',
          role_id: 'role-1',
        });
      });

      expect(result).toEqual(newMember);
      expect(tenantApi.addMember).toHaveBeenCalledWith('tenant-1', {
        email: 'user1@example.com',
        role_id: 'role-1',
      });
    });

    it('should remove member from tenant', async () => {
      vi.mocked(tenantApi.removeMember).mockResolvedValue(undefined);

      await act(async () => {
        await useTenantStore.getState().removeMember('tenant-1', 'user-1');
      });

      expect(tenantApi.removeMember).toHaveBeenCalledWith('tenant-1', 'user-1');
    });

    it('should update member role', async () => {
      vi.mocked(tenantApi.updateMemberRole).mockResolvedValue(undefined);

      await act(async () => {
        await useTenantStore
          .getState()
          .updateMemberRole('tenant-1', 'user-1', 'role-2');
      });

      expect(tenantApi.updateMemberRole).toHaveBeenCalledWith(
        'tenant-1',
        'user-1',
        'role-2'
      );
    });
  });

  describe('clearError', () => {
    it('should clear error', () => {
      act(() => {
        useTenantStore.setState({ error: 'Some error' });
        useTenantStore.getState().clearError();
      });

      expect(useTenantStore.getState().error).toBeNull();
    });
  });

  describe('reset', () => {
    it('should reset to initial state', () => {
      act(() => {
        useTenantStore.setState({
          currentTenant: mockTenant1,
          tenants: [mockTenant1, mockTenant2],
          isLoading: true,
          error: 'Some error',
        });
        useTenantStore.getState().reset();
      });

      const state = useTenantStore.getState();
      expect(state.currentTenant).toBeNull();
      expect(state.tenants).toEqual([]);
      expect(state.isLoading).toBe(false);
      expect(state.error).toBeNull();
    });
  });
});
