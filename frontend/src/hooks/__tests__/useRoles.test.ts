import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';
import {
  roleKeys,
  permissionKeys,
  useRoles,
  useRole,
  usePermissions,
  useRolePermissions,
  useCreateRole,
  useDeleteRole,
  useUpdateRolePermissions,
  useAssignRole,
  useUnassignRole,
  useBulkAssignRoles,
  useExportAuditLogs,
} from '../useRoles';

// Mock roleApi
vi.mock('@services/roleApi', () => ({
  roleApi: {
    getRoles: vi.fn(),
    getRole: vi.fn(),
    getPermissions: vi.fn(),
    getRolePermissions: vi.fn(),
    createRole: vi.fn(),
    updateRole: vi.fn(),
    deleteRole: vi.fn(),
    updateRolePermissions: vi.fn(),
    assignRole: vi.fn(),
    unassignRole: vi.fn(),
    bulkAssignRoles: vi.fn(),
    getRoleAuditLogs: vi.fn(),
    exportAuditLogs: vi.fn(),
  },
}));

// Mock fileDownload
vi.mock('../../utils/fileDownload', () => ({
  downloadFile: vi.fn(),
}));

import { roleApi } from '@services/roleApi';

// Create wrapper with QueryClient
const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
  return ({ children }: { children: React.ReactNode }) =>
    React.createElement(QueryClientProvider, { client: queryClient }, children);
};

describe('useRoles hooks', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('roleKeys', () => {
    it('should generate all key', () => {
      expect(roleKeys.all).toEqual(['roles']);
    });

    it('should generate lists key', () => {
      expect(roleKeys.lists()).toEqual(['roles', 'list']);
    });

    it('should generate list key without filters', () => {
      expect(roleKeys.list()).toEqual(['roles', 'list', undefined]);
    });

    it('should generate list key with filters', () => {
      const filters = { search: 'admin', is_system: true };
      expect(roleKeys.list(filters)).toEqual(['roles', 'list', filters]);
    });

    it('should generate details key', () => {
      expect(roleKeys.details()).toEqual(['roles', 'detail']);
    });

    it('should generate detail key', () => {
      expect(roleKeys.detail('role-123')).toEqual([
        'roles',
        'detail',
        'role-123',
      ]);
    });

    it('should generate permissions key', () => {
      expect(roleKeys.permissions('role-123')).toEqual([
        'roles',
        'detail',
        'role-123',
        'permissions',
      ]);
    });

    it('should generate audit key without filters', () => {
      expect(roleKeys.audit()).toEqual(['roles', 'audit', undefined]);
    });

    it('should generate audit key with filters', () => {
      const filters = { role_id: 'role-123', action: 'create' };
      expect(roleKeys.audit(filters)).toEqual(['roles', 'audit', filters]);
    });
  });

  describe('permissionKeys', () => {
    it('should generate all key', () => {
      expect(permissionKeys.all).toEqual(['permissions']);
    });

    it('should generate list key', () => {
      expect(permissionKeys.list()).toEqual(['permissions', 'list']);
    });
  });

  describe('useRoles', () => {
    it('should fetch roles successfully', async () => {
      const mockRoles = [
        { id: 'role-1', name: 'Admin', is_system: true },
        { id: 'role-2', name: 'User', is_system: false },
      ];

      vi.mocked(roleApi.getRoles).mockResolvedValue({
        data: mockRoles,
      } as never);

      const { result } = renderHook(() => useRoles(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockRoles);
      expect(roleApi.getRoles).toHaveBeenCalledWith(undefined);
    });

    it('should fetch roles with filters', async () => {
      const filters = { search: 'admin' };
      const mockRoles = [{ id: 'role-1', name: 'Admin', is_system: true }];

      vi.mocked(roleApi.getRoles).mockResolvedValue({
        data: mockRoles,
      } as never);

      const { result } = renderHook(() => useRoles(filters), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(roleApi.getRoles).toHaveBeenCalledWith(filters);
    });
  });

  describe('useRole', () => {
    it('should fetch single role', async () => {
      const mockRole = { id: 'role-1', name: 'Admin', is_system: true };

      vi.mocked(roleApi.getRole).mockResolvedValue({
        data: mockRole,
      } as never);

      const { result } = renderHook(() => useRole('role-1'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockRole);
      expect(roleApi.getRole).toHaveBeenCalledWith('role-1');
    });

    it('should not fetch when id is empty', () => {
      const { result } = renderHook(() => useRole(''), {
        wrapper: createWrapper(),
      });

      expect(result.current.isLoading).toBe(false);
      expect(roleApi.getRole).not.toHaveBeenCalled();
    });
  });

  describe('usePermissions', () => {
    it('should fetch permissions', async () => {
      const mockPermissions = [
        { id: 'perm-1', name: 'users.read' },
        { id: 'perm-2', name: 'users.write' },
      ];

      vi.mocked(roleApi.getPermissions).mockResolvedValue({
        data: mockPermissions,
      } as never);

      const { result } = renderHook(() => usePermissions(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockPermissions);
    });
  });

  describe('useRolePermissions', () => {
    it('should fetch role permissions', async () => {
      const mockPermissions = ['perm-1', 'perm-2'];

      vi.mocked(roleApi.getRolePermissions).mockResolvedValue({
        data: mockPermissions,
      } as never);

      const { result } = renderHook(() => useRolePermissions('role-1'), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(result.current.data).toEqual(mockPermissions);
      expect(roleApi.getRolePermissions).toHaveBeenCalledWith('role-1');
    });

    it('should not fetch when roleId is empty', () => {
      const { result } = renderHook(() => useRolePermissions(''), {
        wrapper: createWrapper(),
      });

      expect(result.current.isLoading).toBe(false);
      expect(roleApi.getRolePermissions).not.toHaveBeenCalled();
    });
  });

  describe('useCreateRole', () => {
    it('should create role successfully', async () => {
      const newRole = { id: 'role-new', name: 'New Role', is_system: false };

      vi.mocked(roleApi.createRole).mockResolvedValue({
        data: newRole,
      } as never);

      const { result } = renderHook(() => useCreateRole(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ name: 'New Role', description: 'A new role' });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(roleApi.createRole).toHaveBeenCalledWith({
        name: 'New Role',
        description: 'A new role',
      });
    });
  });

  describe('useDeleteRole', () => {
    it('should delete role successfully', async () => {
      vi.mocked(roleApi.deleteRole).mockResolvedValue(undefined as never);

      const { result } = renderHook(() => useDeleteRole(), {
        wrapper: createWrapper(),
      });

      result.current.mutate('role-1');

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(roleApi.deleteRole).toHaveBeenCalledWith('role-1');
    });
  });

  describe('useUpdateRolePermissions', () => {
    it('should update role permissions successfully', async () => {
      vi.mocked(roleApi.updateRolePermissions).mockResolvedValue(
        undefined as never
      );

      const { result } = renderHook(() => useUpdateRolePermissions(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({
        roleId: 'role-1',
        permissionIds: ['perm-1', 'perm-2'],
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(roleApi.updateRolePermissions).toHaveBeenCalledWith('role-1', [
        'perm-1',
        'perm-2',
      ]);
    });
  });

  describe('useAssignRole', () => {
    it('should assign role to user successfully', async () => {
      vi.mocked(roleApi.assignRole).mockResolvedValue(undefined as never);

      const { result } = renderHook(() => useAssignRole(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({
        user_id: 'user-1',
        role_id: 'role-1',
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(roleApi.assignRole).toHaveBeenCalledWith({
        user_id: 'user-1',
        role_id: 'role-1',
      });
    });
  });

  describe('useUnassignRole', () => {
    it('should unassign role from user successfully', async () => {
      vi.mocked(roleApi.unassignRole).mockResolvedValue(undefined as never);

      const { result } = renderHook(() => useUnassignRole(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({
        userId: 'user-1',
        roleId: 'role-1',
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(roleApi.unassignRole).toHaveBeenCalledWith('user-1', 'role-1');
    });
  });

  describe('useBulkAssignRoles', () => {
    it('should bulk assign roles successfully', async () => {
      vi.mocked(roleApi.bulkAssignRoles).mockResolvedValue(undefined as never);

      const { result } = renderHook(() => useBulkAssignRoles(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({
        user_ids: ['user-1', 'user-2'],
        role_ids: ['role-1', 'role-2'],
      });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(roleApi.bulkAssignRoles).toHaveBeenCalledWith({
        user_ids: ['user-1', 'user-2'],
        role_ids: ['role-1', 'role-2'],
      });
    });
  });

  describe('useExportAuditLogs', () => {
    it('should export audit logs successfully', async () => {
      const mockBlob = new Blob(['csv,data']);
      vi.mocked(roleApi.exportAuditLogs).mockResolvedValue({
        data: mockBlob,
      } as never);

      const { result } = renderHook(() => useExportAuditLogs(), {
        wrapper: createWrapper(),
      });

      result.current.mutate({ role_id: 'role-1' });

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(roleApi.exportAuditLogs).toHaveBeenCalledWith({
        role_id: 'role-1',
      });
    });

    it('should export audit logs without filters', async () => {
      const mockBlob = new Blob(['csv,data']);
      vi.mocked(roleApi.exportAuditLogs).mockResolvedValue({
        data: mockBlob,
      } as never);

      const { result } = renderHook(() => useExportAuditLogs(), {
        wrapper: createWrapper(),
      });

      result.current.mutate(undefined);

      await waitFor(() => {
        expect(result.current.isSuccess).toBe(true);
      });

      expect(roleApi.exportAuditLogs).toHaveBeenCalledWith(undefined);
    });
  });
});
