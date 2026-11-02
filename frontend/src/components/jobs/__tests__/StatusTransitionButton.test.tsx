import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

vi.mock('@services/jobService', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@services/jobService')>();
  return { ...actual, jobService: { transitionStatus: vi.fn() } };
});
vi.mock('@services/analytics', () => ({
  trackEvent: vi.fn(),
  AnalyticsEvents: {
    JOB_COMPLETED: 'job_completed',
    JOB_STATUS_CHANGED: 'job_status_changed',
  },
}));

import { jobService, JobStatus, type Job } from '@services/jobService';
import { StatusTransitionButton } from '../StatusTransitionButton';

const job = (status: JobStatus, allowed?: JobStatus[]): Job =>
  ({
    id: 'job-1',
    job_number: 'JOB-001',
    customer_id: 'c-1',
    title: 'Fix boiler',
    job_type: 'repair',
    status,
    priority: 'normal',
    allowed_transitions: allowed,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-01T00:00:00Z',
  }) as unknown as Job;

// T267: the buttons come from the API's allowed_transitions (the backend
// job state machine), not from a copy of the rules in the UI.
describe('StatusTransitionButton', () => {
  beforeEach(() => vi.clearAllMocks());

  it('offers exactly the transitions the API allows', () => {
    render(
      <StatusTransitionButton
        job={job(JobStatus.ON_HOLD, [JobStatus.IN_PROGRESS])}
        onStatusChange={vi.fn()}
      />
    );
    expect(
      screen.getByRole('button', { name: /in progress/i })
    ).toBeInTheDocument();
    // Cancel is a valid move from on_hold in general, but this response
    // didn't allow it, so the UI doesn't offer it.
    expect(
      screen.queryByRole('button', { name: /cancel job/i })
    ).not.toBeInTheDocument();
  });

  it('offers no actions when the API allows none', () => {
    render(
      <StatusTransitionButton
        job={job(JobStatus.CANCELLED, [])}
        onStatusChange={vi.fn()}
      />
    );
    expect(screen.queryByRole('button')).not.toBeInTheDocument();
  });

  it('posts the chosen status to /transition and hands back the new job', async () => {
    const user = userEvent.setup({ delay: null });
    const updated = job(JobStatus.SCHEDULED, [JobStatus.IN_PROGRESS]);
    vi.mocked(jobService.transitionStatus).mockResolvedValue(updated);
    const onStatusChange = vi.fn();
    render(
      <StatusTransitionButton
        job={job(JobStatus.NEW, [JobStatus.SCHEDULED, JobStatus.CANCELLED])}
        onStatusChange={onStatusChange}
      />
    );

    await user.click(screen.getByRole('button', { name: /^schedule$/i }));

    await waitFor(() => expect(onStatusChange).toHaveBeenCalledWith(updated));
    expect(jobService.transitionStatus).toHaveBeenCalledWith(
      'job-1',
      JobStatus.SCHEDULED
    );
  });
});
