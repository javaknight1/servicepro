import { useNavigate } from 'react-router';
import { History, Copy } from 'lucide-react';
import { DropdownMenu, DropdownMenuItem } from '../shared/DropdownMenu';

interface JobActionsMenuProps {
  jobId: string;
  /** Offer "Repeat Job" (needs jobs.create; technicians can't, T035). */
  canRepeat?: boolean;
}

export function JobActionsMenu({
  jobId,
  canRepeat = true,
}: JobActionsMenuProps) {
  const navigate = useNavigate();

  const handleViewHistory = (e: React.MouseEvent, close: () => void) => {
    e.stopPropagation();
    close();
    navigate(`/jobs/${jobId}/history`);
  };

  const handleRepeatJob = (e: React.MouseEvent, close: () => void) => {
    e.stopPropagation();
    close();
    navigate(`/jobs/new?clone=${jobId}`);
  };

  return (
    <DropdownMenu buttonLabel="Job actions">
      {(close) => (
        <>
          {canRepeat && (
            <DropdownMenuItem
              onClick={(e) => handleRepeatJob(e, close)}
              icon={<Copy className="h-4 w-4" />}
              label="Repeat Job"
            />
          )}
          <DropdownMenuItem
            onClick={(e) => handleViewHistory(e, close)}
            icon={<History className="h-4 w-4" />}
            label="View History"
          />
        </>
      )}
    </DropdownMenu>
  );
}
