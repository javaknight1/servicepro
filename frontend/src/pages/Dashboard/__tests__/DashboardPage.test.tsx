import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import type { ReactNode } from 'react';
import type { Tenant } from '@/types/tenant';
import type { CustomerStats } from '@app-types/customer';
import type { JobStats } from '@app-types/job';
import type { InvoiceStats } from '@app-types/invoice';
import type { QuoteStats } from '@app-types/quote';

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
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
};

vi.mock('@store', () => ({
  useAuthStore: (select: (s: { user: { email: string } }) => unknown) =>
    select({ user: { email: 'owner@acme.test' } }),
  useTenantStore: () => ({
    currentTenant: tenant,
    tenants: [tenant],
    isLoading: false,
    loadTenants: vi.fn().mockResolvedValue(undefined),
  }),
}));

vi.mock('@services/customerService', () => ({
  customerService: { getStats: vi.fn() },
}));
vi.mock('@services/jobService', () => ({
  jobService: { getStats: vi.fn(), getJobs: vi.fn() },
}));
vi.mock('@services/quoteService', () => ({
  quoteService: { getStats: vi.fn(), getQuotes: vi.fn() },
}));
vi.mock('@services/invoiceService', () => ({
  invoiceService: { getStats: vi.fn() },
}));

import { customerService } from '@services/customerService';
import { jobService } from '@services/jobService';
import { quoteService } from '@services/quoteService';
import { invoiceService } from '@services/invoiceService';
import { DashboardPage } from '../DashboardPage';

const customerStats: CustomerStats = {
  total_customers: 12,
  active_customers: 9,
  inactive_customers: 2,
  prospects: 1,
  by_status: { active: 9, inactive: 2, prospect: 1 },
};

const jobStats: JobStats = {
  total_jobs: 20,
  pending_jobs: 4,
  scheduled_jobs: 3,
  in_progress_jobs: 2,
  on_hold_jobs: 1,
  completed_jobs: 9,
  cancelled_jobs: 1,
  by_status: {},
};

const quoteStats: QuoteStats = {
  total_quotes: 5,
  draft_quotes: 2,
  sent_quotes: 3,
  accepted_quotes: 0,
  declined_quotes: 0,
  expired_quotes: 0,
};

const invoiceStats: InvoiceStats = {
  total_outstanding: '2430.50',
  outstanding_invoices: 3,
  overdue_amount: '810.00',
  overdue_invoices: 2,
  collected_this_month: '1620.00',
};

/** The StatsCard body (label, value, change) whose label is `label`. */
const card = (label: string) => {
  const body = screen.getByText(label).parentElement;
  if (!body) throw new Error(`no card for ${label}`);
  return body;
};

describe('DashboardPage stats (T257)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(customerService.getStats).mockResolvedValue(customerStats);
    vi.mocked(jobService.getStats).mockResolvedValue(jobStats);
    vi.mocked(quoteService.getStats).mockResolvedValue(quoteStats);
    vi.mocked(invoiceService.getStats).mockResolvedValue(invoiceStats);
    vi.mocked(jobService.getJobs).mockResolvedValue({
      jobs: [],
      total: 0,
      page: 1,
      page_size: 5,
    } as unknown as Awaited<ReturnType<typeof jobService.getJobs>>);
    vi.mocked(quoteService.getQuotes).mockResolvedValue({
      quotes: [],
      total: 0,
      page: 1,
      page_size: 5,
    } as unknown as Awaited<ReturnType<typeof quoteService.getQuotes>>);
  });

  it('shows the real numbers from the stats endpoints', async () => {
    render(
      <MemoryRouter>
        <DashboardPage />
      </MemoryRouter>
    );

    await waitFor(() =>
      expect(screen.getByText('$2,430.50')).toBeInTheDocument()
    );

    // Outstanding money and overdue count come from GET /invoices/stats.
    expect(card('Outstanding A/R')).toHaveTextContent('$810.00 overdue');
    // Active jobs = scheduled + in progress + on hold; pending = new.
    expect(card('Active Jobs')).toHaveTextContent('6');
    expect(card('Active Jobs')).toHaveTextContent('4 pending');
    // Customers: total and active_customers.
    expect(card('Total Customers')).toHaveTextContent('12');
    expect(card('Total Customers')).toHaveTextContent('9 active');
    expect(card('Pending Quotes')).toHaveTextContent('3');

    expect(invoiceService.getStats).toHaveBeenCalledTimes(1);
  });
});
