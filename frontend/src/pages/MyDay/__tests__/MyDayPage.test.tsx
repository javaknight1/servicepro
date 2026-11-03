import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import type { Tenant } from '@/types/tenant';
import type { Job, MyDayResponse } from '@services/jobService';

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
  useAuthStore: (select: (s: { user: object }) => unknown) =>
    select({
      user: {
        id: 'tech-1',
        email: 'marcus@acme.test',
        first_name: 'Marcus',
        last_name: 'Reed',
      },
    }),
  useTenantStore: (select: (s: { currentTenant: Tenant }) => unknown) =>
    select({ currentTenant: tenant }),
}));

vi.mock('@services/jobService', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@services/jobService')>()),
  jobService: { getMyDay: vi.fn(), transitionStatus: vi.fn() },
}));

import { jobService, JobStatus, JobType } from '@services/jobService';
import { MyDayPage } from '../MyDayPage';
import { addDays, todayIn } from '../myDay';

const ZONE = 'America/Chicago';
const today = todayIn(ZONE);

/** Epoch seconds for hh:mm on `date` in Chicago (CST, UTC-6, in Nov). */
const at = (date: string, hour: number, minute = 0) =>
  Date.parse(
    `${date}T${String(hour).padStart(2, '0')}:${String(minute).padStart(2, '0')}:00-06:00`
  ) / 1000;

const visit = (overrides: Partial<Job>): Job => ({
  id: 'job-x',
  job_number: 'J-0000',
  customer_id: 'cust-1',
  customer: {
    id: 'cust-1',
    email: 'ap@harborview.test',
    first_name: 'Ray',
    last_name: 'Burke',
    company_name: 'Harborview Property Management',
    phone_primary: '555-0199',
  },
  title: 'Nightly cleaning',
  job_type: JobType.MAINTENANCE,
  status: JobStatus.SCHEDULED,
  priority: 'normal' as Job['priority'],
  allowed_transitions: [
    JobStatus.EN_ROUTE,
    JobStatus.IN_PROGRESS,
    JobStatus.CANCELLED,
  ],
  estimated_cost: 0,
  created_at: '2026-11-01T00:00:00Z',
  updated_at: '2026-11-01T00:00:00Z',
  ...overrides,
});

const elm = visit({
  id: 'job-1',
  job_number: 'J-3318',
  scheduled_start_at: at(today, 8),
  scheduled_end_at: at(today, 9, 30),
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
    access_notes: 'Key code 4471. Ray Burke signs off.',
    contact_name: 'Ray Burke',
    contact_phone: '(555) 010-0100',
  },
});

const riverside = visit({
  id: 'job-2',
  job_number: 'J-3319',
  title: '3x weekly cleaning',
  scheduled_start_at: at(today, 13, 15),
  scheduled_end_at: at(today, 14, 15),
  service_address: {
    street: '12 River Rd',
    city: 'Springfield',
    state: 'IL',
    zip: '62702',
  },
  // Prices never reach a technician; even if one did, the page has no
  // place to show it.
  estimated_cost: 480,
  total_amount: 480,
});

const day = (date: string, visits: Job[]): MyDayResponse => ({
  date,
  timezone: ZONE,
  visits,
});

const renderPage = (path = '/my-day') =>
  render(
    <QueryClientProvider
      client={
        new QueryClient({ defaultOptions: { queries: { retry: false } } })
      }
    >
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/my-day" element={<MyDayPage />} />
          <Route path="/my-day/:jobId" element={<p>Visit page</p>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  );

const cards = () =>
  within(screen.getByRole('list', { name: 'Visits' })).getAllByRole('listitem');

describe('MyDayPage (T173)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(jobService.getMyDay).mockImplementation(async (date) =>
      day(date ?? today, date === today ? [elm, riverside] : [])
    );
  });

  it("lists today's visits in the server's time order with site details", async () => {
    renderPage();
    expect(await screen.findByText('Elm Street Plaza')).toBeInTheDocument();
    expect(jobService.getMyDay).toHaveBeenCalledWith(today);
    expect(screen.getByRole('heading', { name: 'My day' })).toBeInTheDocument();

    const [first, second] = cards();
    expect(within(first).getByText('Elm Street Plaza')).toBeInTheDocument();
    expect(within(first).getByText('8:00 AM')).toBeInTheDocument();
    expect(within(second).getByText('3x weekly cleaning')).toBeInTheDocument();
    expect(within(second).getByText('1:15 PM')).toBeInTheDocument();
    expect(screen.getByText('2 stops')).toBeInTheDocument();

    // Address → directions, site contact → call, access notes shown.
    const address = within(first).getByRole('link', {
      name: '410 Elm St, Springfield, IL 62701',
    });
    expect(address.getAttribute('href')).toContain(
      'google.com/maps/dir/?api=1&destination=410%20Elm%20St'
    );
    expect(
      within(first).getByRole('link', { name: /Ray Burke · \(555\) 010-0100/ })
    ).toHaveAttribute('href', 'tel:5550100100');
    expect(
      within(first).getByText('Site note: Key code 4471. Ray Burke signs off.')
    ).toBeInTheDocument();
    expect(
      within(first).getByRole('link', {
        name: 'Directions to Elm Street Plaza',
      })
    ).toBeInTheDocument();

    // Without a site the job's service address and the customer's phone.
    expect(
      within(second).getByRole('link', { name: /12 River Rd/ })
    ).toBeInTheDocument();
    expect(
      within(second).getByRole('link', { name: /555-0199/ })
    ).toHaveAttribute('href', 'tel:5550199');

    // Visit title opens the visit page for the day.
    expect(
      within(first).getByRole('link', { name: 'Elm Street Plaza' })
    ).toHaveAttribute('href', `/my-day/job-1?date=${today}`);
  });

  it('never shows money', async () => {
    renderPage();
    await screen.findByText('Elm Street Plaza');
    expect(screen.queryByText(/\$/)).not.toBeInTheDocument();
    expect(screen.queryByText(/480/)).not.toBeInTheDocument();
  });

  it('starts the next visit through the job state machine', async () => {
    vi.mocked(jobService.transitionStatus).mockResolvedValue({
      ...elm,
      status: JobStatus.IN_PROGRESS,
    });
    renderPage();
    const user = userEvent.setup();
    const start = await screen.findByRole('button', {
      name: 'Start Elm Street Plaza',
    });
    // The next visit gets the filled button; later ones an outline one.
    expect(
      screen.getByRole('button', { name: 'Start 3x weekly cleaning' })
    ).toBeInTheDocument();
    await user.click(start);

    expect(jobService.transitionStatus).toHaveBeenCalledWith(
      'job-1',
      JobStatus.IN_PROGRESS
    );
    // The day is reloaded after the change.
    await waitFor(() =>
      expect(vi.mocked(jobService.getMyDay).mock.calls.length).toBeGreaterThan(
        1
      )
    );
  });

  it('offers Complete on the visit in progress and nothing on done ones', async () => {
    vi.mocked(jobService.getMyDay).mockResolvedValue(
      day(today, [
        {
          ...elm,
          status: JobStatus.COMPLETED,
          actual_end_at: at(today, 9, 5),
          allowed_transitions: [JobStatus.INVOICED, JobStatus.IN_PROGRESS],
        },
        {
          ...riverside,
          status: JobStatus.IN_PROGRESS,
          allowed_transitions: [
            JobStatus.ON_HOLD,
            JobStatus.COMPLETED,
            JobStatus.CANCELLED,
          ],
        },
      ])
    );
    vi.mocked(jobService.transitionStatus).mockResolvedValue(riverside);
    renderPage();
    const user = userEvent.setup();

    const [done, current] = await waitFor(() => cards());
    expect(within(done).getByText('Done 9:05 AM')).toBeInTheDocument();
    expect(within(done).queryByRole('button')).toBeNull();
    expect(within(current).getByText('In progress')).toBeInTheDocument();

    await user.click(
      within(current).getByRole('button', {
        name: 'Complete 3x weekly cleaning',
      })
    );
    expect(jobService.transitionStatus).toHaveBeenCalledWith(
      'job-2',
      JobStatus.COMPLETED
    );
  });

  it("shows the API's reason when the action fails", async () => {
    vi.mocked(jobService.transitionStatus).mockRejectedValue(
      new Error("This job can't move to that status")
    );
    renderPage();
    const user = userEvent.setup();
    await user.click(
      await screen.findByRole('button', { name: 'Start Elm Street Plaza' })
    );
    expect(await screen.findByRole('alert')).toHaveTextContent(
      "This job can't move to that status"
    );
  });

  it('switches between yesterday, today and tomorrow', async () => {
    renderPage();
    const user = userEvent.setup();
    await screen.findByText('Elm Street Plaza');
    expect(screen.getByRole('button', { name: 'Today' })).toHaveAttribute(
      'aria-pressed',
      'true'
    );

    await user.click(screen.getByRole('button', { name: 'Tomorrow' }));
    expect(jobService.getMyDay).toHaveBeenCalledWith(addDays(today, 1));
    expect(await screen.findByText('No visits this day')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Tomorrow' })).toHaveAttribute(
      'aria-pressed',
      'true'
    );

    await user.click(screen.getByRole('button', { name: 'Yesterday' }));
    expect(jobService.getMyDay).toHaveBeenCalledWith(addDays(today, -1));

    await user.click(screen.getByRole('button', { name: 'Today' }));
    expect(await screen.findByText('Elm Street Plaza')).toBeInTheDocument();
  });

  it('opens on the day in the link', async () => {
    const tomorrow = addDays(today, 1);
    renderPage(`/my-day?date=${tomorrow}`);
    await screen.findByText('No visits this day');
    expect(jobService.getMyDay).toHaveBeenCalledWith(tomorrow);
  });

  it('shows an empty state with no visits', async () => {
    vi.mocked(jobService.getMyDay).mockResolvedValue(day(today, []));
    renderPage();
    expect(await screen.findByText('No visits today')).toBeInTheDocument();
  });
});
