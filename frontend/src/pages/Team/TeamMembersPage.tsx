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
import {
  UserPlus,
  Trash2,
  Shield,
  Search,
  Building2,
} from 'lucide-react';
import type { TenantMember } from '@/types/tenant';
import type { Role } from '@/types/role';

export function TeamMembersPage() {
  const {
    currentTenant,
    getMembers,
    addMember,
    removeMember,
    updateMemberRole,
  } = useTenantStore();

  const [members, setMembers] = useState<TenantMember[]>([]);
  const [roles, setRoles] = useState<Role[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');

  // Add member modal
  const [showAddModal, setShowAddModal] = useState(false);
  const [newMemberEmail, setNewMemberEmail] = useState('');
  const [newMemberRoleId, setNewMemberRoleId] = useState('');
  const [isAdding, setIsAdding] = useState(false);

  // Change role modal
  const [showRoleModal, setShowRoleModal] = useState(false);
  const [selectedMember, setSelectedMember] = useState<TenantMember | null>(
    null
  );
  const [selectedRoleId, setSelectedRoleId] = useState('');

  useEffect(() => {
    loadData();
  }, [currentTenant]);

  const loadData = async () => {
    if (!currentTenant) return;

    setIsLoading(true);
    try {
      // Load members first (primary data)
      const memberList = await getMembers(currentTenant.id);
      setMembers(memberList);

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

  const handleAddMember = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!currentTenant || !newMemberEmail.trim()) return;

    setIsAdding(true);
    try {
      await addMember(currentTenant.id, {
        email: newMemberEmail.trim(),
        role_id: newMemberRoleId || '00000000-0000-0000-0000-000000000004',
      });
      setShowAddModal(false);
      setNewMemberEmail('');
      setNewMemberRoleId('');
      loadData();
    } catch (err) {
      console.error('Failed to add member:', err);
    } finally {
      setIsAdding(false);
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

  const filteredMembers = members.filter(
    (member) =>
      member.email.toLowerCase().includes(searchQuery.toLowerCase()) ||
      member.role_name.toLowerCase().includes(searchQuery.toLowerCase())
  );

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
            Add Member
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
            ) : filteredMembers.length === 0 ? (
              <div className="py-8 text-center text-neutral-500">
                {searchQuery
                  ? 'No members match your search'
                  : 'No members found'}
              </div>
            ) : (
              <div className="divide-y divide-neutral-200">
                {filteredMembers.map((member) => (
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
                          <Badge variant="neutral">{member.role_name}</Badge>
                          {!member.accepted_at && member.invited_at && (
                            <Badge variant="warning">Pending Invite</Badge>
                          )}
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
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </div>

      {/* Add Member Modal */}
      <Modal
        isOpen={showAddModal}
        onClose={() => {
          setShowAddModal(false);
          setNewMemberEmail('');
          setNewMemberRoleId('');
        }}
        title="Add Team Member"
      >
        <form onSubmit={handleAddMember} className="space-y-4">
          <p className="text-sm text-neutral-600">
            Invite a user to join your organization. They must already have an
            account.
          </p>

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
            <Button type="submit" variant="primary" disabled={isAdding}>
              {isAdding ? 'Adding...' : 'Add Member'}
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
