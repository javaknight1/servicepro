import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CancellationFlow } from '../../../src/components/subscriptions/CancellationFlow';
import type { Subscription, Plan } from '../../../src/types/subscription';

// Mock plan
const mockPlan: Plan = {
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
    {
      id: 'f3',
      planId: 'plan-starter',
      featureKey: 'support',
      featureName: 'Email Support',
      featureType: 'boolean',
      isEnabled: true,
      sortOrder: 3,
    },
    {
      id: 'f4',
      planId: 'plan-starter',
      featureKey: 'storage',
      featureName: '10GB Storage',
      featureType: 'boolean',
      isEnabled: true,
      sortOrder: 4,
    },
  ],
};

// Mock subscription
const createMockSubscription = (
  overrides: Partial<Subscription> = {}
): Subscription => ({
  id: 'sub-123',
  customerId: 'cust-456',
  plan: mockPlan,
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

describe('CancellationFlow Component', () => {
  const mockOnCancel = vi.fn();
  const mockOnKeepSubscription = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Step 1 - Reason Selection', () => {
    it('should render the initial step', () => {
      const subscription = createMockSubscription();
      render(
        <CancellationFlow
          subscription={subscription}
          onCancel={mockOnCancel}
          onKeepSubscription={mockOnKeepSubscription}
        />
      );

      expect(screen.getByText("We're sorry to see you go")).toBeInTheDocument();
      expect(
        screen.getByText(
          "Please let us know why you're cancelling so we can improve."
        )
      ).toBeInTheDocument();
    });

    it('should display cancellation reasons', () => {
      const subscription = createMockSubscription();
      render(
        <CancellationFlow
          subscription={subscription}
          onCancel={mockOnCancel}
          onKeepSubscription={mockOnKeepSubscription}
        />
      );

      expect(screen.getByText("I don't need it anymore")).toBeInTheDocument();
      expect(screen.getByText("It's too expensive")).toBeInTheDocument();
      expect(screen.getByText('Missing features')).toBeInTheDocument();
      expect(screen.getByText('Switching to competitor')).toBeInTheDocument();
      expect(screen.getByText('Other reason')).toBeInTheDocument();
    });

    it('should disable Continue button until reason is selected', () => {
      const subscription = createMockSubscription();
      render(
        <CancellationFlow
          subscription={subscription}
          onCancel={mockOnCancel}
          onKeepSubscription={mockOnKeepSubscription}
        />
      );

      const continueButton = screen.getByRole('button', { name: 'Continue' });
      expect(continueButton).toBeDisabled();
    });

    it('should enable Continue button after selecting a reason', async () => {
      const user = userEvent.setup();
      const subscription = createMockSubscription();

      render(
        <CancellationFlow
          subscription={subscription}
          onCancel={mockOnCancel}
          onKeepSubscription={mockOnKeepSubscription}
        />
      );

      await user.click(screen.getByText("I don't need it anymore"));

      const continueButton = screen.getByRole('button', { name: 'Continue' });
      expect(continueButton).not.toBeDisabled();
    });

    it('should call onKeepSubscription when "Never mind" is clicked', async () => {
      const user = userEvent.setup();
      const subscription = createMockSubscription();

      render(
        <CancellationFlow
          subscription={subscription}
          onCancel={mockOnCancel}
          onKeepSubscription={mockOnKeepSubscription}
        />
      );

      await user.click(screen.getByRole('button', { name: 'Never mind' }));

      expect(mockOnKeepSubscription).toHaveBeenCalled();
    });
  });

  describe('Step 2 - Feedback', () => {
    const goToStep2 = async (user: ReturnType<typeof userEvent.setup>) => {
      await user.click(screen.getByText("I don't need it anymore"));
      await user.click(screen.getByRole('button', { name: 'Continue' }));
    };

    it('should navigate to step 2 after selecting reason', async () => {
      const user = userEvent.setup();
      const subscription = createMockSubscription();

      render(
        <CancellationFlow
          subscription={subscription}
          onCancel={mockOnCancel}
          onKeepSubscription={mockOnKeepSubscription}
        />
      );

      await goToStep2(user);

      expect(screen.getByText('Any additional feedback?')).toBeInTheDocument();
    });

    it('should display feedback textarea', async () => {
      const user = userEvent.setup();
      const subscription = createMockSubscription();

      render(
        <CancellationFlow
          subscription={subscription}
          onCancel={mockOnCancel}
          onKeepSubscription={mockOnKeepSubscription}
        />
      );

      await goToStep2(user);

      expect(
        screen.getByPlaceholderText('Tell us more about your experience...')
      ).toBeInTheDocument();
    });

    it('should allow typing feedback', async () => {
      const user = userEvent.setup();
      const subscription = createMockSubscription();

      render(
        <CancellationFlow
          subscription={subscription}
          onCancel={mockOnCancel}
          onKeepSubscription={mockOnKeepSubscription}
        />
      );

      await goToStep2(user);

      const textarea = screen.getByPlaceholderText(
        'Tell us more about your experience...'
      );
      await user.type(textarea, 'This is my feedback');

      expect(textarea).toHaveValue('This is my feedback');
    });

    it('should navigate back to step 1', async () => {
      const user = userEvent.setup();
      const subscription = createMockSubscription();

      render(
        <CancellationFlow
          subscription={subscription}
          onCancel={mockOnCancel}
          onKeepSubscription={mockOnKeepSubscription}
        />
      );

      await goToStep2(user);
      await user.click(screen.getByRole('button', { name: 'Back' }));

      expect(screen.getByText("We're sorry to see you go")).toBeInTheDocument();
    });
  });

  describe('Step 3 - Confirmation', () => {
    const goToStep3 = async (user: ReturnType<typeof userEvent.setup>) => {
      await user.click(screen.getByText("I don't need it anymore"));
      await user.click(screen.getByRole('button', { name: 'Continue' }));
      await user.click(screen.getByRole('button', { name: 'Continue' }));
    };

    it('should navigate to step 3', async () => {
      const user = userEvent.setup();
      const subscription = createMockSubscription();

      render(
        <CancellationFlow
          subscription={subscription}
          onCancel={mockOnCancel}
          onKeepSubscription={mockOnKeepSubscription}
        />
      );

      await goToStep3(user);

      expect(screen.getByText('Confirm Cancellation')).toBeInTheDocument();
    });

    it('should show what features will be lost', async () => {
      const user = userEvent.setup();
      const subscription = createMockSubscription();

      render(
        <CancellationFlow
          subscription={subscription}
          onCancel={mockOnCancel}
          onKeepSubscription={mockOnKeepSubscription}
        />
      );

      await goToStep3(user);

      expect(
        screen.getByText("What you'll lose access to:")
      ).toBeInTheDocument();
      expect(screen.getByText('Projects')).toBeInTheDocument();
    });

    it('should show cancellation timing options', async () => {
      const user = userEvent.setup();
      const subscription = createMockSubscription();

      render(
        <CancellationFlow
          subscription={subscription}
          onCancel={mockOnCancel}
          onKeepSubscription={mockOnKeepSubscription}
        />
      );

      await goToStep3(user);

      expect(
        screen.getByText('Cancel at end of billing period')
      ).toBeInTheDocument();
      expect(screen.getByText('Cancel immediately')).toBeInTheDocument();
    });

    it('should show subscription summary', async () => {
      const user = userEvent.setup();
      const subscription = createMockSubscription();

      render(
        <CancellationFlow
          subscription={subscription}
          onCancel={mockOnCancel}
          onKeepSubscription={mockOnKeepSubscription}
        />
      );

      await goToStep3(user);

      expect(screen.getByText('Current plan')).toBeInTheDocument();
      expect(screen.getByText('Starter')).toBeInTheDocument();
      expect(screen.getByText('Monthly cost')).toBeInTheDocument();
    });

    it('should default to cancel at period end', async () => {
      const user = userEvent.setup();
      const subscription = createMockSubscription();

      render(
        <CancellationFlow
          subscription={subscription}
          onCancel={mockOnCancel}
          onKeepSubscription={mockOnKeepSubscription}
        />
      );

      await goToStep3(user);

      // The "Cancel at end of billing period" option should be checked by default
      const periodEndRadio = screen
        .getByText('Cancel at end of billing period')
        .closest('label')
        ?.querySelector('input');
      expect(periodEndRadio).toBeChecked();
    });

    it('should allow selecting cancel immediately', async () => {
      const user = userEvent.setup();
      const subscription = createMockSubscription();

      render(
        <CancellationFlow
          subscription={subscription}
          onCancel={mockOnCancel}
          onKeepSubscription={mockOnKeepSubscription}
        />
      );

      await goToStep3(user);

      const immediateOption = screen
        .getByText('Cancel immediately')
        .closest('label');
      await user.click(immediateOption!);

      const immediateRadio = immediateOption?.querySelector('input');
      expect(immediateRadio).toBeChecked();
    });
  });

  describe('Cancellation Submission', () => {
    it('should call onCancel with correct parameters for period end', async () => {
      const user = userEvent.setup();
      const subscription = createMockSubscription();

      render(
        <CancellationFlow
          subscription={subscription}
          onCancel={mockOnCancel}
          onKeepSubscription={mockOnKeepSubscription}
        />
      );

      // Navigate through all steps
      await user.click(screen.getByText("I don't need it anymore"));
      await user.click(screen.getByRole('button', { name: 'Continue' }));
      await user.click(screen.getByRole('button', { name: 'Continue' }));
      await user.click(
        screen.getByRole('button', { name: 'Confirm Cancellation' })
      );

      expect(mockOnCancel).toHaveBeenCalledWith(
        'user_requested',
        undefined,
        false
      );
    });

    it('should call onCancel with immediate flag when selected', async () => {
      const user = userEvent.setup();
      const subscription = createMockSubscription();

      render(
        <CancellationFlow
          subscription={subscription}
          onCancel={mockOnCancel}
          onKeepSubscription={mockOnKeepSubscription}
        />
      );

      await user.click(screen.getByText("I don't need it anymore"));
      await user.click(screen.getByRole('button', { name: 'Continue' }));
      await user.click(screen.getByRole('button', { name: 'Continue' }));

      const immediateOption = screen
        .getByText('Cancel immediately')
        .closest('label');
      await user.click(immediateOption!);

      await user.click(
        screen.getByRole('button', { name: 'Confirm Cancellation' })
      );

      expect(mockOnCancel).toHaveBeenCalledWith(
        'user_requested',
        undefined,
        true
      );
    });

    it('should include feedback when provided', async () => {
      const user = userEvent.setup();
      const subscription = createMockSubscription();

      render(
        <CancellationFlow
          subscription={subscription}
          onCancel={mockOnCancel}
          onKeepSubscription={mockOnKeepSubscription}
        />
      );

      await user.click(screen.getByText("I don't need it anymore"));
      await user.click(screen.getByRole('button', { name: 'Continue' }));

      const textarea = screen.getByPlaceholderText(
        'Tell us more about your experience...'
      );
      await user.type(textarea, 'My feedback here');

      await user.click(screen.getByRole('button', { name: 'Continue' }));
      await user.click(
        screen.getByRole('button', { name: 'Confirm Cancellation' })
      );

      expect(mockOnCancel).toHaveBeenCalledWith(
        'user_requested',
        'My feedback here',
        false
      );
    });
  });

  describe('Special Offer', () => {
    it('should display special offer on confirmation step', async () => {
      const user = userEvent.setup();
      const subscription = createMockSubscription();

      render(
        <CancellationFlow
          subscription={subscription}
          onCancel={mockOnCancel}
          onKeepSubscription={mockOnKeepSubscription}
        />
      );

      await user.click(screen.getByText("I don't need it anymore"));
      await user.click(screen.getByRole('button', { name: 'Continue' }));
      await user.click(screen.getByRole('button', { name: 'Continue' }));

      expect(
        screen.getByText('Wait! Special offer just for you')
      ).toBeInTheDocument();
      expect(
        screen.getByText(/20% off for the next 3 months/i)
      ).toBeInTheDocument();
    });

    it('should call onKeepSubscription when claiming offer', async () => {
      const user = userEvent.setup();
      const subscription = createMockSubscription();

      render(
        <CancellationFlow
          subscription={subscription}
          onCancel={mockOnCancel}
          onKeepSubscription={mockOnKeepSubscription}
        />
      );

      await user.click(screen.getByText("I don't need it anymore"));
      await user.click(screen.getByRole('button', { name: 'Continue' }));
      await user.click(screen.getByRole('button', { name: 'Continue' }));

      await user.click(
        screen.getByRole('button', { name: 'Claim Offer & Stay' })
      );

      expect(mockOnKeepSubscription).toHaveBeenCalled();
    });
  });

  describe('Step Indicator', () => {
    it('should show progress indicators', () => {
      const subscription = createMockSubscription();
      const { container } = render(
        <CancellationFlow
          subscription={subscription}
          onCancel={mockOnCancel}
          onKeepSubscription={mockOnKeepSubscription}
        />
      );

      const indicators = container.querySelectorAll('.rounded-full.h-2.w-2');
      expect(indicators.length).toBe(3);
    });

    it('should update progress indicators as steps change', async () => {
      const user = userEvent.setup();
      const subscription = createMockSubscription();

      const { container } = render(
        <CancellationFlow
          subscription={subscription}
          onCancel={mockOnCancel}
          onKeepSubscription={mockOnKeepSubscription}
        />
      );

      await user.click(screen.getByText("I don't need it anymore"));
      await user.click(screen.getByRole('button', { name: 'Continue' }));

      const filledIndicators = container.querySelectorAll('.bg-primary-600');
      expect(filledIndicators.length).toBeGreaterThan(0);
    });
  });

  describe('Processing State', () => {
    it('should show processing state when isProcessing', async () => {
      const user = userEvent.setup();
      const subscription = createMockSubscription();

      render(
        <CancellationFlow
          subscription={subscription}
          onCancel={mockOnCancel}
          onKeepSubscription={mockOnKeepSubscription}
          isProcessing
        />
      );

      await user.click(screen.getByText("I don't need it anymore"));
      await user.click(screen.getByRole('button', { name: 'Continue' }));
      await user.click(screen.getByRole('button', { name: 'Continue' }));

      expect(screen.getByText('Cancelling...')).toBeInTheDocument();
    });

    it('should disable buttons when processing', async () => {
      const user = userEvent.setup();
      const subscription = createMockSubscription();

      render(
        <CancellationFlow
          subscription={subscription}
          onCancel={mockOnCancel}
          onKeepSubscription={mockOnKeepSubscription}
          isProcessing
        />
      );

      await user.click(screen.getByText("I don't need it anymore"));
      await user.click(screen.getByRole('button', { name: 'Continue' }));
      await user.click(screen.getByRole('button', { name: 'Continue' }));

      expect(screen.getByRole('button', { name: 'Back' })).toBeDisabled();
    });
  });

  describe('Keep Subscription Links', () => {
    it('should show "changed my mind" link on confirmation step', async () => {
      const user = userEvent.setup();
      const subscription = createMockSubscription();

      render(
        <CancellationFlow
          subscription={subscription}
          onCancel={mockOnCancel}
          onKeepSubscription={mockOnKeepSubscription}
        />
      );

      await user.click(screen.getByText("I don't need it anymore"));
      await user.click(screen.getByRole('button', { name: 'Continue' }));
      await user.click(screen.getByRole('button', { name: 'Continue' }));

      expect(
        screen.getByText('I changed my mind - keep my subscription')
      ).toBeInTheDocument();
    });

    it('should call onKeepSubscription when clicked', async () => {
      const user = userEvent.setup();
      const subscription = createMockSubscription();

      render(
        <CancellationFlow
          subscription={subscription}
          onCancel={mockOnCancel}
          onKeepSubscription={mockOnKeepSubscription}
        />
      );

      await user.click(screen.getByText("I don't need it anymore"));
      await user.click(screen.getByRole('button', { name: 'Continue' }));
      await user.click(screen.getByRole('button', { name: 'Continue' }));

      await user.click(
        screen.getByText('I changed my mind - keep my subscription')
      );

      expect(mockOnKeepSubscription).toHaveBeenCalled();
    });
  });

  describe('Custom ClassName', () => {
    it('should apply custom className', () => {
      const subscription = createMockSubscription();
      const { container } = render(
        <CancellationFlow
          subscription={subscription}
          onCancel={mockOnCancel}
          onKeepSubscription={mockOnKeepSubscription}
          className="custom-class"
        />
      );

      expect(container.firstChild).toHaveClass('custom-class');
    });
  });
});
