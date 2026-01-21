import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SubscriptionManager } from '../../../src/components/subscriptions/SubscriptionManager';
import type { Plan, Subscription } from '../../../src/types/subscription';

// Mock plans
const mockPlans: Plan[] = [
  {
    id: 'plan-starter',
    name: 'Starter',
    description: 'Perfect for small teams',
    tier: 'starter',
    monthlyPrice: 2900,
    annualPrice: 29000,
    quarterlyPrice: 7800,
    currency: 'USD',
    trialDays: 14,
    isActive: true,
    isPublic: true,
    sortOrder: 1,
    features: [
      {
        id: 'f1',
        planId: 'plan-starter',
        featureKey: 'projects',
        featureName: 'Projects',
        featureType: 'numeric',
        numericValue: 10,
        isEnabled: true,
        sortOrder: 1,
      },
      {
        id: 'f2',
        planId: 'plan-starter',
        featureKey: 'api_access',
        featureName: 'API Access',
        featureType: 'boolean',
        isEnabled: true,
        sortOrder: 2,
      },
    ],
  },
  {
    id: 'plan-pro',
    name: 'Professional',
    description: 'Advanced features for pros',
    tier: 'professional',
    monthlyPrice: 7900,
    annualPrice: 79000,
    quarterlyPrice: 21000,
    currency: 'USD',
    trialDays: 14,
    isActive: true,
    isPublic: true,
    sortOrder: 2,
    features: [
      {
        id: 'f3',
        planId: 'plan-pro',
        featureKey: 'projects',
        featureName: 'Projects',
        featureType: 'numeric',
        numericValue: 50,
        isEnabled: true,
        sortOrder: 1,
      },
      {
        id: 'f4',
        planId: 'plan-pro',
        featureKey: 'api_access',
        featureName: 'API Access',
        featureType: 'boolean',
        isEnabled: true,
        sortOrder: 2,
      },
      {
        id: 'f5',
        planId: 'plan-pro',
        featureKey: 'priority_support',
        featureName: 'Priority Support',
        featureType: 'boolean',
        isEnabled: true,
        sortOrder: 3,
      },
    ],
  },
];

// Mock subscription
const createMockSubscription = (
  overrides: Partial<Subscription> = {}
): Subscription => ({
  id: 'sub-123',
  customerId: 'cust-456',
  plan: mockPlans[0],
  status: 'active',
  billingCycle: 'monthly',
  currentPeriodStart: new Date().toISOString(),
  currentPeriodEnd: new Date(
    Date.now() + 30 * 24 * 60 * 60 * 1000
  ).toISOString(),
  cancelAtPeriodEnd: false,
  daysUntilRenewal: 30,
  trialDaysRemaining: 0,
  createdAt: new Date().toISOString(),
  ...overrides,
});

describe('SubscriptionManager Component', () => {
  const mockOnChangePlan = vi.fn();
  const mockOnChangeBillingCycle = vi.fn();
  const mockOnCancel = vi.fn();
  const mockOnReactivate = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Rendering', () => {
    it('should render plan selector when no subscription', () => {
      render(<SubscriptionManager plans={mockPlans} />);

      expect(screen.getByText('Choose a Plan')).toBeInTheDocument();
    });

    it('should render current subscription details', () => {
      const subscription = createMockSubscription();
      render(
        <SubscriptionManager subscription={subscription} plans={mockPlans} />
      );

      expect(screen.getByText('Starter')).toBeInTheDocument();
      expect(screen.getByText('Active')).toBeInTheDocument();
    });

    it('should display subscription price', () => {
      const subscription = createMockSubscription();
      render(
        <SubscriptionManager subscription={subscription} plans={mockPlans} />
      );

      expect(screen.getByText('$29.00')).toBeInTheDocument();
      expect(screen.getByText('per month')).toBeInTheDocument();
    });

    it('should show loading skeleton when isLoading', () => {
      render(<SubscriptionManager plans={mockPlans} isLoading />);

      // Check for skeleton elements
      const skeletons = document.querySelectorAll('.animate-pulse');
      expect(skeletons.length).toBeGreaterThan(0);
    });

    it('should display error message', () => {
      render(
        <SubscriptionManager plans={mockPlans} error="Something went wrong" />
      );

      expect(screen.getByText('Something went wrong')).toBeInTheDocument();
    });
  });

  describe('Subscription Status', () => {
    it('should show trial status with days remaining', () => {
      const subscription = createMockSubscription({
        status: 'trialing',
        trialDaysRemaining: 7,
        trialEnd: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString(),
      });

      render(
        <SubscriptionManager subscription={subscription} plans={mockPlans} />
      );

      expect(screen.getByText('Trial')).toBeInTheDocument();
      expect(
        screen.getByText(/7 days left in your trial/i)
      ).toBeInTheDocument();
    });

    it('should show cancellation scheduled message', () => {
      const subscription = createMockSubscription({
        cancelAtPeriodEnd: true,
      });

      render(
        <SubscriptionManager
          subscription={subscription}
          plans={mockPlans}
          onReactivate={mockOnReactivate}
        />
      );

      expect(screen.getByText(/Subscription will end on/i)).toBeInTheDocument();
    });

    it('should show past due warning', () => {
      const subscription = createMockSubscription({
        status: 'past_due',
      });

      render(
        <SubscriptionManager subscription={subscription} plans={mockPlans} />
      );

      expect(screen.getByText('Past Due')).toBeInTheDocument();
      expect(screen.getByText(/Payment failed/i)).toBeInTheDocument();
    });

    it('should show canceled state with resubscribe option', () => {
      const subscription = createMockSubscription({
        status: 'canceled',
      });

      render(
        <SubscriptionManager
          subscription={subscription}
          plans={mockPlans}
          onChangePlan={mockOnChangePlan}
        />
      );

      expect(screen.getByText('Canceled')).toBeInTheDocument();
      expect(screen.getByText('Subscription Canceled')).toBeInTheDocument();
      expect(
        screen.getByRole('button', { name: 'Subscribe Again' })
      ).toBeInTheDocument();
    });
  });

  describe('Features List', () => {
    it('should display included features', () => {
      const subscription = createMockSubscription();
      render(
        <SubscriptionManager subscription={subscription} plans={mockPlans} />
      );

      expect(screen.getByText('Included Features')).toBeInTheDocument();
      expect(screen.getByText('Projects')).toBeInTheDocument();
      expect(screen.getByText('API Access')).toBeInTheDocument();
    });
  });

  describe('Plan Change', () => {
    it('should show Change Plan button', () => {
      const subscription = createMockSubscription();
      render(
        <SubscriptionManager
          subscription={subscription}
          plans={mockPlans}
          onChangePlan={mockOnChangePlan}
        />
      );

      expect(
        screen.getByRole('button', { name: 'Change Plan' })
      ).toBeInTheDocument();
    });

    it('should open plan selector when Change Plan is clicked', async () => {
      const user = userEvent.setup();
      const subscription = createMockSubscription();

      render(
        <SubscriptionManager
          subscription={subscription}
          plans={mockPlans}
          onChangePlan={mockOnChangePlan}
        />
      );

      await user.click(screen.getByRole('button', { name: 'Change Plan' }));

      expect(screen.getByText('Change Plan')).toBeInTheDocument();
      expect(
        screen.getByText('Select a new plan. Changes will be prorated.')
      ).toBeInTheDocument();
    });

    it('should close plan selector when cancel is clicked', async () => {
      const user = userEvent.setup();
      const subscription = createMockSubscription();

      render(
        <SubscriptionManager
          subscription={subscription}
          plans={mockPlans}
          onChangePlan={mockOnChangePlan}
        />
      );

      await user.click(screen.getByRole('button', { name: 'Change Plan' }));
      await user.click(screen.getByRole('button', { name: 'Cancel' }));

      expect(
        screen.queryByText('Select a new plan. Changes will be prorated.')
      ).not.toBeInTheDocument();
    });

    it('should not show Change Plan button if onChangePlan not provided', () => {
      const subscription = createMockSubscription();
      render(
        <SubscriptionManager subscription={subscription} plans={mockPlans} />
      );

      expect(
        screen.queryByRole('button', { name: 'Change Plan' })
      ).not.toBeInTheDocument();
    });
  });

  describe('Cancellation', () => {
    it('should show Cancel Subscription button', () => {
      const subscription = createMockSubscription();
      render(
        <SubscriptionManager
          subscription={subscription}
          plans={mockPlans}
          onCancel={mockOnCancel}
        />
      );

      expect(
        screen.getByRole('button', { name: 'Cancel Subscription' })
      ).toBeInTheDocument();
    });

    it('should not show Cancel button when already scheduled for cancellation', () => {
      const subscription = createMockSubscription({
        cancelAtPeriodEnd: true,
      });

      render(
        <SubscriptionManager
          subscription={subscription}
          plans={mockPlans}
          onCancel={mockOnCancel}
        />
      );

      expect(
        screen.queryByRole('button', { name: 'Cancel Subscription' })
      ).not.toBeInTheDocument();
    });

    it('should open cancellation flow when Cancel is clicked', async () => {
      const user = userEvent.setup();
      const subscription = createMockSubscription();

      render(
        <SubscriptionManager
          subscription={subscription}
          plans={mockPlans}
          onCancel={mockOnCancel}
        />
      );

      await user.click(
        screen.getByRole('button', { name: 'Cancel Subscription' })
      );

      expect(screen.getByText("We're sorry to see you go")).toBeInTheDocument();
    });

    it('should not show Cancel button if onCancel not provided', () => {
      const subscription = createMockSubscription();
      render(
        <SubscriptionManager subscription={subscription} plans={mockPlans} />
      );

      expect(
        screen.queryByRole('button', { name: 'Cancel Subscription' })
      ).not.toBeInTheDocument();
    });
  });

  describe('Reactivation', () => {
    it('should show reactivation link when scheduled for cancellation', () => {
      const subscription = createMockSubscription({
        cancelAtPeriodEnd: true,
      });

      render(
        <SubscriptionManager
          subscription={subscription}
          plans={mockPlans}
          onReactivate={mockOnReactivate}
        />
      );

      expect(
        screen.getByText(/Cancel cancellation and keep subscription/i)
      ).toBeInTheDocument();
    });

    it('should call onReactivate when reactivation is clicked', async () => {
      const user = userEvent.setup();
      const subscription = createMockSubscription({
        cancelAtPeriodEnd: true,
      });

      render(
        <SubscriptionManager
          subscription={subscription}
          plans={mockPlans}
          onReactivate={mockOnReactivate}
        />
      );

      await user.click(
        screen.getByText(/Cancel cancellation and keep subscription/i)
      );

      expect(mockOnReactivate).toHaveBeenCalled();
    });
  });

  describe('Billing Information', () => {
    it('should display billing cycle', () => {
      const subscription = createMockSubscription();
      render(
        <SubscriptionManager subscription={subscription} plans={mockPlans} />
      );

      expect(screen.getByText('Billing cycle')).toBeInTheDocument();
      expect(screen.getByText('Monthly')).toBeInTheDocument();
    });

    it('should display next billing date', () => {
      const subscription = createMockSubscription();
      render(
        <SubscriptionManager subscription={subscription} plans={mockPlans} />
      );

      expect(screen.getByText('Next billing date')).toBeInTheDocument();
    });
  });

  describe('Custom ClassName', () => {
    it('should apply custom className', () => {
      const subscription = createMockSubscription();
      const { container } = render(
        <SubscriptionManager
          subscription={subscription}
          plans={mockPlans}
          className="custom-class"
        />
      );

      expect(container.firstChild).toHaveClass('custom-class');
    });
  });
});
