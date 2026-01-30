import { useState } from 'react';
import { Button } from '@components/shared';
import { jobService, JobStatus, Job } from '@services/jobService';
import { getJobStatusLabel } from '@app-types';
import {
  Loader2,
  ArrowRight,
  Pause,
  Play,
  Truck,
  Undo2,
  XCircle,
  CheckCircle,
  FileText,
  DollarSign,
  Calendar,
} from 'lucide-react';

interface StatusTransitionButtonProps {
  job: Job;
  onStatusChange: (updatedJob: Job) => void;
  disabled?: boolean;
}

// Define allowed transitions from each status
const transitionRules: Record<JobStatus, JobStatus[]> = {
  [JobStatus.NEW]: [JobStatus.SCHEDULED, JobStatus.CANCELLED],
  [JobStatus.SCHEDULED]: [
    JobStatus.IN_PROGRESS,
    JobStatus.EN_ROUTE,
    JobStatus.NEW,
    JobStatus.CANCELLED,
  ],
  [JobStatus.EN_ROUTE]: [
    JobStatus.IN_PROGRESS,
    JobStatus.SCHEDULED,
    JobStatus.CANCELLED,
  ],
  [JobStatus.IN_PROGRESS]: [
    JobStatus.COMPLETED,
    JobStatus.ON_HOLD,
    JobStatus.EN_ROUTE,
    JobStatus.SCHEDULED,
    JobStatus.CANCELLED,
  ],
  [JobStatus.ON_HOLD]: [JobStatus.IN_PROGRESS, JobStatus.CANCELLED],
  [JobStatus.COMPLETED]: [
    JobStatus.INVOICED,
    JobStatus.IN_PROGRESS,
    JobStatus.CANCELLED,
  ],
  [JobStatus.INVOICED]: [JobStatus.PAID, JobStatus.COMPLETED],
  [JobStatus.PAID]: [JobStatus.INVOICED],
  [JobStatus.CANCELLED]: [],
};

// Get button configuration for each status
function getButtonConfig(toStatus: JobStatus, fromStatus: JobStatus) {
  const isBackward = isBackwardTransition(fromStatus, toStatus);

  const configs: Record<
    JobStatus,
    {
      label: string;
      icon: typeof ArrowRight;
      variant: 'primary' | 'secondary' | 'danger';
    }
  > = {
    [JobStatus.NEW]: {
      label: 'Revert to New',
      icon: Undo2,
      variant: 'secondary',
    },
    [JobStatus.SCHEDULED]: {
      label: isBackward ? 'Revert to Scheduled' : 'Schedule',
      icon: isBackward ? Undo2 : Calendar,
      variant: isBackward ? 'secondary' : 'primary',
    },
    [JobStatus.EN_ROUTE]: {
      label: isBackward ? 'Back to En Route' : 'Set En Route',
      icon: isBackward ? Undo2 : Truck,
      variant: 'secondary',
    },
    [JobStatus.IN_PROGRESS]: {
      label: isBackward ? 'Revert to In Progress' : 'Start Job',
      icon: isBackward ? Undo2 : Play,
      variant: isBackward ? 'secondary' : 'primary',
    },
    [JobStatus.ON_HOLD]: {
      label: 'Put On Hold',
      icon: Pause,
      variant: 'secondary',
    },
    [JobStatus.COMPLETED]: {
      label: isBackward ? 'Revert to Completed' : 'Complete Job',
      icon: isBackward ? Undo2 : CheckCircle,
      variant: isBackward ? 'secondary' : 'primary',
    },
    [JobStatus.INVOICED]: {
      label: isBackward ? 'Revert to Invoiced' : 'Mark Invoiced',
      icon: isBackward ? Undo2 : FileText,
      variant: isBackward ? 'secondary' : 'primary',
    },
    [JobStatus.PAID]: {
      label: 'Mark Paid',
      icon: DollarSign,
      variant: 'primary',
    },
    [JobStatus.CANCELLED]: {
      label: 'Cancel Job',
      icon: XCircle,
      variant: 'danger',
    },
  };

  return configs[toStatus];
}

// Check if this is a backward transition
function isBackwardTransition(from: JobStatus, to: JobStatus): boolean {
  const statusOrder = [
    JobStatus.NEW,
    JobStatus.SCHEDULED,
    JobStatus.EN_ROUTE,
    JobStatus.IN_PROGRESS,
    JobStatus.ON_HOLD,
    JobStatus.COMPLETED,
    JobStatus.INVOICED,
    JobStatus.PAID,
  ];

  const fromIndex = statusOrder.indexOf(from);
  const toIndex = statusOrder.indexOf(to);

  // Special cases
  if (to === JobStatus.ON_HOLD) return false; // On hold is not backward
  if (to === JobStatus.CANCELLED) return false; // Cancel is not backward

  return toIndex < fromIndex;
}

// Categorize transitions
function categorizeTransitions(currentStatus: JobStatus) {
  const allowed = transitionRules[currentStatus] || [];

  const primary: JobStatus[] = [];
  const secondary: JobStatus[] = [];
  const backward: JobStatus[] = [];
  let cancel: JobStatus | null = null;

  for (const status of allowed) {
    if (status === JobStatus.CANCELLED) {
      cancel = status;
    } else if (isBackwardTransition(currentStatus, status)) {
      backward.push(status);
    } else if (
      status === currentStatus ||
      status === JobStatus.ON_HOLD ||
      status === JobStatus.EN_ROUTE
    ) {
      secondary.push(status);
    } else {
      primary.push(status);
    }
  }

  return { primary, secondary, backward, cancel };
}

export function StatusTransitionButton({
  job,
  onStatusChange,
  disabled,
}: StatusTransitionButtonProps) {
  const [isTransitioning, setIsTransitioning] = useState<JobStatus | null>(
    null
  );
  const [error, setError] = useState<string | null>(null);

  const { primary, secondary, backward, cancel } = categorizeTransitions(
    job.status
  );

  const handleTransition = async (toStatus: JobStatus) => {
    if (toStatus === JobStatus.CANCELLED) {
      const reason = window.prompt('Please provide a reason for cancellation:');
      if (!reason) return;

      setIsTransitioning(toStatus);
      setError(null);

      try {
        const updatedJob = await jobService.transitionStatus(
          job.id,
          toStatus,
          'cancellation',
          reason
        );
        onStatusChange(updatedJob);
      } catch (err) {
        console.error('Failed to transition status:', err);
        setError('Failed to cancel job');
      } finally {
        setIsTransitioning(null);
      }
      return;
    }

    if (toStatus === JobStatus.ON_HOLD) {
      const reason = window.prompt(
        'Please provide a reason for putting on hold:'
      );
      if (!reason) return;

      setIsTransitioning(toStatus);
      setError(null);

      try {
        const updatedJob = await jobService.transitionStatus(
          job.id,
          toStatus,
          'technical_issue',
          reason
        );
        onStatusChange(updatedJob);
      } catch (err) {
        console.error('Failed to transition status:', err);
        setError('Failed to put job on hold');
      } finally {
        setIsTransitioning(null);
      }
      return;
    }

    setIsTransitioning(toStatus);
    setError(null);

    try {
      const updatedJob = await jobService.transitionStatus(job.id, toStatus);
      onStatusChange(updatedJob);
    } catch (err) {
      console.error('Failed to transition status:', err);
      setError('Failed to update status');
    } finally {
      setIsTransitioning(null);
    }
  };

  const allTransitions = [...primary, ...secondary, ...backward];
  if (allTransitions.length === 0 && !cancel) {
    return (
      <div className="text-sm text-neutral-500">
        Status: {getJobStatusLabel(job.status)} (no transitions available)
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-center gap-2">
        {/* Primary actions */}
        {primary.map((status) => {
          const config = getButtonConfig(status, job.status);
          const Icon = config.icon;
          return (
            <Button
              key={status}
              variant={config.variant}
              onClick={() => handleTransition(status)}
              disabled={disabled || isTransitioning !== null}
              className="flex items-center gap-2"
            >
              {isTransitioning === status ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Icon className="h-4 w-4" />
              )}
              {config.label}
            </Button>
          );
        })}

        {/* Secondary actions */}
        {secondary.map((status) => {
          const config = getButtonConfig(status, job.status);
          const Icon = config.icon;
          return (
            <Button
              key={status}
              variant="secondary"
              onClick={() => handleTransition(status)}
              disabled={disabled || isTransitioning !== null}
              className="flex items-center gap-2"
            >
              {isTransitioning === status ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Icon className="h-4 w-4" />
              )}
              {config.label}
            </Button>
          );
        })}

        {/* Backward transitions */}
        {backward.map((status) => {
          const config = getButtonConfig(status, job.status);
          const Icon = config.icon;
          return (
            <Button
              key={status}
              variant="secondary"
              onClick={() => handleTransition(status)}
              disabled={disabled || isTransitioning !== null}
              className="flex items-center gap-2"
            >
              {isTransitioning === status ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Icon className="h-4 w-4" />
              )}
              {config.label}
            </Button>
          );
        })}

        {/* Cancel button */}
        {cancel && (
          <Button
            variant="danger"
            onClick={() => handleTransition(cancel)}
            disabled={disabled || isTransitioning !== null}
            className="flex items-center gap-2"
          >
            {isTransitioning === cancel ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <XCircle className="h-4 w-4" />
            )}
            Cancel Job
          </Button>
        )}
      </div>

      {error && <p className="text-xs text-red-600">{error}</p>}
    </div>
  );
}
