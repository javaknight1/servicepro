import { describe, it, expect, vi, beforeEach } from 'vitest';
import { tenantApi, invitationApi } from '../tenantService';
import api from '../api';

// Mock the api module
vi.mock('../api', () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}));

describe('tenantService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('tenantApi', () => {
    describe('list', () => {
      it('should call GET /v1/tenants', async () => {
        const mockResponse = { data: { tenants: [] } };
        vi.mocked(api.get).mockResolvedValue(mockResponse);

        await tenantApi.list();

        expect(api.get).toHaveBeenCalledWith('/v1/tenants');
      });
    });

    describe('get', () => {
      it('should call GET /v1/tenants/:id', async () => {
        const mockResponse = { data: { id: 'tenant-1', name: 'Test' } };
        vi.mocked(api.get).mockResolvedValue(mockResponse);

        await tenantApi.get('tenant-1');

        expect(api.get).toHaveBeenCalledWith('/v1/tenants/tenant-1');
      });
    });

    describe('create', () => {
      it('should call POST /v1/tenants with data', async () => {
        const createData = { name: 'New Tenant', slug: 'new-tenant' };
        const mockResponse = { data: { id: 'new-id', ...createData } };
        vi.mocked(api.post).mockResolvedValue(mockResponse);

        await tenantApi.create(createData);

        expect(api.post).toHaveBeenCalledWith('/v1/tenants', createData);
      });
    });

    describe('update', () => {
      it('should call PUT /v1/tenants/:id with data', async () => {
        const updateData = { name: 'Updated Tenant' };
        const mockResponse = { data: { id: 'tenant-1', ...updateData } };
        vi.mocked(api.put).mockResolvedValue(mockResponse);

        await tenantApi.update('tenant-1', updateData);

        expect(api.put).toHaveBeenCalledWith(
          '/v1/tenants/tenant-1',
          updateData
        );
      });
    });

    describe('delete', () => {
      it('should call DELETE /v1/tenants/:id', async () => {
        vi.mocked(api.delete).mockResolvedValue({});

        await tenantApi.delete('tenant-1');

        expect(api.delete).toHaveBeenCalledWith('/v1/tenants/tenant-1');
      });
    });

    describe('getMembers', () => {
      it('should call GET /v1/tenants/:id/members', async () => {
        const mockResponse = { data: { members: [] } };
        vi.mocked(api.get).mockResolvedValue(mockResponse);

        await tenantApi.getMembers('tenant-1');

        expect(api.get).toHaveBeenCalledWith('/v1/tenants/tenant-1/members');
      });
    });

    describe('addMember', () => {
      it('should call POST /v1/tenants/:id/members with data', async () => {
        const memberData = { user_id: 'user-1', role_id: 'role-1' };
        const mockResponse = { data: { id: 'member-1', ...memberData } };
        vi.mocked(api.post).mockResolvedValue(mockResponse);

        await tenantApi.addMember('tenant-1', memberData);

        expect(api.post).toHaveBeenCalledWith(
          '/v1/tenants/tenant-1/members',
          memberData
        );
      });
    });

    describe('removeMember', () => {
      it('should call DELETE /v1/tenants/:tenantId/members/:userId', async () => {
        vi.mocked(api.delete).mockResolvedValue({});

        await tenantApi.removeMember('tenant-1', 'user-1');

        expect(api.delete).toHaveBeenCalledWith(
          '/v1/tenants/tenant-1/members/user-1'
        );
      });
    });

    describe('updateMemberRole', () => {
      it('should call PUT /v1/tenants/:tenantId/members/:userId/role', async () => {
        vi.mocked(api.put).mockResolvedValue({});

        await tenantApi.updateMemberRole('tenant-1', 'user-1', 'role-2');

        expect(api.put).toHaveBeenCalledWith(
          '/v1/tenants/tenant-1/members/user-1/role',
          { role_id: 'role-2' }
        );
      });
    });

    describe('inviteMember', () => {
      it('should call POST /v1/tenants/:id/invitations with data', async () => {
        const inviteData = { email: 'test@example.com', role_id: 'role-1' };
        const mockResponse = { data: { id: 'invite-1', ...inviteData } };
        vi.mocked(api.post).mockResolvedValue(mockResponse);

        await tenantApi.inviteMember('tenant-1', inviteData);

        expect(api.post).toHaveBeenCalledWith(
          '/v1/tenants/tenant-1/invitations',
          inviteData
        );
      });
    });

    describe('getPendingInvitations', () => {
      it('should call GET /v1/tenants/:id/invitations', async () => {
        const mockResponse = { data: [] };
        vi.mocked(api.get).mockResolvedValue(mockResponse);

        await tenantApi.getPendingInvitations('tenant-1');

        expect(api.get).toHaveBeenCalledWith(
          '/v1/tenants/tenant-1/invitations'
        );
      });
    });

    describe('cancelInvitation', () => {
      it('should call DELETE /v1/tenants/:tenantId/invitations/:invitationId', async () => {
        vi.mocked(api.delete).mockResolvedValue({});

        await tenantApi.cancelInvitation('tenant-1', 'invite-1');

        expect(api.delete).toHaveBeenCalledWith(
          '/v1/tenants/tenant-1/invitations/invite-1'
        );
      });
    });

    describe('resendInvitation', () => {
      it('should call POST /v1/tenants/:tenantId/invitations/:invitationId/resend', async () => {
        vi.mocked(api.post).mockResolvedValue({});

        await tenantApi.resendInvitation('tenant-1', 'invite-1');

        expect(api.post).toHaveBeenCalledWith(
          '/v1/tenants/tenant-1/invitations/invite-1/resend'
        );
      });
    });
  });

  describe('invitationApi', () => {
    describe('getByToken', () => {
      it('should call GET /v1/invitations/:token', async () => {
        const mockResponse = { data: { id: 'invite-1', token: 'abc123' } };
        vi.mocked(api.get).mockResolvedValue(mockResponse);

        await invitationApi.getByToken('abc123');

        expect(api.get).toHaveBeenCalledWith('/v1/invitations/abc123');
      });
    });

    describe('accept', () => {
      it('should call POST /v1/invitations/:token/accept', async () => {
        vi.mocked(api.post).mockResolvedValue({});

        await invitationApi.accept('abc123');

        expect(api.post).toHaveBeenCalledWith('/v1/invitations/abc123/accept');
      });
    });
  });
});
