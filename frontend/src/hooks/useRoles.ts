import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { roleApi } from '@services/roleApi';
import type {
  Role,
  Permission,
  CreateRoleRequest,
  UpdateRoleRequest,
  AssignRoleRequest,
  BulkAssignRolesRequest,
  RoleFilters,
  AuditFilters,
} from '@types/role';

// Query keys
export const roleKeys = {
  all: ['roles'] as const,
  lists: () => [...roleKeys.all, 'list'] as const,
  list: (filters?: RoleFilters) => [...roleKeys.lists(), filters] as const,
  details: () => [...roleKeys.all, 'detail'] as const,
  detail: (id: string) => [...roleKeys.details(), id] as const,
  permissions: (id: string) => [...roleKeys.detail(id), 'permissions'] as const,
  audit: (filters?: AuditFilters) =>
    [...roleKeys.all, 'audit', filters] as const,
};

export const permissionKeys = {
  all: ['permissions'] as const,
  list: () => [...permissionKeys.all, 'list'] as const,
};

// Queries
export function useRoles(filters?: RoleFilters) {
  return useQuery({
    queryKey: roleKeys.list(filters),
    queryFn: () => roleApi.getRoles(filters).then((res) => res.data),
  });
}

export function useRole(id: string) {
  return useQuery({
    queryKey: roleKeys.detail(id),
    queryFn: () => roleApi.getRole(id).then((res) => res.data),
    enabled: !!id,
  });
}

export function usePermissions() {
  return useQuery({
    queryKey: permissionKeys.list(),
    queryFn: () => roleApi.getPermissions().then((res) => res.data),
  });
}

export function useRolePermissions(roleId: string) {
  return useQuery({
    queryKey: roleKeys.permissions(roleId),
    queryFn: () => roleApi.getRolePermissions(roleId).then((res) => res.data),
    enabled: !!roleId,
  });
}

export function useRoleAuditLogs(filters?: AuditFilters) {
  return useQuery({
    queryKey: roleKeys.audit(filters),
    queryFn: () => roleApi.getRoleAuditLogs(filters).then((res) => res.data),
  });
}

// Mutations
export function useCreateRole() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateRoleRequest) =>
      roleApi.createRole(data).then((res) => res.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: roleKeys.lists() });
    },
  });
}

export function useUpdateRole() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateRoleRequest }) =>
      roleApi.updateRole(id, data).then((res) => res.data),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: roleKeys.detail(variables.id),
      });
      queryClient.invalidateQueries({ queryKey: roleKeys.lists() });
    },
  });
}

export function useDeleteRole() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => roleApi.deleteRole(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: roleKeys.lists() });
    },
  });
}

export function useUpdateRolePermissions() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      roleId,
      permissionIds,
    }: {
      roleId: string;
      permissionIds: string[];
    }) => roleApi.updateRolePermissions(roleId, permissionIds),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({
        queryKey: roleKeys.permissions(variables.roleId),
      });
      queryClient.invalidateQueries({
        queryKey: roleKeys.detail(variables.roleId),
      });
    },
  });
}

export function useAssignRole() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: AssignRoleRequest) => roleApi.assignRole(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] });
    },
  });
}

export function useUnassignRole() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ userId, roleId }: { userId: string; roleId: string }) =>
      roleApi.unassignRole(userId, roleId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] });
    },
  });
}

export function useBulkAssignRoles() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: BulkAssignRolesRequest) => roleApi.bulkAssignRoles(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['users'] });
    },
  });
}

export function useExportAuditLogs() {
  return useMutation({
    mutationFn: (filters?: AuditFilters) =>
      roleApi.exportAuditLogs(filters).then((res) => {
        // Create blob and download
        const url = window.URL.createObjectURL(new Blob([res.data]));
        const link = document.createElement('a');
        link.href = url;
        link.setAttribute(
          'download',
          `audit-logs-${new Date().toISOString()}.csv`
        );
        document.body.appendChild(link);
        link.click();
        link.remove();
      }),
  });
}
