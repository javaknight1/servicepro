import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { PlanSelector } from '../../../src/components/subscriptions/PlanSelector';
import type { Plan, BillingCycle } from '../../../src/types/subscription';

// Mock plans for testing
const mockPlans: Plan[] = [
  {
    id: 'plan-free',
    name: 'Free',
    description: 'Get started with basic features',
    tier: 'free',
    monthlyPrice: 0,
    annualPrice: 0,
    quarterlyPrice: 0,
    currency: 'USD',
    trialDays: 0,
    isActive: true,
    isPublic: true,
    sortOrder: 1,
    features: [
      {
        id: 'f1',
        planId: 'plan-free',
        featureKey: 'projects',
        featureName: 'Projects',
        featureType: 'numeric',
        numericValue: 3,
        isEnabled: true,
        sortOrder: 1,
      },
    ],
  },
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
    sortOrder: 2,
    features: [
      {
        id: 'f2',
        planId: 'plan-starter',
        featureKey: 'projects',
        featureName: 'Projects',
        featureType: 'numeric',
        numericValue: 10,
        isEnabled: true,
        sortOrder: 1,
      },
      {
        id: 'f3',
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
    sortOrder: 3,
    features: [
      {
        id: 'f4',
        planId: 'plan-pro',
        featureKey: 'projects',
        featureName: 'Projects',
        featureType: 'numeric',
        numericValue: 50,
        isEnabled: true,
        sortOrder: 1,
      },
      {
        id: 'f5',
        planId: 'plan-pro',
        featureKey: 'api_access',
        featureName: 'API Access',
        featureType: 'boolean',
        isEnabled: true,
        sortOrder: 2,
      },
      {
        id: 'f6',
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

describe('PlanSelector Component', () => {
  const mockOnSelectPlan = vi.fn();
  const mockOnChangeBillingCycle = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Rendering', () => {
    it('should render all plans', () => {
      render(<PlanSelector plans={mockPlans} billingCycle="monthly" />);

      expect(screen.getByText('Free')).toBeInTheDocument();
      expect(screen.getByText('Starter')).toBeInTheDocument();
      expect(screen.getByText('Professional')).toBeInTheDocument();
    });

    it('should render plan descriptions', () => {
      render(<PlanSelector plans={mockPlans} billingCycle="monthly" />);

      expect(
        screen.getByText('Get started with basic features')
      ).toBeInTheDocument();
      expect(screen.getByText('Perfect for small teams')).toBeInTheDocument();
    });

    it('should render monthly prices correctly', () => {
      render(<PlanSelector plans={mockPlans} billingCycle="monthly" />);

      expect(screen.getByText('$0')).toBeInTheDocument();
      expect(screen.getByText('$29.00')).toBeInTheDocument();
      expect(screen.getByText('$79.00')).toBeInTheDocument();
    });

    it('should render annual prices correctly', () => {
      render(<PlanSelector plans={mockPlans} billingCycle="annual" />);

      expect(screen.getByText('$290.00')).toBeInTheDocument();
      expect(screen.getByText('$790.00')).toBeInTheDocument();
    });

    it('should display billing cycle toggle when onChangeBillingCycle provided', () => {
      render(
        <PlanSelector
          plans={mockPlans}
          billingCycle="monthly"
          onChangeBillingCycle={mockOnChangeBillingCycle}
        />
      );

      expect(
        screen.getByRole('button', { name: 'Monthly' })
      ).toBeInTheDocument();
      expect(
        screen.getByRole('button', { name: /Annual/i })
      ).toBeInTheDocument();
    });

    it('should not display billing cycle toggle without onChangeBillingCycle', () => {
      render(<PlanSelector plans={mockPlans} billingCycle="monthly" />);

      expect(
        screen.queryByRole('button', { name: 'Monthly' })
      ).not.toBeInTheDocument();
    });

    it('should highlight the selected plan', () => {
      const { container } = render(
        <PlanSelector
          plans={mockPlans}
          billingCycle="monthly"
          selectedPlanId="plan-starter"
        />
      );

      // Check that the selected plan has different styling
      const selectedCard = container.querySelector('.border-primary-500');
      expect(selectedCard).toBeInTheDocument();
    });

    it('should show current badge for current plan', () => {
      render(
        <PlanSelector
          plans={mockPlans}
          billingCycle="monthly"
          currentPlanId="plan-starter"
        />
      );

      expect(screen.getByText('Current')).toBeInTheDocument();
    });

    it('should show Most Popular badge for highlighted plan', () => {
      render(
        <PlanSelector
          plans={mockPlans}
          billingCycle="monthly"
          highlightedPlanId="plan-pro"
        />
      );

      expect(screen.getByText('Most Popular')).toBeInTheDocument();
    });
  });

  describe('Interactions', () => {
    it('should call onSelectPlan when plan is clicked', async () => {
      const user = userEvent.setup();
      render(
        <PlanSelector
          plans={mockPlans}
          billingCycle="monthly"
          onSelectPlan={mockOnSelectPlan}
        />
      );

      const selectButtons = screen.getAllByRole('button', {
        name: 'Select Plan',
      });
      await user.click(selectButtons[0]);

      expect(mockOnSelectPlan).toHaveBeenCalledWith('plan-free');
    });

    it('should call onChangeBillingCycle when toggle is clicked', async () => {
      const user = userEvent.setup();
      render(
        <PlanSelector
          plans={mockPlans}
          billingCycle="monthly"
          onChangeBillingCycle={mockOnChangeBillingCycle}
        />
      );

      const annualButton = screen.getByRole('button', { name: /Annual/i });
      await user.click(annualButton);

      expect(mockOnChangeBillingCycle).toHaveBeenCalledWith('annual');
    });

    it('should not call onSelectPlan for current plan', async () => {
      const user = userEvent.setup();
      render(
        <PlanSelector
          plans={mockPlans}
          billingCycle="monthly"
          currentPlanId="plan-starter"
          onSelectPlan={mockOnSelectPlan}
        />
      );

      const currentPlanButton = screen.getByRole('button', {
        name: 'Current Plan',
      });
      await user.click(currentPlanButton);

      expect(mockOnSelectPlan).not.toHaveBeenCalled();
    });

    it('should disable plan selection when disabled prop is true', async () => {
      const user = userEvent.setup();
      render(
        <PlanSelector
          plans={mockPlans}
          billingCycle="monthly"
          onSelectPlan={mockOnSelectPlan}
          disabled
        />
      );

      const selectButtons = screen.getAllByRole('button', {
        name: 'Select Plan',
      });
      await user.click(selectButtons[0]);

      expect(mockOnSelectPlan).not.toHaveBeenCalled();
    });
  });

  describe('Plan Features', () => {
    it('should display plan features', () => {
      render(<PlanSelector plans={mockPlans} billingCycle="monthly" />);

      expect(screen.getAllByText('Projects')).toHaveLength(3);
      expect(screen.getAllByText('API Access')).toHaveLength(2);
    });

    it('should show feature numeric values', () => {
      render(<PlanSelector plans={mockPlans} billingCycle="monthly" />);

      expect(screen.getByText('(3)')).toBeInTheDocument();
      expect(screen.getByText('(10)')).toBeInTheDocument();
      expect(screen.getByText('(50)')).toBeInTheDocument();
    });

    it('should show trial days info', () => {
      render(<PlanSelector plans={mockPlans} billingCycle="monthly" />);

      expect(screen.getAllByText('14-day free trial')).toHaveLength(2);
    });
  });

  describe('Feature Comparison', () => {
    it('should render comparison table when showComparison is true', () => {
      render(
        <PlanSelector plans={mockPlans} billingCycle="monthly" showComparison />
      );

      expect(screen.getByText('Compare All Features')).toBeInTheDocument();
    });

    it('should not render comparison table by default', () => {
      render(<PlanSelector plans={mockPlans} billingCycle="monthly" />);

      expect(
        screen.queryByText('Compare All Features')
      ).not.toBeInTheDocument();
    });
  });

  describe('Savings Display', () => {
    it('should show savings badge on annual toggle', () => {
      render(
        <PlanSelector
          plans={mockPlans}
          billingCycle="monthly"
          onChangeBillingCycle={mockOnChangeBillingCycle}
        />
      );

      expect(screen.getByText('Save 17%')).toBeInTheDocument();
    });

    it('should show annual savings when billing cycle is annual', () => {
      render(
        <PlanSelector
          plans={mockPlans}
          billingCycle="annual"
          onChangeBillingCycle={mockOnChangeBillingCycle}
        />
      );

      // Check that savings info is displayed
      const savingsText = screen.getAllByText(/Save \$/i);
      expect(savingsText.length).toBeGreaterThan(0);
    });
  });

  describe('Custom ClassName', () => {
    it('should apply custom className', () => {
      const { container } = render(
        <PlanSelector
          plans={mockPlans}
          billingCycle="monthly"
          className="custom-class"
        />
      );

      expect(container.firstChild).toHaveClass('custom-class');
    });
  });

  describe('Enterprise Plan', () => {
    const plansWithEnterprise: Plan[] = [
      ...mockPlans,
      {
        id: 'plan-enterprise',
        name: 'Enterprise',
        description: 'Custom solutions',
        tier: 'enterprise',
        monthlyPrice: 0,
        annualPrice: 0,
        quarterlyPrice: 0,
        currency: 'USD',
        trialDays: 30,
        isActive: true,
        isPublic: true,
        sortOrder: 4,
        features: [],
      },
    ];

    it('should display Custom pricing for enterprise', () => {
      render(
        <PlanSelector plans={plansWithEnterprise} billingCycle="monthly" />
      );

      expect(screen.getByText('Custom')).toBeInTheDocument();
      expect(screen.getByText('Contact us for pricing')).toBeInTheDocument();
    });

    it('should show Contact Sales button for enterprise', () => {
      render(
        <PlanSelector plans={plansWithEnterprise} billingCycle="monthly" />
      );

      expect(
        screen.getByRole('button', { name: 'Contact Sales' })
      ).toBeInTheDocument();
    });
  });
});
