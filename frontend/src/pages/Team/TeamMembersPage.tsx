import { useState, useEffect } from 'react';
import { DashboardLayout } from '@components/layout';
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  Button,
  Input,
  Modal,
  Badge,
  Avatar,
} from '@components/shared';
import { useTenantStore } from '@store';
import { roleApi } from '@services/roleApi';
import { tenantApi } from '@services/tenantService';
import { getErrorMessage } from '@/utils/error';
import {
  UserPlus,
  Trash2,
  Shield,
  Search,
  Building2,
  Mail,
  RefreshCw,
  X,
  Clock,
} from 'lucide-react';
import { getDisplayName } from '@/utils/avatar';
import type { TenantMember, Invitation } from '@/types/tenant';
import type { Role } from '@/types/role';

export function TeamMembersPage() {
  const { currentTenant, getMembers, removeMember, updateMemberRole } =
    useTenantStore();

  const [members, setMembers] = useState<TenantMember[]>([]);
  const [pendingInvitations, setPendingInvitations] = useState<Invitation[]>(
    []
  );
  const [roles, setRoles] = useState<Role[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');

  // Invite member modal
  const [showAddModal, setShowAddModal] = useState(false);
  const [newMemberEmail, setNewMemberEmail] = useState('');
  const [newMemberRoleId, setNewMemberRoleId] = useState('');
  const [isAdding, setIsAdding] = useState(false);
  const [inviteError, setInviteError] = useState<string | null>(null);

  // Change role modal
  const [showRoleModal, setShowRoleModal] = useState(false);
  const [selectedMember, setSelectedMember] = useState<TenantMember | null>(
    null
  );
  const [selectedRoleId, setSelectedRoleId] = useState('');

  useEffect(() => {
    loadData();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentTenant]);

  const loadData = async () => {
    if (!currentTenant) return;

    setIsLoading(true);
    try {
      // Load members first (primary data)
      const memberList = await getMembers(currentTenant.id);
      setMembers(memberList);

      // Load pending invitations
      try {
        const invitationsResponse = await tenantApi.getPendingInvitations(
          currentTenant.id
        );
        setPendingInvitations(invitationsResponse.data || []);
      } catch (invErr) {
        console.error('Failed to load invitations:', invErr);
        setPendingInvitations([]);
      }

      // Load roles separately (for the add member dropdown)
      try {
        const roleList = await roleApi.getRoles().then((res) => res.data);
        setRoles(roleList);
      } catch (roleErr) {
        console.error('Failed to load roles:', roleErr);
        // Continue without roles - members are still shown
      }
    } catch (err) {
      console.error('Failed to load members:', err);
    } finally {
      setIsLoading(false);
    }
  };

  const handleInviteMember = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!currentTenant || !newMemberEmail.trim() || !newMemberRoleId) return;

    setIsAdding(true);
    setInviteError(null);
    try {
      await tenantApi.inviteMember(currentTenant.id, {
        email: newMemberEmail.trim(),
        role_id: newMemberRoleId,
      });
      setShowAddModal(false);
      setNewMemberEmail('');
      setNewMemberRoleId('');
      loadData();
    } catch (err) {
      console.error('Failed to invite member:', err);
      setInviteError(getErrorMessage(err, 'Failed to send invitation'));
    } finally {
      setIsAdding(false);
    }
  };

  const handleCancelInvitation = async (invitation: Invitation) => {
    if (!currentTenant) return;
    if (
      !confirm(
        `Are you sure you want to cancel the invitation for ${invitation.email}?`
      )
    )
      return;

    try {
      await tenantApi.cancelInvitation(currentTenant.id, invitation.id);
      loadData();
    } catch (err) {
      console.error('Failed to cancel invitation:', err);
    }
  };

  const handleResendInvitation = async (invitation: Invitation) => {
    if (!currentTenant) return;

    try {
      await tenantApi.resendInvitation(currentTenant.id, invitation.id);
      alert('Invitation resent successfully');
    } catch (err) {
      console.error('Failed to resend invitation:', err);
    }
  };

  const handleRemoveMember = async (member: TenantMember) => {
    if (!currentTenant) return;
    if (
      !confirm(
        `Are you sure you want to remove ${member.email} from this organization?`
      )
    )
      return;

    try {
      await removeMember(currentTenant.id, member.user_id);
      loadData();
    } catch (err) {
      console.error('Failed to remove member:', err);
    }
  };

  const handleChangeRole = async () => {
    if (!currentTenant || !selectedMember || !selectedRoleId) return;

    try {
      await updateMemberRole(
        currentTenant.id,
        selectedMember.user_id,
        selectedRoleId
      );
      setShowRoleModal(false);
      setSelectedMember(null);
      setSelectedRoleId('');
      loadData();
    } catch (err) {
      console.error('Failed to update role:', err);
    }
  };

  const openChangeRoleModal = (member: TenantMember) => {
    setSelectedMember(member);
    setSelectedRoleId(member.role_id);
    setShowRoleModal(true);
  };

  const filteredMembers = members.filter((member) => {
    const searchLower = searchQuery.toLowerCase();
    const displayName = getDisplayName(
      member.email,
      member.first_name,
      member.last_name
    );
    return (
      member.email.toLowerCase().includes(searchLower) ||
      member.role_name.toLowerCase().includes(searchLower) ||
      displayName.toLowerCase().includes(searchLower)
    );
  });

  const filteredInvitations = pendingInvitations.filter((invitation) => {
    const searchLower = searchQuery.toLowerCase();
    return (
      invitation.email.toLowerCase().includes(searchLower) ||
      invitation.role_name.toLowerCase().includes(searchLower)
    );
  });

  if (!currentTenant) {
    return (
      <DashboardLayout>
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          <Card variant="elevated" padding="lg">
            <CardContent>
              <div className="text-center py-8">
                <Building2 className="h-12 w-12 text-neutral-400 mx-auto mb-4" />
                <h2 className="text-lg font-medium text-neutral-900 mb-2">
                  No Organization Selected
                </h2>
                <p className="text-neutral-600">
                  Please select an organization to manage team members.
                </p>
              </div>
            </CardContent>
          </Card>
        </div>
      </DashboardLayout>
    );
  }

  return (
    <DashboardLayout>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        <div className="mb-8 flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold text-neutral-900">
              Team Members
            </h1>
            <p className="text-neutral-600 mt-2">
              Manage who has access to{' '}
              <span className="font-medium">{currentTenant.name}</span>
            </p>
          </div>
          <Button variant="primary" onClick={() => setShowAddModal(true)}>
            <UserPlus className="h-4 w-4 mr-2" />
            Invite Member
          </Button>
        </div>

        <Card variant="elevated" padding="lg">
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle>Members ({members.length})</CardTitle>
                <CardDescription>
                  People with access to this organization
                </CardDescription>
              </div>
              <div className="relative w-64">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-neutral-400" />
                <Input
                  type="text"
                  placeholder="Search members..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-10"
                  fullWidth
                />
              </div>
            </div>
          </CardHeader>
          <CardContent>
            {isLoading ? (
              <div className="py-8 text-center text-neutral-500">
                Loading members...
              </div>
            ) : filteredMembers.length === 0 &&
              filteredInvitations.length === 0 ? (
              <div className="py-8 text-center text-neutral-500">
                {searchQuery
                  ? 'No members match your search'
                  : 'No members found'}
              </div>
            ) : (
              <div className="space-y-6">
                {/* Active Members */}
                {filteredMembers.length > 0 && (
                  <div className="divide-y divide-neutral-200">
                    {filteredMembers.map((member) => {
                      const memberDisplayName = getDisplayName(
                        member.email,
                        member.first_name,
                        member.last_name
                      );
                      const hasName = member.first_name || member.last_name;

                      return (
                        <div
                          key={member.id}
                          className="flex items-center justify-between py-4"
                        >
                          <div className="flex items-center">
                            <Avatar
                              email={member.email}
                              firstName={member.first_name}
                              lastName={member.last_name}
                              profilePictureUrl={member.profile_picture_url}
                              size="md"
                            />
                            <div className="ml-4">
                              <p className="text-sm font-medium text-neutral-900">
                                {memberDisplayName}
                              </p>
                              {hasName && (
                                <p className="text-xs text-neutral-500">
                                  {member.email}
                                </p>
                              )}
                              <div className="flex items-center mt-1 space-x-2">
                                <Badge variant="neutral">
                                  {member.role_name}
                                </Badge>
                              </div>
                            </div>
                          </div>
                          <div className="flex items-center space-x-2">
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => openChangeRoleModal(member)}
                              title="Change role"
                            >
                              <Shield className="h-4 w-4" />
                            </Button>
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => handleRemoveMember(member)}
                              className="text-red-600 hover:text-red-700 hover:bg-red-50"
                              title="Remove member"
                            >
                              <Trash2 className="h-4 w-4" />
                            </Button>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}

                {/* Pending Invitations */}
                {filteredInvitations.length > 0 && (
                  <div>
                    <h3 className="text-sm font-medium text-neutral-700 mb-3 flex items-center">
                      <Clock className="h-4 w-4 mr-2" />
                      Pending Invitations ({filteredInvitations.length})
                    </h3>
                    <div className="divide-y divide-neutral-200 bg-neutral-50 rounded-lg">
                      {filteredInvitations.map((invitation) => (
                        <div
                          key={invitation.id}
                          className="flex items-center justify-between py-4 px-4"
                        >
                          <div className="flex items-center">
                            <div className="h-10 w-10 rounded-full bg-neutral-200 flex items-center justify-center">
                              <Mail className="h-5 w-5 text-neutral-500" />
                            </div>
                            <div className="ml-4">
                              <p className="text-sm font-medium text-neutral-900">
                                {invitation.email}
                              </p>
                              <div className="flex items-center mt-1 space-x-2">
                                <Badge variant="neutral">
                                  {invitation.role_name}
                                </Badge>
                                <Badge variant="warning">
                                  {invitation.user_registered
                                    ? 'Awaiting Acceptance'
                                    : 'Awaiting Registration'}
                                </Badge>
                              </div>
                              <p className="text-xs text-neutral-500 mt-1">
                                Invited by {invitation.invited_by_name} &bull;
                                Expires{' '}
                                {new Date(
                                  invitation.expires_at
                                ).toLocaleDateString()}
                              </p>
                            </div>
                          </div>
                          <div className="flex items-center space-x-2">
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => handleResendInvitation(invitation)}
                              title="Resend invitation"
                            >
                              <RefreshCw className="h-4 w-4" />
                            </Button>
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => handleCancelInvitation(invitation)}
                              className="text-red-600 hover:text-red-700 hover:bg-red-50"
                              title="Cancel invitation"
                            >
                              <X className="h-4 w-4" />
                            </Button>
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Invite Member Modal */}
      <Modal
        isOpen={showAddModal}
        onClose={() => {
          setShowAddModal(false);
          setNewMemberEmail('');
          setNewMemberRoleId('');
          setInviteError(null);
        }}
        title="Invite Team Member"
      >
        <form onSubmit={handleInviteMember} className="space-y-4">
          <p className="text-sm text-neutral-600">
            Send an invitation to join your organization. If they don't have an
            account, they'll be prompted to register.
          </p>

          {inviteError && (
            <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
              {inviteError}
            </div>
          )}

          <Input
            label="Email Address"
            type="email"
            value={newMemberEmail}
            onChange={(e) => setNewMemberEmail(e.target.value)}
            placeholder="member@company.com"
            required
            fullWidth
          />

          <div>
            <label className="block text-sm font-medium text-neutral-700 mb-1">
              Role
            </label>
            <select
              value={newMemberRoleId}
              onChange={(e) => setNewMemberRoleId(e.target.value)}
              className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500"
              required
            >
              <option value="">Select a role</option>
              {roles.map((role) => (
                <option key={role.id} value={role.id}>
                  {role.name} - {role.description}
                </option>
              ))}
            </select>
          </div>

          <div className="flex justify-end space-x-3 pt-4">
            <Button
              type="button"
              variant="outline"
              onClick={() => setShowAddModal(false)}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              variant="primary"
              disabled={isAdding || !newMemberRoleId}
            >
              {isAdding ? 'Sending...' : 'Send Invitation'}
            </Button>
          </div>
        </form>
      </Modal>

      {/* Change Role Modal */}
      <Modal
        isOpen={showRoleModal}
        onClose={() => {
          setShowRoleModal(false);
          setSelectedMember(null);
          setSelectedRoleId('');
        }}
        title="Change Member Role"
      >
        <div className="space-y-4">
          {selectedMember && (
            <p className="text-sm text-neutral-600">
              Change the role for{' '}
              <span className="font-medium">{selectedMember.email}</span>
            </p>
          )}

          <div>
            <label className="block text-sm font-medium text-neutral-700 mb-1">
              New Role
            </label>
            <select
              value={selectedRoleId}
              onChange={(e) => setSelectedRoleId(e.target.value)}
              className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-primary-500"
            >
              {roles.map((role) => (
                <option key={role.id} value={role.id}>
                  {role.name} - {role.description}
                </option>
              ))}
            </select>
          </div>

          <div className="flex justify-end space-x-3 pt-4">
            <Button
              type="button"
              variant="outline"
              onClick={() => setShowRoleModal(false)}
            >
              Cancel
            </Button>
            <Button type="button" variant="primary" onClick={handleChangeRole}>
              Update Role
            </Button>
          </div>
        </div>
      </Modal>
    </DashboardLayout>
  );
}
