import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AxiosError, type AxiosResponse } from 'axios';
import type { ReactNode } from 'react';
import type { Tenant } from '@/types/tenant';
import type { Job } from '@services/jobService';

vi.mock('@components/layout', () => ({
  DashboardLayout: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
}));

const tenant: Tenant = {
  id: 'tenant-1',
  name: 'Acme Cleaning',
  slug: 'acme',
  owner_id: 'owner-1',
  is_active: true,
  role: 'technician',
  settings: { timezone: 'America/Chicago' },
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
};

vi.mock('@store', () => ({
  useTenantStore: (select: (s: { currentTenant: Tenant }) => unknown) =>
    select({ currentTenant: tenant }),
}));

vi.mock('@services/jobService', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@services/jobService')>()),
  jobService: {
    getJob: vi.fn(),
    transitionStatus: vi.fn(),
    addNote: vi.fn(),
    addMaterial: vi.fn(),
  },
}));

import { jobService, JobStatus, JobType } from '@services/jobService';
import { VisitPage } from '../VisitPage';

// 2026-11-03 17:32 CST
const STARTED = Date.parse('2026-11-03T17:32:00-06:00') / 1000;

const job: Job = {
  id: 'job-1',
  job_number: 'J-3318',
  customer_id: 'cust-1',
  customer: {
    id: 'cust-1',
    email: 'ap@harborview.test',
    first_name: 'Ray',
    last_name: 'Burke',
    company_name: 'Harborview Property Management',
  },
  site_id: 'site-1',
  site: {
    id: 'site-1',
    name: 'Elm Street Plaza',
    address: {
      street: '410 Elm St',
      city: 'Springfield',
      state: 'IL',
      zip: '62701',
    },
    access_notes: 'Key code 4471',
    contact_name: 'Ray Burke',
    contact_phone: '555-0100',
  },
  title: 'Nightly cleaning',
  description: 'Vacuum lobby and entry floors.\nEmpty trash, all floors.',
  job_type: JobType.MAINTENANCE,
  status: JobStatus.IN_PROGRESS,
  priority: 'normal' as Job['priority'],
  actual_start_at: STARTED,
  allowed_transitions: [
    JobStatus.ON_HOLD,
    JobStatus.COMPLETED,
    JobStatus.CANCELLED,
  ],
  assignments: [
    {
      id: 'a-1',
      job_id: 'job-1',
      user_id: 'tech-1',
      user_name: 'Tasha Green',
      role: 'technician',
      assigned_at: 0,
      hours_worked: 0,
    },
  ],
  materials: [
    {
      id: 'm-1',
      job_id: 'job-1',
      name: 'Trash liners',
      quantity: 12,
      unit: 'each',
      unit_cost: 0,
      total_cost: 0,
      billable: true,
    },
  ],
  notes: [
    {
      id: 'n-1',
      job_id: 'job-1',
      user_id: 'tech-1',
      user_name: 'Tasha Green',
      note: 'Lobby door sticks',
      is_internal: false,
      created_at: '2026-11-03T23:40:00Z',
    },
  ],
  estimated_cost: 0,
  created_at: '2026-11-01T00:00:00Z',
  updated_at: '2026-11-01T00:00:00Z',
};

const renderPage = (path = '/my-day/job-1?date=2026-11-03') =>
  render(
    <QueryClientProvider
      client={
        new QueryClient({ defaultOptions: { queries: { retry: false } } })
      }
    >
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/my-day/:jobId" element={<VisitPage />} />
          <Route path="/my-day" element={<p>My day page</p>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  );

describe('VisitPage (T173)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(jobService.getJob).mockResolvedValue(job);
    vi.mocked(jobService.addNote).mockResolvedValue(undefined);
    vi.mocked(jobService.addMaterial).mockResolvedValue(undefined);
  });

  it('shows what to do, the site, notes and materials', async () => {
    renderPage();
    expect(
      await screen.findByRole('heading', { name: 'Elm Street Plaza' })
    ).toBeInTheDocument();
    expect(jobService.getJob).toHaveBeenCalledWith('job-1');
    expect(screen.getByText('In progress')).toBeInTheDocument();
    expect(screen.getByText('J-3318')).toBeInTheDocument();
    expect(screen.getByText('Started 5:32 PM')).toBeInTheDocument();
    expect(
      screen.getByText('Tasha Green', { selector: 'span' })
    ).toBeInTheDocument();

    expect(
      screen.getByText(/Vacuum lobby and entry floors/)
    ).toBeInTheDocument();
    expect(screen.getByText('Site note: Key code 4471')).toBeInTheDocument();
    expect(
      screen.getByRole('link', { name: /Ray Burke · 555-0100/ })
    ).toHaveAttribute('href', 'tel:5550100');
    expect(
      within(screen.getByRole('list', { name: 'Notes' })).getByText(
        'Lobby door sticks'
      )
    ).toBeInTheDocument();
    expect(
      within(screen.getByRole('list', { name: 'Materials' })).getByText(
        '12 each'
      )
    ).toBeInTheDocument();

    // Back to the same day; no money, no photo or signature placeholders.
    expect(screen.getByRole('link', { name: 'My day' })).toHaveAttribute(
      'href',
      '/my-day?date=2026-11-03'
    );
    expect(screen.queryByText(/\$/)).not.toBeInTheDocument();
    expect(screen.queryByText(/photo/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/signature/i)).not.toBeInTheDocument();
  });

  it('completes the visit', async () => {
    vi.mocked(jobService.transitionStatus).mockResolvedValue({
      ...job,
      status: JobStatus.COMPLETED,
    });
    renderPage();
    const user = userEvent.setup();
    await user.click(await screen.findByRole('button', { name: 'Complete' }));
    expect(jobService.transitionStatus).toHaveBeenCalledWith(
      'job-1',
      JobStatus.COMPLETED
    );
    // Reloaded after the change.
    expect(vi.mocked(jobService.getJob).mock.calls.length).toBeGreaterThan(0);
  });

  it('adds a note', async () => {
    renderPage();
    const user = userEvent.setup();
    await screen.findByRole('heading', { name: 'Elm Street Plaza' });

    await user.click(screen.getByRole('button', { name: 'Add note' }));
    const box = screen.getByLabelText('Add a note');
    expect(box).toHaveFocus();
    await user.type(box, 'Restocked restrooms');
    await user.click(screen.getByRole('button', { name: 'Save note' }));

    expect(jobService.addNote).toHaveBeenCalledWith(
      'job-1',
      'Restocked restrooms'
    );
  });

  it('logs a material without a price', async () => {
    renderPage();
    const user = userEvent.setup();
    await screen.findByRole('heading', { name: 'Elm Street Plaza' });

    await user.type(screen.getByLabelText('Material'), 'Paper towels');
    const qty = screen.getByLabelText('Quantity');
    await user.clear(qty);
    await user.type(qty, '6');
    const unit = screen.getByLabelText('Unit');
    await user.clear(unit);
    await user.type(unit, 'rolls');
    await user.click(screen.getByRole('button', { name: 'Add material' }));

    expect(jobService.addMaterial).toHaveBeenCalledWith('job-1', {
      name: 'Paper towels',
      quantity: 6,
      unit: 'rolls',
    });
  });

  it('does not offer to add an empty material', async () => {
    renderPage();
    await screen.findByRole('heading', { name: 'Elm Street Plaza' });
    expect(screen.getByRole('button', { name: 'Add material' })).toBeDisabled();
  });

  it('shows Visit not found for a job the caller cannot see', async () => {
    vi.mocked(jobService.getJob).mockRejectedValue(
      new AxiosError('Not Found', '404', undefined, undefined, {
        status: 404,
        data: { error: 'job_not_found', message: 'Job not found' },
      } as AxiosResponse)
    );
    renderPage();
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Visit not found'
    );
  });
});
