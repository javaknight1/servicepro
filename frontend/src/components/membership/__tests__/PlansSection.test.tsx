import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { AxiosResponse, InternalAxiosRequestConfig } from 'axios';
import type {
  MembershipTier,
  MembershipTiersResponse,
  PlanCheckoutRequest,
  PlanCheckoutResponse,
  TenantMembership,
} from '@/types/membership';

const getAllTiers =
  vi.fn<() => Promise<AxiosResponse<MembershipTiersResponse>>>();
const getTenantMembership =
  vi.fn<(tenantId: string) => Promise<AxiosResponse<TenantMembership>>>();
const startCheckout =
  vi.fn<
    (
      tenantId: string,
      data: PlanCheckoutRequest
    ) => Promise<AxiosResponse<PlanCheckoutResponse>>
  >();

vi.mock('@/services/membershipService', () => ({
  membershipApi: {
    getAllTiers: () => getAllTiers(),
    getTenantMembership: (id: string) => getTenantMembership(id),
    startCheckout: (id: string, data: PlanCheckoutRequest) =>
      startCheckout(id, data),
  },
  membershipTiersQueryKey: ['membership-tiers'],
  tenantMembershipQueryKey: (id: string) => ['tenant-membership', id],
}));

const redirectTo = vi.fn<(url: string) => void>();
vi.mock('@/utils/redirect', () => ({
  redirectTo: (url: string) => redirectTo(url),
}));

import { PlansSection } from '../PlansSection';

function ok<T>(data: T): AxiosResponse<T> {
  return {
    data,
    status: 200,
    statusText: 'OK',
    headers: {},
    config: { headers: {} } as InternalAxiosRequestConfig,
  };
}

function tier(
  name: MembershipTier['name'],
  monthly: number,
  users: number,
  sort: number
): MembershipTier {
  return {
    id: `tier-${name}`,
    name,
    display_name: name.charAt(0).toUpperCase() + name.slice(1),
    monthly_price_cents: monthly,
    annual_price_cents: monthly * 10,
    features: {
      team_members: users,
      advanced_reporting: false,
      api_access: false,
      custom_branding: true,
      highlights: [`${name} highlight`],
    },
    enabled: true,
    sort_order: sort,
  };
}

const tiers = [
  tier('pro', 19900, -1, 3),
  tier('solo', 3900, 1, 1),
  tier('team', 9900, -1, 2),
];

function membership(over: Partial<TenantMembership> = {}): TenantMembership {
  return {
    subscription: null,
    tier: tiers[2],
    trial: { ends_at: '2026-12-03T12:00:00Z', days_left: 12, ended: false },
    has_paid_subscription: false,
    checkout_available: true,
    founding_offer_available: false,
    ...over,
  };
}

function renderSection(canChoose = true) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <PlansSection tenantId="tenant-1" canChoose={canChoose} />
      </MemoryRouter>
    </QueryClientProvider>
  );
}

describe('PlansSection (T210)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    getAllTiers.mockResolvedValue(ok({ tiers }));
  });

  it('shows Solo, Team and Pro in order with trial days left', async () => {
    getTenantMembership.mockResolvedValue(ok(membership()));
    renderSection();

    const cards = await screen.findAllByTestId(/^tier-card-/);
    expect(cards.map((c) => c.dataset.testid)).toEqual([
      'tier-card-solo',
      'tier-card-team',
      'tier-card-pro',
    ]);
    expect(within(cards[0]).getByText('$39.00')).toBeInTheDocument();
    expect(within(cards[0]).getByText('1 user')).toBeInTheDocument();
    expect(within(cards[1]).getByText('Unlimited users')).toBeInTheDocument();
    expect(await screen.findByText(/12 days left/)).toBeInTheDocument();
  });

  it('switches to annual prices', async () => {
    getTenantMembership.mockResolvedValue(ok(membership()));
    renderSection();
    const user = userEvent.setup();

    await screen.findByTestId('tier-card-pro');
    await user.click(screen.getByRole('button', { name: /annual/i }));
    expect(
      within(screen.getByTestId('tier-card-pro')).getByText('$1,990.00')
    ).toBeInTheDocument();
  });

  it('shows the founding price on Team monthly only', async () => {
    getTenantMembership.mockResolvedValue(
      ok(membership({ founding_offer_available: true }))
    );
    renderSection();
    const user = userEvent.setup();

    const team = await screen.findByTestId('tier-card-team');
    await waitFor(() =>
      expect(within(team).getByText('$59.00')).toBeInTheDocument()
    );
    expect(screen.getByText(/Founding offer/)).toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: /annual/i }));
    expect(within(team).getByText('$990.00')).toBeInTheDocument();
    expect(within(team).queryByText('$59.00')).not.toBeInTheDocument();
  });

  it('Choose plan starts Checkout and redirects to Stripe', async () => {
    getTenantMembership.mockResolvedValue(ok(membership()));
    startCheckout.mockResolvedValue(
      ok({ url: 'https://checkout.stripe.test/c/1', founding_offer: false })
    );
    renderSection();
    const user = userEvent.setup();

    const solo = await screen.findByTestId('tier-card-solo');
    await waitFor(() =>
      expect(
        within(solo).getByRole('button', { name: 'Choose plan' })
      ).toBeEnabled()
    );
    await user.click(within(solo).getByRole('button', { name: 'Choose plan' }));

    await waitFor(() =>
      expect(redirectTo).toHaveBeenCalledWith(
        'https://checkout.stripe.test/c/1'
      )
    );
    expect(startCheckout).toHaveBeenCalledWith('tenant-1', {
      tier_id: 'tier-solo',
      billing_cycle: 'monthly',
    });
  });

  it('shows a non-blocking notice once the trial has ended', async () => {
    getTenantMembership.mockResolvedValue(
      ok(
        membership({
          trial: { ends_at: '2026-10-01T12:00:00Z', days_left: 0, ended: true },
        })
      )
    );
    renderSection();
    expect(
      await screen.findByText(/free trial of Team ended/)
    ).toBeInTheDocument();
    expect(screen.getAllByRole('button', { name: 'Choose plan' }).length).toBe(
      3
    );
  });

  it('disables Choose plan when plan billing is not set up', async () => {
    getTenantMembership.mockResolvedValue(
      ok(membership({ checkout_available: false }))
    );
    renderSection();
    expect(
      await screen.findByText(/plan billing isn.t switched on yet/i)
    ).toBeInTheDocument();
    for (const b of screen.getAllByRole('button', { name: 'Choose plan' })) {
      expect(b).toBeDisabled();
    }
  });

  it('hides Choose plan for members without membership.update', async () => {
    getTenantMembership.mockResolvedValue(ok(membership()));
    renderSection(false);
    await screen.findByTestId('tier-card-team');
    expect(
      screen.queryByRole('button', { name: 'Choose plan' })
    ).not.toBeInTheDocument();
    expect(screen.getByText(/Ask an organization owner/)).toBeInTheDocument();
  });

  it('marks the current paid plan and offers a change instead', async () => {
    getTenantMembership.mockResolvedValue(
      ok(membership({ has_paid_subscription: true, trial: undefined }))
    );
    renderSection();
    const team = await screen.findByTestId('tier-card-team');
    await waitFor(() =>
      expect(within(team).getByText('Current Plan')).toBeInTheDocument()
    );
    expect(
      screen.getByRole('link', { name: /change your plan/i })
    ).toBeInTheDocument();
  });
});
