import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Route, Routes } from 'react-router';
import type { ReactElement, ReactNode } from 'react';

vi.mock('@components/layout', () => ({
  DashboardLayout: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
}));
vi.mock('@services/agreementService', async (importOriginal) => {
  const actual =
    await importOriginal<typeof import('@services/agreementService')>();
  return {
    ...actual,
    agreementService: {
      list: vi.fn(),
      get: vi.fn(),
      create: vi.fn(),
      update: vi.fn(),
      pause: vi.fn(),
      resume: vi.fn(),
      remove: vi.fn(),
      preview: vi.fn(),
    },
  };
});
vi.mock('@services/siteService', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@services/siteService')>();
  return { ...actual, siteService: { list: vi.fn() } };
});
vi.mock('@services/customerLookup', () => ({ fetchAllCustomers: vi.fn() }));
vi.mock('@services/customerService', () => ({
  customerService: { getCustomer: vi.fn() },
}));
vi.mock('@services/tenantService', () => ({
  tenantApi: { getMembers: vi.fn() },
}));
vi.mock('@store/tenantStore', () => ({
  useTenantStore: (selector: (s: unknown) => unknown) =>
    selector({ currentTenant: { id: 'tenant-1' } }),
}));

import {
  agreementService,
  describeFrequency,
  monthlyAmount,
  type Agreement,
} from '@services/agreementService';
import { siteService, type Site } from '@services/siteService';
import { fetchAllCustomers } from '@services/customerLookup';
import { customerService, type Customer } from '@services/customerService';
import { tenantApi } from '@services/tenantService';
import { AgreementsPage } from '../AgreementsPage';
import { AgreementFormPage } from '../AgreementFormPage';
import { AgreementDetailPage } from '../AgreementDetailPage';
import { formatClock, formatVisit } from '../agreementDisplay';

const customer = {
  id: 'c-1',
  first_name: 'Karen',
  last_name: 'Doyle',
  company_name: 'Harborview Property Management',
  consolidated_billing: true,
} as unknown as Customer;

const site: Site = {
  id: 's-1',
  customer_id: 'c-1',
  name: 'Harborview Lofts',
  address_street: '220 W Spring St',
  address_city: 'Columbus',
  address_state: 'OH',
  address_zip: '43215',
  address_full: '220 W Spring St, Columbus, OH 43215',
  is_active: true,
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
};

const base: Agreement = {
  id: 'ag-1',
  customer_id: 'c-1',
  site_id: 's-1',
  title: 'Nightly cleaning',
  frequency: 'weekdays',
  weekdays: [1, 2, 3, 4, 5],
  start_date: '2026-11-02',
  visit_start_time: '18:00',
  visit_duration_minutes: 120,
  price_per_period: '2400.00',
  billing_period: 'month',
  job_type: 'maintenance' as Agreement['job_type'],
  status: 'active',
  created_at: '2026-11-01T00:00:00Z',
  updated_at: '2026-11-01T00:00:00Z',
};

function renderAt(path: string, routes: ReactElement) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[path]}>
        <Routes>{routes}</Routes>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(fetchAllCustomers).mockResolvedValue([customer]);
  vi.mocked(siteService.list).mockResolvedValue([site]);
  vi.mocked(customerService.getCustomer).mockResolvedValue(customer);
  vi.mocked(tenantApi.getMembers).mockResolvedValue({
    data: {
      members: [
        {
          id: 'm-1',
          user_id: 'u-1',
          email: 'marcus@example.com',
          first_name: 'Marcus',
          last_name: 'Lee',
          role_id: 'r',
          role_name: 'Technician',
          is_active: true,
          joined_at: '2026-01-01T00:00:00Z',
        },
      ],
    },
  } as unknown as Awaited<ReturnType<typeof tenantApi.getMembers>>);
  vi.mocked(agreementService.preview).mockResolvedValue({
    agreement_id: 'ag-1',
    timezone: 'America/New_York',
    visits: [
      {
        date: '2026-11-04',
        starts_at: '2026-11-04T23:00:00Z',
        ends_at: '2026-11-05T01:00:00Z',
      },
      {
        date: '2026-11-05',
        starts_at: '2026-11-05T23:00:00Z',
        ends_at: '2026-11-06T01:00:00Z',
      },
    ],
  });
});

describe('agreement helpers', () => {
  it('describes each frequency like the mockup', () => {
    expect(describeFrequency(base)).toBe('Nightly, Mon–Fri');
    expect(
      describeFrequency({ ...base, frequency: 'weekly', weekdays: [5, 2] })
    ).toBe('Weekly, Tue/Fri');
    expect(
      describeFrequency({
        ...base,
        frequency: 'times_per_week',
        weekdays: [],
        times_per_week: 3,
      })
    ).toBe('3x weekly, M/W/F');
    expect(
      describeFrequency({ ...base, frequency: 'monthly', day_of_month: 22 })
    ).toBe('Monthly on the 22nd');
  });

  it('estimates a monthly amount for per-visit prices', () => {
    expect(monthlyAmount(base)).toBe(2400);
    expect(
      monthlyAmount({
        ...base,
        frequency: 'monthly',
        day_of_month: 1,
        billing_period: 'visit',
        price_per_period: '150',
      })
    ).toBe(150);
  });

  it('formats a wall-clock visit time', () => {
    expect(formatClock('18:00')).toBe('6:00 PM');
    expect(formatClock('07:30')).toBe('7:30 AM');
    expect(formatClock('00:15')).toBe('12:15 AM');
  });

  it('formats a visit as today, tomorrow or a date', () => {
    const now = new Date('2026-11-04T15:00:00Z');
    expect(formatVisit('2026-11-04T23:00:00Z', 'America/New_York', now)).toBe(
      'Today · 6:00 PM'
    );
    expect(formatVisit('2026-11-05T23:00:00Z', 'America/New_York', now)).toBe(
      'Tomorrow · 6:00 PM'
    );
    expect(formatVisit('2026-11-07T23:00:00Z', 'America/New_York', now)).toBe(
      'Sat, Nov 7 · 6:00 PM'
    );
  });
});

describe('AgreementsPage', () => {
  it('lists agreements with site, customer, frequency and price', async () => {
    const user = userEvent.setup();
    vi.mocked(agreementService.list).mockResolvedValue([
      base,
      {
        ...base,
        id: 'ag-2',
        title: 'Porter',
        frequency: 'weekly',
        weekdays: [6],
        price_per_period: '300',
        status: 'paused',
      },
    ]);
    renderAt(
      '/agreements',
      <Route path="/agreements" element={<AgreementsPage />} />
    );

    const table = await screen.findByRole('table');
    expect(await within(table).findAllByText('Harborview Lofts')).toHaveLength(
      2
    );
    expect(
      within(table).getAllByText('Harborview Property Management')
    ).toHaveLength(2);
    expect(within(table).getByText('Nightly, Mon–Fri')).toBeInTheDocument();
    expect(within(table).getAllByText('Consolidated')).toHaveLength(2);
    // Recurring revenue counts active agreements only.
    expect(screen.getByText('$2,400/mo')).toBeInTheDocument();

    await user.click(screen.getByRole('tab', { name: /paused 1/i }));
    expect(within(table).getAllByRole('row')).toHaveLength(2); // header + 1
    expect(within(table).getByText('Paused')).toBeInTheDocument();
  });

  it('searches by site or customer', async () => {
    const user = userEvent.setup();
    vi.mocked(agreementService.list).mockResolvedValue([base]);
    renderAt(
      '/agreements',
      <Route path="/agreements" element={<AgreementsPage />} />
    );
    await screen.findAllByText('Harborview Lofts');
    await user.type(screen.getByRole('searchbox'), 'kessler');
    expect(await screen.findByText('No agreements match')).toBeInTheDocument();
  });
});

describe('AgreementFormPage', () => {
  it('creates a weekly agreement for a site', async () => {
    const user = userEvent.setup();
    vi.mocked(agreementService.create).mockResolvedValue(base);
    renderAt(
      '/agreements/new',
      <>
        <Route path="/agreements/new" element={<AgreementFormPage />} />
        <Route path="/agreements/:id" element={<p>detail page</p>} />
      </>
    );

    await screen.findByRole('option', {
      name: 'Harborview Property Management',
    });
    await user.selectOptions(screen.getByLabelText(/customer/i), 'c-1');
    await screen.findByRole('option', { name: 'Harborview Lofts' });
    await user.selectOptions(screen.getByLabelText(/^site/i), 's-1');
    await user.type(screen.getByLabelText(/title/i), 'Day porter');
    await user.selectOptions(screen.getByLabelText(/frequency/i), 'weekly');
    await user.click(screen.getByRole('button', { name: 'Tue' }));
    await user.click(screen.getByRole('button', { name: 'Fri' }));
    await user.type(screen.getByLabelText(/price \(\$\)/i), '1180');
    await screen.findByRole('option', { name: 'Marcus Lee' });
    await user.selectOptions(screen.getByLabelText(/lead tech/i), 'u-1');
    await user.click(screen.getByRole('button', { name: /create agreement/i }));

    await waitFor(() =>
      expect(agreementService.create).toHaveBeenCalledWith(
        expect.objectContaining({
          customer_id: 'c-1',
          site_id: 's-1',
          title: 'Day porter',
          frequency: 'weekly',
          weekdays: [2, 5],
          price_per_period: '1180',
          billing_period: 'month',
          visit_start_time: '18:00',
          assigned_technician_id: 'u-1',
          end_date: undefined,
        })
      )
    );
    expect(await screen.findByText('detail page')).toBeInTheDocument();
  });

  it('edits an agreement, replacing the schedule and clearing the end date', async () => {
    const user = userEvent.setup();
    vi.mocked(agreementService.get).mockResolvedValue({
      ...base,
      end_date: '2027-10-31',
    });
    vi.mocked(agreementService.update).mockResolvedValue(base);
    renderAt(
      '/agreements/ag-1/edit',
      <>
        <Route path="/agreements/:id/edit" element={<AgreementFormPage />} />
        <Route path="/agreements/:id" element={<p>detail page</p>} />
      </>
    );
    const end = await screen.findByLabelText(/ends/i);
    expect(end).toHaveValue('2027-10-31');
    // Customer and site can't change on an existing agreement.
    expect(screen.getByLabelText(/customer/i)).toBeDisabled();
    await user.clear(end);
    await user.selectOptions(screen.getByLabelText(/frequency/i), 'monthly');
    const day = screen.getByLabelText(/day of the month/i);
    await user.clear(day);
    await user.type(day, '15');
    await user.click(screen.getByRole('button', { name: /save changes/i }));

    await waitFor(() =>
      expect(agreementService.update).toHaveBeenCalledWith(
        'ag-1',
        expect.objectContaining({
          schedule: { frequency: 'monthly', day_of_month: 15 },
          clear_end_date: true,
          clear_technician: true,
        })
      )
    );
  });
});

describe('AgreementDetailPage', () => {
  it('shows the terms and the next visits, and pauses', async () => {
    const user = userEvent.setup();
    vi.mocked(agreementService.get).mockResolvedValue({
      ...base,
      assigned_technician_id: 'u-1',
    });
    vi.mocked(agreementService.pause).mockResolvedValue({
      ...base,
      status: 'paused',
    });
    renderAt(
      '/agreements/ag-1',
      <Route path="/agreements/:id" element={<AgreementDetailPage />} />
    );

    expect(
      await screen.findByRole('heading', { name: 'Harborview Lofts' })
    ).toBeInTheDocument();
    expect(screen.getByText('Nightly, Mon–Fri')).toBeInTheDocument();
    expect(screen.getByText('$2,400.00/month')).toBeInTheDocument();
    expect(await screen.findByText('Marcus Lee')).toBeInTheDocument();
    const visits = await screen.findByRole('list', { name: 'Upcoming visits' });
    expect(within(visits).getAllByRole('listitem')).toHaveLength(2);
    expect(agreementService.preview).toHaveBeenCalledWith('ag-1', 14);

    await user.click(screen.getByRole('button', { name: /pause/i }));
    await waitFor(() =>
      expect(agreementService.pause).toHaveBeenCalledWith('ag-1')
    );
  });

  it('offers resume on a paused agreement', async () => {
    vi.mocked(agreementService.get).mockResolvedValue({
      ...base,
      status: 'paused',
    });
    renderAt(
      '/agreements/ag-1',
      <Route path="/agreements/:id" element={<AgreementDetailPage />} />
    );
    expect(
      await screen.findByRole('button', { name: /resume/i })
    ).toBeInTheDocument();
    expect(
      screen.getByText(/paused: no visits are scheduled/i)
    ).toBeInTheDocument();
    expect(agreementService.preview).not.toHaveBeenCalled();
  });
});
