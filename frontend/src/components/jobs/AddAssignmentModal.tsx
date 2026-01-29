import { useState, useEffect, useCallback } from 'react';
import { Modal, Button } from '@components/shared';
import { JobAssignment, jobService } from '@services/jobService';
import { useTenantStore } from '@store/tenantStore';
import { TenantMember } from '@/types/tenant';
import { Loader2 } from 'lucide-react';

interface AddAssignmentModalProps {
  isOpen: boolean;
  onClose: () => void;
  jobId: string;
  existingAssignments: JobAssignment[];
  onAssignmentAdded: () => void;
}

const ROLES = [
  { value: 'lead_technician', label: 'Lead Technician' },
  { value: 'technician', label: 'Technician' },
  { value: 'helper', label: 'Helper' },
  { value: 'inspector', label: 'Inspector' },
];

export function AddAssignmentModal({
  isOpen,
  onClose,
  jobId,
  existingAssignments,
  onAssignmentAdded,
}: AddAssignmentModalProps) {
  const { currentTenant, getMembers } = useTenantStore();
  const [members, setMembers] = useState<TenantMember[]>([]);
  const [isLoadingMembers, setIsLoadingMembers] = useState(false);
  const [selectedMemberId, setSelectedMemberId] = useState('');
  const [selectedRole, setSelectedRole] = useState('technician');
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');

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
    if (isOpen && currentTenant) {
      loadMembers();
    }
  }, [isOpen, currentTenant, loadMembers]);

  useEffect(() => {
    if (!isOpen) {
      setSelectedMemberId('');
      setSelectedRole('technician');
      setSearchQuery('');
      setError(null);
    }
  }, [isOpen]);

  const assignedUserIds = new Set(existingAssignments.map((a) => a.user_id));

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

  const handleSave = async () => {
    if (!selectedMemberId) {
      setError('Please select a team member');
      return;
    }

    setIsSaving(true);
    setError(null);

    try {
      await jobService.assignMember(jobId, selectedMemberId, selectedRole);
      onAssignmentAdded();
    } catch (err) {
      console.error('Failed to assign member:', err);
      setError('Failed to assign member to job');
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Add Team Member" size="md">
      <div className="space-y-4">
        {error && (
          <div className="p-3 bg-red-50 border border-red-200 rounded-lg text-red-700 text-sm">
            {error}
          </div>
        )}

        <div>
          <label
            htmlFor="member-search"
            className="block text-sm font-medium text-neutral-700 mb-1"
          >
            Search Members
          </label>
          <input
            type="text"
            id="member-search"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search by name or email..."
            className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
          />
        </div>

        <div>
          <label
            htmlFor="member-select"
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
              id="member-select"
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
            htmlFor="role-select"
            className="block text-sm font-medium text-neutral-700 mb-1"
          >
            Role *
          </label>
          <select
            id="role-select"
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
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button
            variant="primary"
            onClick={handleSave}
            disabled={isSaving || !selectedMemberId}
            className="flex items-center gap-2"
          >
            {isSaving && <Loader2 className="h-4 w-4 animate-spin" />}
            Add Member
          </Button>
        </div>
      </div>
    </Modal>
  );
}
