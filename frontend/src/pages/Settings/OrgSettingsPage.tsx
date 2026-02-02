import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
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
import { tenantApi } from '@services/tenantService';
import { roleApi } from '@services/roleApi';
import { getErrorMessage } from '@/utils/error';
import {
  Building2,
  Users,
  Settings,
  Trash2,
  UserPlus,
  CreditCard,
  Mail,
  RefreshCw,
  X,
  Clock,
} from 'lucide-react';
import { MembershipTab } from '@components/membership';
import type {
  TenantMember,
  UpdateTenantRequest,
  Invitation,
} from '@/types/tenant';
import type { Role } from '@/types/role';

type TabId = 'general' | 'members' | 'membership';

export function OrgSettingsPage() {
  const navigate = useNavigate();
  const {
    currentTenant,
    isLoading,
    error,
    updateTenant,
    deleteTenant,
    getMembers,
    removeMember,
    clearError,
  } = useTenantStore();

  const [activeTab, setActiveTab] = useState<TabId>('general');
  const [members, setMembers] = useState<TenantMember[]>([]);
  const [pendingInvitations, setPendingInvitations] = useState<Invitation[]>(
    []
  );
  const [roles, setRoles] = useState<Role[]>([]);
  const [membersLoading, setMembersLoading] = useState(false);

  // Form state for general settings
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [phone, setPhone] = useState('');
  const [isSaving, setIsSaving] = useState(false);

  // Modal state for inviting member
  const [showAddMemberModal, setShowAddMemberModal] = useState(false);
  const [newMemberEmail, setNewMemberEmail] = useState('');
  const [newMemberRoleId, setNewMemberRoleId] = useState('');
  const [isInviting, setIsInviting] = useState(false);
  const [inviteError, setInviteError] = useState<string | null>(null);

  // Modal state for delete confirmation
  const [showDeleteModal, setShowDeleteModal] = useState(false);
  const [deleteConfirmText, setDeleteConfirmText] = useState('');

  const tabs = [
    { id: 'general' as const, label: 'General', icon: Settings },
    { id: 'members' as const, label: 'Members', icon: Users },
    { id: 'membership' as const, label: 'Membership', icon: CreditCard },
  ];

  // Load current tenant data into form
  useEffect(() => {
    if (currentTenant) {
      setName(currentTenant.name);
      setEmail(currentTenant.email || '');
      setPhone(currentTenant.phone || '');
    }
  }, [currentTenant]);

  // Load members when members tab is active
  useEffect(() => {
    if (activeTab === 'members' && currentTenant) {
      loadMembers();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activeTab, currentTenant]);

  const loadMembers = async () => {
    if (!currentTenant) return;
    setMembersLoading(true);
    try {
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

      // Load roles for the invite modal
      try {
        const roleList = await roleApi.getRoles().then((res) => res.data);
        setRoles(roleList);
      } catch (roleErr) {
        console.error('Failed to load roles:', roleErr);
      }
    } catch (err) {
      console.error('Failed to load members:', err);
    } finally {
      setMembersLoading(false);
    }
  };

  const handleSaveGeneral = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!currentTenant) return;

    setIsSaving(true);
    clearError();

    try {
      const data: UpdateTenantRequest = {
        name: name.trim(),
        email: email.trim() || undefined,
        phone: phone.trim() || undefined,
      };
      await updateTenant(currentTenant.id, data);
    } catch (err) {
      console.error('Failed to save:', err);
    } finally {
      setIsSaving(false);
    }
  };

  const handleInviteMember = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!currentTenant || !newMemberEmail.trim() || !newMemberRoleId) return;

    setIsInviting(true);
    setInviteError(null);
    try {
      await tenantApi.inviteMember(currentTenant.id, {
        email: newMemberEmail.trim(),
        role_id: newMemberRoleId,
      });
      setShowAddMemberModal(false);
      setNewMemberEmail('');
      setNewMemberRoleId('');
      loadMembers();
    } catch (err) {
      console.error('Failed to invite member:', err);
      setInviteError(getErrorMessage(err, 'Failed to send invitation'));
    } finally {
      setIsInviting(false);
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
      loadMembers();
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

  const handleRemoveMember = async (userId: string) => {
    if (!currentTenant) return;
    if (!confirm('Are you sure you want to remove this member?')) return;

    try {
      await removeMember(currentTenant.id, userId);
      loadMembers();
    } catch (err) {
      console.error('Failed to remove member:', err);
    }
  };

  const handleDeleteOrganization = async () => {
    if (!currentTenant) return;
    if (deleteConfirmText !== currentTenant.name) return;

    try {
      await deleteTenant(currentTenant.id);
      navigate('/dashboard');
    } catch (err) {
      console.error('Failed to delete organization:', err);
    }
  };

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
                <p className="text-neutral-600 mb-4">
                  Please select or create an organization to manage settings.
                </p>
                <Button
                  variant="primary"
                  onClick={() => navigate('/settings/organization/new')}
                >
                  Create Organization
                </Button>
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
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-neutral-900">
            Organization Settings
          </h1>
          <p className="text-neutral-600 mt-2">
            Manage settings for{' '}
            <span className="font-medium">{currentTenant.name}</span>
          </p>
        </div>

        {error && (
          <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
            {error}
          </div>
        )}

        <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
          {/* Sidebar */}
          <div className="lg:col-span-1">
            <Card variant="elevated" padding="sm">
              <nav className="space-y-1">
                {tabs.map((tab) => {
                  const Icon = tab.icon;
                  return (
                    <button
                      key={tab.id}
                      onClick={() => setActiveTab(tab.id)}
                      className={`w-full flex items-center px-4 py-3 text-sm font-medium rounded-lg transition-colors ${
                        activeTab === tab.id
                          ? 'bg-primary-50 text-primary-700'
                          : 'text-neutral-700 hover:bg-neutral-50'
                      }`}
                    >
                      <Icon className="h-5 w-5 mr-3" />
                      {tab.label}
                    </button>
                  );
                })}
              </nav>
            </Card>
          </div>

          {/* Content */}
          <div className="lg:col-span-3 space-y-6">
            {activeTab === 'general' && (
              <>
                <Card variant="elevated" padding="lg">
                  <CardHeader>
                    <CardTitle>Organization Information</CardTitle>
                    <CardDescription>
                      Update your organization's basic information
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    <form
                      onSubmit={handleSaveGeneral}
                      className="space-y-4 mt-6"
                    >
                      <Input
                        label="Organization Name"
                        type="text"
                        value={name}
                        onChange={(e) => setName(e.target.value)}
                        required
                        fullWidth
                      />

                      <Input
                        label="Email Address"
                        type="email"
                        value={email}
                        onChange={(e) => setEmail(e.target.value)}
                        placeholder="contact@company.com"
                        fullWidth
                      />

                      <Input
                        label="Phone Number"
                        type="tel"
                        value={phone}
                        onChange={(e) => setPhone(e.target.value)}
                        placeholder="+1 (555) 123-4567"
                        fullWidth
                      />

                      <div className="flex justify-end space-x-3 pt-4">
                        <Button
                          type="button"
                          variant="outline"
                          onClick={() => {
                            setName(currentTenant.name);
                            setEmail(currentTenant.email || '');
                            setPhone(currentTenant.phone || '');
                          }}
                        >
                          Reset
                        </Button>
                        <Button
                          type="submit"
                          variant="primary"
                          disabled={isSaving || isLoading}
                        >
                          {isSaving ? 'Saving...' : 'Save Changes'}
                        </Button>
                      </div>
                    </form>
                  </CardContent>
                </Card>

                {/* Danger Zone */}
                <Card
                  variant="elevated"
                  padding="lg"
                  className="border-red-200"
                >
                  <CardHeader>
                    <CardTitle className="text-red-700">Danger Zone</CardTitle>
                    <CardDescription>
                      Irreversible and destructive actions
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    <div className="flex items-center justify-between py-4">
                      <div>
                        <p className="text-sm font-medium text-neutral-900">
                          Delete Organization
                        </p>
                        <p className="text-sm text-neutral-600">
                          Permanently delete this organization and all its data
                        </p>
                      </div>
                      <Button
                        variant="outline"
                        className="text-red-600 border-red-300 hover:bg-red-50"
                        onClick={() => setShowDeleteModal(true)}
                      >
                        <Trash2 className="h-4 w-4 mr-2" />
                        Delete
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              </>
            )}

            {activeTab === 'members' && (
              <Card variant="elevated" padding="lg">
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <div>
                      <CardTitle>Team Members</CardTitle>
                      <CardDescription>
                        Manage who has access to this organization
                      </CardDescription>
                    </div>
                    <Button
                      variant="primary"
                      size="sm"
                      onClick={() => setShowAddMemberModal(true)}
                    >
                      <UserPlus className="h-4 w-4 mr-2" />
                      Invite Member
                    </Button>
                  </div>
                </CardHeader>
                <CardContent>
                  {membersLoading ? (
                    <div className="py-8 text-center text-neutral-500">
                      Loading members...
                    </div>
                  ) : members.length === 0 &&
                    pendingInvitations.length === 0 ? (
                    <div className="py-8 text-center text-neutral-500">
                      No members found
                    </div>
                  ) : (
                    <div className="space-y-6 mt-4">
                      {/* Active Members */}
                      {members.length > 0 && (
                        <div className="divide-y divide-neutral-200">
                          {members.map((member) => (
                            <div
                              key={member.id}
                              className="flex items-center justify-between py-4"
                            >
                              <div className="flex items-center">
                                <Avatar
                                  email={member.email}
                                  profilePictureUrl={member.profile_picture_url}
                                  size="md"
                                />
                                <div className="ml-4">
                                  <p className="text-sm font-medium text-neutral-900">
                                    {member.email}
                                  </p>
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
                                  onClick={() =>
                                    handleRemoveMember(member.user_id)
                                  }
                                  className="text-red-600 hover:text-red-700 hover:bg-red-50"
                                >
                                  <Trash2 className="h-4 w-4" />
                                </Button>
                              </div>
                            </div>
                          ))}
                        </div>
                      )}

                      {/* Pending Invitations */}
                      {pendingInvitations.length > 0 && (
                        <div>
                          <h3 className="text-sm font-medium text-neutral-700 mb-3 flex items-center">
                            <Clock className="h-4 w-4 mr-2" />
                            Pending Invitations ({pendingInvitations.length})
                          </h3>
                          <div className="divide-y divide-neutral-200 bg-neutral-50 rounded-lg">
                            {pendingInvitations.map((invitation) => (
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
                                    onClick={() =>
                                      handleResendInvitation(invitation)
                                    }
                                    title="Resend invitation"
                                  >
                                    <RefreshCw className="h-4 w-4" />
                                  </Button>
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    onClick={() =>
                                      handleCancelInvitation(invitation)
                                    }
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
            )}

            {activeTab === 'membership' && <MembershipTab />}
          </div>
        </div>
      </div>

      {/* Invite Member Modal */}
      <Modal
        isOpen={showAddMemberModal}
        onClose={() => {
          setShowAddMemberModal(false);
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
              onClick={() => setShowAddMemberModal(false)}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              variant="primary"
              disabled={isInviting || !newMemberRoleId}
            >
              {isInviting ? 'Sending...' : 'Send Invitation'}
            </Button>
          </div>
        </form>
      </Modal>

      {/* Delete Confirmation Modal */}
      <Modal
        isOpen={showDeleteModal}
        onClose={() => {
          setShowDeleteModal(false);
          setDeleteConfirmText('');
        }}
        title="Delete Organization"
      >
        <div className="space-y-4">
          <div className="p-4 bg-red-50 border border-red-200 rounded-lg">
            <p className="text-sm text-red-700">
              This action is irreversible. All data including customers, jobs,
              quotes, and invoices will be permanently deleted.
            </p>
          </div>

          <p className="text-sm text-neutral-600">
            Type{' '}
            <span className="font-mono font-bold">{currentTenant.name}</span> to
            confirm:
          </p>

          <Input
            type="text"
            value={deleteConfirmText}
            onChange={(e) => setDeleteConfirmText(e.target.value)}
            placeholder={currentTenant.name}
            fullWidth
          />

          <div className="flex justify-end space-x-3 pt-4">
            <Button
              type="button"
              variant="outline"
              onClick={() => {
                setShowDeleteModal(false);
                setDeleteConfirmText('');
              }}
            >
              Cancel
            </Button>
            <Button
              type="button"
              variant="primary"
              className="bg-red-600 hover:bg-red-700"
              disabled={deleteConfirmText !== currentTenant.name}
              onClick={handleDeleteOrganization}
            >
              Delete Organization
            </Button>
          </div>
        </div>
      </Modal>
    </DashboardLayout>
  );
}
