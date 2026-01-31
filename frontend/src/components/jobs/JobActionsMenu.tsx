import { useNavigate } from 'react-router-dom';
import { History } from 'lucide-react';
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

  return (
    <DropdownMenu buttonLabel="Job actions">
      {(close) => (
        <DropdownMenuItem
          onClick={(e) => handleViewHistory(e, close)}
          icon={<History className="h-4 w-4" />}
          label="View History"
        />
      )}
    </DropdownMenu>
  );
}
