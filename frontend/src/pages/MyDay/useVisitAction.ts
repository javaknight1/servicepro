import { useMutation, useQueryClient } from '@tanstack/react-query';
import { jobService, type Job } from '@services/jobService';
import type { VisitAction } from './myDay';
import { visitQueryKey } from './myDay';

/**
 * Runs a visit's big action (Start / Complete / Resume) through the job state
 * machine (POST /jobs/{id}/transition; the API stamps the actual start and
 * end times) and refreshes the day and the visit.
 */
export function useVisitAction() {
  const queryClient = useQueryClient();
  const mutation = useMutation({
    mutationFn: ({ job, action }: { job: Job; action: VisitAction }) =>
      jobService.transitionStatus(job.id, action.to),
    onSuccess: (updated) => {
      queryClient.invalidateQueries({ queryKey: ['my-day'] });
      queryClient.invalidateQueries({ queryKey: visitQueryKey(updated.id) });
    },
  });
  return {
    run: (job: Job, action: VisitAction) => mutation.mutate({ job, action }),
    /** The job whose action is in flight. */
    pendingJobId: mutation.isPending ? mutation.variables?.job.id : undefined,
    /** The job whose last action failed, and why. */
    failedJobId: mutation.isError ? mutation.variables?.job.id : undefined,
    error: mutation.error,
  };
}
