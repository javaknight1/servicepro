import { useNavigate } from 'react-router-dom';
import { History, Copy } from 'lucide-react';
import { DropdownMenu, DropdownMenuItem } from '../shared/DropdownMenu';

interface JobActionsMenuProps {
  jobId: string;
}

export function JobActionsMenu({ jobId }: JobActionsMenuProps) {
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
          <DropdownMenuItem
            onClick={(e) => handleRepeatJob(e, close)}
            icon={<Copy className="h-4 w-4" />}
            label="Repeat Job"
          />
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
