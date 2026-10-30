import api from './api';
import type { Role, Permission, RoleFilters } from '@app-types/role';

// Only the role endpoints the API actually serves (GET /roles,
// GET /roles/{id}/permissions, GET /permissions). Calls to endpoints that
// never existed were removed in T257.
export const roleApi = {
  getRoles: (filters?: RoleFilters) =>
    api.get<Role[]>('/v1/roles', { params: filters }),

  getPermissions: () => api.get<Permission[]>('/v1/permissions'),

  getRolePermissions: (roleId: string) =>
    api.get<Permission[]>(`/v1/roles/${roleId}/permissions`),
};
