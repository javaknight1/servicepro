import { useState } from 'react';
import { Button } from '@components/shared';
import { JobAssignment, jobService } from '@services/jobService';
import { UserPlus, X, Loader2, Users } from 'lucide-react';
import { AddAssignmentModal } from './AddAssignmentModal';

interface JobAssignmentsSectionProps {
  jobId: string;
  assignments: JobAssignment[];
  onAssignmentChange: () => void;
}

const roleLabels: Record<string, string> = {
  lead_technician: 'Lead Technician',
  technician: 'Technician',
  helper: 'Helper',
  inspector: 'Inspector',
};

export function JobAssignmentsSection({
  jobId,
  assignments,
  onAssignmentChange,
}: JobAssignmentsSectionProps) {
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [removingId, setRemovingId] = useState<string | null>(null);

  const handleRemove = async (userId: string) => {
    setRemovingId(userId);
    try {
      await jobService.unassignMember(jobId, userId);
      onAssignmentChange();
    } catch (err) {
      console.error('Failed to remove assignment:', err);
    } finally {
      setRemovingId(null);
    }
  };

  const handleAssignmentAdded = () => {
    setIsModalOpen(false);
    onAssignmentChange();
  };

  return (
    <div className="bg-white rounded-lg border border-neutral-200 p-6">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold text-neutral-900">
          Assigned Members
        </h2>
        <Button
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
              key={assignment.id}
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
                onClick={() => handleRemove(assignment.user_id)}
                disabled={removingId === assignment.user_id}
                className="p-2 text-neutral-400 hover:text-red-500 hover:bg-red-50 rounded-lg transition-colors disabled:opacity-50"
                title="Remove assignment"
              >
                {removingId === assignment.user_id ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <X className="h-4 w-4" />
                )}
              </button>
            </li>
          ))}
        </ul>
      )}

      <AddAssignmentModal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        jobId={jobId}
        existingAssignments={assignments}
        onAssignmentAdded={handleAssignmentAdded}
      />
    </div>
  );
}
