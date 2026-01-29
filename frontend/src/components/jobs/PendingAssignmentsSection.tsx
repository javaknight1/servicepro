import { useState, useEffect, useCallback } from 'react';
import { Button, Modal } from '@components/shared';
import { CreateJobAssignment } from '@services/jobService';
import { useTenantStore } from '@store/tenantStore';
import { TenantMember } from '@/types/tenant';
import { UserPlus, X, Loader2, Users } from 'lucide-react';

interface PendingAssignment extends CreateJobAssignment {
  user_name: string;
}

interface PendingAssignmentsSectionProps {
  assignments: PendingAssignment[];
  onAssignmentsChange: (assignments: PendingAssignment[]) => void;
}

const roleLabels: Record<string, string> = {
  lead_technician: 'Lead Technician',
  technician: 'Technician',
  helper: 'Helper',
  inspector: 'Inspector',
};

const ROLES = [
  { value: 'lead_technician', label: 'Lead Technician' },
  { value: 'technician', label: 'Technician' },
  { value: 'helper', label: 'Helper' },
  { value: 'inspector', label: 'Inspector' },
];

export function PendingAssignmentsSection({
  assignments,
  onAssignmentsChange,
}: PendingAssignmentsSectionProps) {
  const { currentTenant, getMembers } = useTenantStore();
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [members, setMembers] = useState<TenantMember[]>([]);
  const [isLoadingMembers, setIsLoadingMembers] = useState(false);
  const [selectedMemberId, setSelectedMemberId] = useState('');
  const [selectedRole, setSelectedRole] = useState('technician');
  const [searchQuery, setSearchQuery] = useState('');
  const [error, setError] = useState<string | null>(null);

  const loadMembers = useCallback(async () => {
    if (!currentTenant) return;

    setIsLoadingMembers(true);
    try {
      const fetchedMembers = await getMembers(currentTenant.id);
      setMembers(fetchedMembers);
    } catch (err) {
      console.error('Failed to load members:', err);
      setError('Failed to load team members');
    } finally {
      setIsLoadingMembers(false);
    }
  }, [currentTenant, getMembers]);

  useEffect(() => {
    if (isModalOpen && currentTenant) {
      loadMembers();
    }
  }, [isModalOpen, currentTenant, loadMembers]);

  useEffect(() => {
    if (!isModalOpen) {
      setSelectedMemberId('');
      setSelectedRole('technician');
      setSearchQuery('');
      setError(null);
    }
  }, [isModalOpen]);

  const handleRemove = (userId: string) => {
    onAssignmentsChange(assignments.filter((a) => a.user_id !== userId));
  };

  const assignedUserIds = new Set(assignments.map((a) => a.user_id));

  const availableMembers = members.filter(
    (member) =>
      !assignedUserIds.has(member.user_id) &&
      member.is_active &&
      (searchQuery === '' ||
        `${member.first_name} ${member.last_name}`
          .toLowerCase()
          .includes(searchQuery.toLowerCase()) ||
        member.email.toLowerCase().includes(searchQuery.toLowerCase()))
  );

  const handleAddMember = () => {
    if (!selectedMemberId) {
      setError('Please select a team member');
      return;
    }

    const member = members.find((m) => m.user_id === selectedMemberId);
    if (!member) return;

    const newAssignment: PendingAssignment = {
      user_id: selectedMemberId,
      role: selectedRole,
      user_name:
        `${member.first_name || ''} ${member.last_name || ''}`.trim() ||
        member.email,
    };

    onAssignmentsChange([...assignments, newAssignment]);
    setIsModalOpen(false);
  };

  return (
    <div className="bg-white rounded-lg border border-neutral-200 p-6">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold text-neutral-900">
          Assigned Members
        </h2>
        <Button
          type="button"
          variant="secondary"
          size="sm"
          onClick={() => setIsModalOpen(true)}
          className="flex items-center gap-2"
        >
          <UserPlus className="h-4 w-4" />
          Add Member
        </Button>
      </div>

      {assignments.length === 0 ? (
        <div className="text-center py-8 text-neutral-500">
          <Users className="h-12 w-12 mx-auto mb-3 text-neutral-300" />
          <p>No team members assigned to this job yet.</p>
          <p className="text-sm mt-1">
            Click "Add Member" to assign someone to this job.
          </p>
        </div>
      ) : (
        <ul className="divide-y divide-neutral-200">
          {assignments.map((assignment) => (
            <li
              key={assignment.user_id}
              className="flex items-center justify-between py-3"
            >
              <div>
                <p className="font-medium text-neutral-900">
                  {assignment.user_name}
                </p>
                <p className="text-sm text-neutral-500">
                  {roleLabels[assignment.role] || assignment.role}
                </p>
              </div>
              <button
                type="button"
                onClick={() => handleRemove(assignment.user_id)}
                className="p-2 text-neutral-400 hover:text-red-500 hover:bg-red-50 rounded-lg transition-colors"
                title="Remove assignment"
              >
                <X className="h-4 w-4" />
              </button>
            </li>
          ))}
        </ul>
      )}

      <Modal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        title="Add Team Member"
        size="md"
      >
        <div className="space-y-4">
          {error && (
            <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
              {error}
            </div>
          )}

          <div>
            <label
              htmlFor="pending-member-search"
              className="block text-sm font-medium text-neutral-700 mb-1"
            >
              Search Members
            </label>
            <input
              type="text"
              id="pending-member-search"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search by name or email..."
              className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            />
          </div>

          <div>
            <label
              htmlFor="pending-member-select"
              className="block text-sm font-medium text-neutral-700 mb-1"
            >
              Team Member *
            </label>
            {isLoadingMembers ? (
              <div className="flex items-center justify-center py-4">
                <Loader2 className="h-5 w-5 animate-spin text-primary-500" />
              </div>
            ) : (
              <select
                id="pending-member-select"
                value={selectedMemberId}
                onChange={(e) => setSelectedMemberId(e.target.value)}
                className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
              >
                <option value="">Select a team member</option>
                {availableMembers.map((member) => (
                  <option key={member.user_id} value={member.user_id}>
                    {member.first_name} {member.last_name} ({member.email})
                  </option>
                ))}
              </select>
            )}
            {!isLoadingMembers && availableMembers.length === 0 && (
              <p className="mt-2 text-sm text-neutral-500">
                {members.length === 0
                  ? 'No team members found'
                  : 'All team members are already assigned to this job'}
              </p>
            )}
          </div>

          <div>
            <label
              htmlFor="pending-role-select"
              className="block text-sm font-medium text-neutral-700 mb-1"
            >
              Role *
            </label>
            <select
              id="pending-role-select"
              value={selectedRole}
              onChange={(e) => setSelectedRole(e.target.value)}
              className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            >
              {ROLES.map((role) => (
                <option key={role.value} value={role.value}>
                  {role.label}
                </option>
              ))}
            </select>
          </div>

          <div className="flex justify-end gap-3 pt-4">
            <Button
              type="button"
              variant="secondary"
              onClick={() => setIsModalOpen(false)}
            >
              Cancel
            </Button>
            <Button
              type="button"
              variant="primary"
              onClick={handleAddMember}
              disabled={!selectedMemberId}
              className="flex items-center gap-2"
            >
              Add Member
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
