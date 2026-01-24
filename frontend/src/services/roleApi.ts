import api from './api';
import type {
  Role,
  Permission,
  RoleAuditLog,
  CreateRoleRequest,
  UpdateRoleRequest,
  AssignRoleRequest,
  BulkAssignRolesRequest,
  RoleFilters,
  AuditFilters,
} from '@app-types/role';

export const roleApi = {
  // Role CRUD operations
  getRoles: (filters?: RoleFilters) =>
    api.get<Role[]>('/v1/roles', { params: filters }),

  getRole: (id: string) => api.get<Role>(`/v1/roles/${id}`),

  createRole: (data: CreateRoleRequest) => api.post<Role>('/v1/roles', data),

  updateRole: (id: string, data: UpdateRoleRequest) =>
    api.put<Role>(`/v1/roles/${id}`, data),

  deleteRole: (id: string) => api.delete(`/v1/roles/${id}`),

  // Permission operations
  getPermissions: () => api.get<Permission[]>('/v1/permissions'),

  getRolePermissions: (roleId: string) =>
    api.get<Permission[]>(`/v1/roles/${roleId}/permissions`),

  addPermissionToRole: (roleId: string, permissionId: string) =>
    api.post(`/v1/roles/${roleId}/permissions/${permissionId}`),

  removePermissionFromRole: (roleId: string, permissionId: string) =>
    api.delete(`/v1/roles/${roleId}/permissions/${permissionId}`),

  updateRolePermissions: (roleId: string, permissionIds: string[]) =>
    api.put(`/v1/roles/${roleId}/permissions`, {
      permission_ids: permissionIds,
    }),

  // Role assignment operations
  assignRole: (data: AssignRoleRequest) => api.post('/v1/roles/assign', data),

  unassignRole: (userId: string, roleId: string) =>
    api.delete(`/v1/roles/unassign/${userId}/${roleId}`),

  bulkAssignRoles: (data: BulkAssignRolesRequest) =>
    api.post('/v1/roles/bulk-assign', data),

  getUserRoles: (userId: string) =>
    api.get<Role[]>(`/v1/users/${userId}/roles`),

  // Audit operations
  getRoleAuditLogs: (filters?: AuditFilters) =>
    api.get<RoleAuditLog[]>('/v1/roles/audit', { params: filters }),

  exportAuditLogs: (filters?: AuditFilters) =>
    api.get('/v1/roles/audit/export', {
      params: filters,
      responseType: 'blob',
    }),
};
