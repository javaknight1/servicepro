import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Route, Routes } from 'react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import type { Tenant } from '@/types/tenant';
import type { JobStats } from '@app-types/job';
import type { InvoiceStats } from '@app-types/invoice';
import type { ARAgingBucketSummary } from '@app-types/arAging';
import type { Job, JobListResponse } from '@services/jobService';

vi.mock('@components/layout', () => ({
  DashboardLayout: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
}));

const tenant: Tenant = {
  id: 'tenant-1',
  name: 'Acme HVAC',
  slug: 'acme',
  owner_id: 'owner-1',
  is_active: true,
  settings: { timezone: 'America/New_York' },
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
};

vi.mock('@store', () => ({
  useAuthStore: (
    select: (s: {
      user: { id: string; email: string; first_name: string };
    }) => unknown
  ) =>
    select({
      user: { id: 'owner-1', email: 'owner@acme.test', first_name: 'Dana' },
    }),
  useTenantStore: () => ({
    currentTenant: tenant,
    tenants: [tenant],
    isLoading: false,
    loadTenants: vi.fn().mockResolvedValue(undefined),
  }),
}));

vi.mock('@services/jobService', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@services/jobService')>()),
  jobService: { getStats: vi.fn(), getJobs: vi.fn(), invoiceJob: vi.fn() },
}));
vi.mock('@services/invoiceService', () => ({
  invoiceService: { getStats: vi.fn() },
}));
vi.mock('@services/arAgingService', () => ({
  arAgingService: { getBucketSummary: vi.fn() },
}));
vi.mock('@services/tenantService', () => ({
  tenantApi: { getMyAccess: vi.fn() },
}));

import { jobService, JobStatus } from '@services/jobService';
import { invoiceService } from '@services/invoiceService';
import { arAgingService } from '@services/arAgingService';
import { tenantApi } from '@services/tenantService';
import { DashboardPage } from '../DashboardPage';

const jobStats: JobStats = {
  total_jobs: 20,
  pending_jobs: 4,
  scheduled_jobs: 3,
  in_progress_jobs: 2,
  on_hold_jobs: 1,
  completed_jobs: 9,
  cancelled_jobs: 1,
  by_status: { completed: 7 },
};

const invoiceStats: InvoiceStats = {
  total_outstanding: '2430.50',
  outstanding_invoices: 3,
  overdue_amount: '810.00',
  overdue_invoices: 2,
  collected_this_month: '1620.00',
};

const bucket = (
  b: ARAgingBucketSummary['bucket'],
  amount: string
): ARAgingBucketSummary => ({
  bucket: b,
  display_name: b,
  invoice_count: 1,
  total_amount: amount,
  percentage: '0',
  color: '',
});

const aging: ARAgingBucketSummary[] = [
  bucket('current', '1000.00'),
  bucket('1-30', '620.50'),
  bucket('31-60', '500.00'),
  bucket('61-90', '200.00'),
  bucket('90+', '110.00'),
];

// 2026-10-29T15:00:00Z
const DONE_OCT_29 = 1793286000;

const completedJob = {
  id: 'job-1',
  job_number: 'J-3317',
  customer_id: 'cust-1',
  customer: {
    id: 'cust-1',
    email: 'ap@polaris.test',
    first_name: 'Pat',
    last_name: 'Lee',
    company_name: 'Polaris Pet Clinic',
  },
  title: 'Deep clean, exam rooms',
  status: JobStatus.COMPLETED,
  actual_end_at: DONE_OCT_29,
  estimated_cost: 380,
} as unknown as Job;

const list = (jobs: Job[], total: number) =>
  ({ jobs, total, page: 1, page_size: 6 }) as JobListResponse;

/** The StatsCard whose label is `label`. */
const card = (label: string) => {
  const body = screen
    .getByText(label, { selector: 'p' })
    .closest<HTMLElement>('div.rounded-2xl');
  if (!body) throw new Error(`no card for ${label}`);
  return body;
};

const renderPage = () =>
  render(
    <QueryClientProvider
      client={
        new QueryClient({ defaultOptions: { queries: { retry: false } } })
      }
    >
      <MemoryRouter initialEntries={['/dashboard']}>
        <Routes>
          <Route path="/dashboard" element={<DashboardPage />} />
          <Route path="/invoices/:id" element={<p>Invoice page</p>} />
          <Route path="/my-day" element={<p>My day page</p>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  );

/** GET /tenants/:id/me answers with these permissions. */
const grant = (role: string, permissions: string[]) =>
  vi.mocked(tenantApi.getMyAccess).mockResolvedValue({
    data: {
      tenant_id: 'tenant-1',
      role_id: `r-${role}`,
      role,
      role_level: 30,
      permissions,
    },
  } as unknown as Awaited<ReturnType<typeof tenantApi.getMyAccess>>);

describe('DashboardPage (T269)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(jobService.getStats).mockResolvedValue(jobStats);
    vi.mocked(invoiceService.getStats).mockResolvedValue(invoiceStats);
    vi.mocked(arAgingService.getBucketSummary).mockResolvedValue(aging);
    vi.mocked(jobService.getJobs).mockResolvedValue(list([completedJob], 7));
    // The permission list stays loading: the office fallback applies.
    vi.mocked(tenantApi.getMyAccess).mockReturnValue(new Promise(() => {}));
  });

  // T235: an owner who left the onboarding wizard can pick it up again.
  it('offers to resume an unfinished onboarding', async () => {
    tenant.settings = { onboardingStep: 2 };
    try {
      renderPage();
      expect(
        await screen.findByText('Finish setting up your workspace')
      ).toBeInTheDocument();
      expect(screen.getByText(/Step 2 of 3/)).toBeInTheDocument();
    } finally {
      tenant.settings = { timezone: 'America/New_York' };
    }
  });

  it('hides the onboarding prompt once finished', async () => {
    tenant.settings = {
      onboardingStep: 3,
      onboardingCompletedAt: '2026-11-03T00:00:00Z',
    };
    try {
      renderPage();
      await screen.findByRole('heading', { name: 'Welcome back, Dana' });
      expect(
        screen.queryByText('Finish setting up your workspace')
      ).not.toBeInTheDocument();
    } finally {
      tenant.settings = { timezone: 'America/New_York' };
    }
  });

  it('greets the user by first name', async () => {
    renderPage();
    expect(
      await screen.findByRole('heading', { name: 'Welcome back, Dana' })
    ).toBeInTheDocument();
  });

  it('shows the KPI numbers from the stats endpoints', async () => {
    renderPage();
    await waitFor(() =>
      expect(card('Outstanding A/R')).toHaveTextContent('$2,431')
    );

    // Over 30 days = the 31-60, 61-90 and 90+ aging buckets.
    expect(card('Outstanding A/R')).toHaveTextContent('$810 over 30 days');
    // Active = scheduled + in progress + on hold.
    expect(card('Active jobs')).toHaveTextContent('6');
    // Completed jobs not yet invoiced: the completed-status list total.
    expect(card('Jobs needing invoice')).toHaveTextContent('7');
    expect(card('Jobs needing invoice')).toHaveTextContent(
      'Oldest done Oct 29'
    );
    expect(screen.getByText(/^Collected in /).parentElement).toHaveTextContent(
      '$1,620'
    );
  });

  it('lists completed jobs with a button to invoice each one', async () => {
    renderPage();
    const row = (await screen.findByText('Polaris Pet Clinic')).closest('li');
    if (!row) throw new Error('no row');
    expect(within(row).getByText('J-3317')).toBeInTheDocument();
    expect(within(row).getByText('$380.00')).toBeInTheDocument();
    expect(
      within(row).getByRole('button', { name: 'Create invoice for J-3317' })
    ).toBeEnabled();
    expect(screen.getByText('6 more ready to invoice')).toBeInTheDocument();

    expect(jobService.getJobs).toHaveBeenCalledWith(
      expect.objectContaining({ status: JobStatus.COMPLETED })
    );
  });

  // T029: the API drafts the invoice from the job, then it opens.
  it('generates the invoice from the job and opens it', async () => {
    vi.mocked(jobService.invoiceJob).mockResolvedValue({
      id: 'inv-9',
      lines: [],
    } as unknown as Awaited<ReturnType<typeof jobService.invoiceJob>>);
    renderPage();
    const user = userEvent.setup();
    await user.click(
      await screen.findByRole('button', { name: 'Create invoice for J-3317' })
    );

    expect(jobService.invoiceJob).toHaveBeenCalledWith('job-1');
    expect(await screen.findByText('Invoice page')).toBeInTheDocument();
  });

  it('opens the existing invoice when the job was already invoiced', async () => {
    vi.mocked(jobService.invoiceJob).mockRejectedValue({
      response: {
        status: 409,
        data: {
          error: 'already_invoiced',
          message: 'This job has already been invoiced',
          resource_id: 'inv-1',
        },
      },
    });
    renderPage();
    const user = userEvent.setup();
    await user.click(
      await screen.findByRole('button', { name: 'Create invoice for J-3317' })
    );
    expect(await screen.findByText('Invoice page')).toBeInTheDocument();
  });

  it("shows the API's reason when the job can't be invoiced", async () => {
    vi.mocked(jobService.invoiceJob).mockRejectedValue({
      response: {
        status: 409,
        data: {
          error: 'job_not_completed',
          message: 'Only completed jobs can be invoiced',
        },
      },
    });
    renderPage();
    const user = userEvent.setup();
    await user.click(
      await screen.findByRole('button', { name: 'Create invoice for J-3317' })
    );
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Only completed jobs can be invoiced'
    );
  });

  it('leaves agreement visits to the consolidated invoice', async () => {
    vi.mocked(jobService.getJobs).mockResolvedValue(
      list([{ ...completedJob, agreement_id: 'agr-1' }], 1)
    );
    renderPage();
    const row = (await screen.findByText('Polaris Pet Clinic')).closest('li');
    if (!row) throw new Error('no row');
    expect(within(row).getByText('On agreement')).toBeInTheDocument();
    expect(within(row).queryByRole('button')).toBeNull();
  });

  it('shows the A/R aging buckets', async () => {
    renderPage();
    expect(await screen.findByText('1–30 days')).toBeInTheDocument();
    expect(screen.getByText('$621')).toBeInTheDocument();
    // KPI value and the aging panel's total.
    expect(screen.getAllByText('$2,431')).toHaveLength(2);
  });

  it('shows an empty state when nothing needs invoicing', async () => {
    vi.mocked(jobService.getJobs).mockResolvedValue(list([], 0));
    renderPage();
    expect(
      await screen.findByText('Nothing waiting to be invoiced')
    ).toBeInTheDocument();
  });
});

describe('DashboardPage for technicians (T035)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(jobService.getStats).mockResolvedValue(jobStats);
    vi.mocked(jobService.getJobs).mockResolvedValue(
      list(
        [
          {
            ...completedJob,
            status: JobStatus.SCHEDULED,
            estimated_cost: 0,
          } as Job,
        ],
        1
      )
    );
  });

  // T173: technicians land on My day instead of the office dashboard.
  it('sends technicians to My day without loading the dashboard', async () => {
    tenant.role = 'technician';
    grant('technician', ['jobs.read', 'jobs.list', 'jobs.update']);
    try {
      renderPage();
      expect(await screen.findByText('My day page')).toBeInTheDocument();
      expect(jobService.getStats).not.toHaveBeenCalled();
      expect(jobService.getJobs).not.toHaveBeenCalled();
      expect(invoiceService.getStats).not.toHaveBeenCalled();
    } finally {
      tenant.role = undefined;
    }
  });

  // A custom role that sees every job but no money keeps a money-free
  // dashboard of jobs. (Until its permission list arrives the office
  // fallback applies to custom role names, so money requests may go out
  // first; the API refuses them. Technicians never get that far.)
  it('shows jobs and no money cards or invoice buttons', async () => {
    tenant.role = 'dispatcher';
    grant('dispatcher', ['jobs.read', 'jobs.list', 'jobs.view_all']);
    try {
      renderPage();
      expect(await screen.findByText('Your jobs')).toBeInTheDocument();
      expect(await screen.findByText('Polaris Pet Clinic')).toBeInTheDocument();
      expect(card('Your active jobs')).toHaveTextContent('6');

      for (const label of [
        'Outstanding A/R',
        'Jobs needing invoice',
        'A/R aging',
        'Quick actions',
      ]) {
        expect(screen.queryByText(label)).not.toBeInTheDocument();
      }
      expect(screen.queryByText(/^Collected in /)).not.toBeInTheDocument();
      expect(screen.queryByText(/\$/)).not.toBeInTheDocument();
      expect(
        screen.queryByRole('button', { name: /new job/i })
      ).not.toBeInTheDocument();
      expect(
        screen.queryByRole('button', { name: /create invoice/i })
      ).not.toBeInTheDocument();
    } finally {
      tenant.role = undefined;
    }
  });
});
