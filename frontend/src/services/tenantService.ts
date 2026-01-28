import api from './api';
import type {
  Tenant,
  TenantMember,
  CreateTenantRequest,
  UpdateTenantRequest,
  AddMemberRequest,
  TenantListResponse,
  TenantMembersResponse,
  Invitation,
  InviteMemberRequest,
} from '@/types/tenant';

// Tenant API endpoints
export const tenantApi = {
  // Tenant CRUD
  list: () => api.get<TenantListResponse>('/v1/tenants'),

  get: (id: string) => api.get<Tenant>(`/v1/tenants/${id}`),

  create: (data: CreateTenantRequest) => api.post<Tenant>('/v1/tenants', data),

  update: (id: string, data: UpdateTenantRequest) =>
    api.put<Tenant>(`/v1/tenants/${id}`, data),

  delete: (id: string) => api.delete(`/v1/tenants/${id}`),

  // Member management
  getMembers: (tenantId: string) =>
    api.get<TenantMembersResponse>(`/v1/tenants/${tenantId}/members`),

  addMember: (tenantId: string, data: AddMemberRequest) =>
    api.post<TenantMember>(`/v1/tenants/${tenantId}/members`, data),

  removeMember: (tenantId: string, userId: string) =>
    api.delete(`/v1/tenants/${tenantId}/members/${userId}`),

  updateMemberRole: (tenantId: string, userId: string, roleId: string) =>
    api.put(`/v1/tenants/${tenantId}/members/${userId}/role`, {
      role_id: roleId,
    }),

  // Invitation management
  inviteMember: (tenantId: string, data: InviteMemberRequest) =>
    api.post<Invitation>(`/v1/tenants/${tenantId}/invitations`, data),

  getPendingInvitations: (tenantId: string) =>
    api.get<Invitation[]>(`/v1/tenants/${tenantId}/invitations`),

  cancelInvitation: (tenantId: string, invitationId: string) =>
    api.delete(`/v1/tenants/${tenantId}/invitations/${invitationId}`),

  resendInvitation: (tenantId: string, invitationId: string) =>
    api.post(`/v1/tenants/${tenantId}/invitations/${invitationId}/resend`),
};

// Public invitation endpoints (no auth required for token lookup)
export const invitationApi = {
  getByToken: (token: string) =>
    api.get<Invitation>(`/v1/invitations/${token}`),

  accept: (token: string) => api.post(`/v1/invitations/${token}/accept`),
};

export default tenantApi;
