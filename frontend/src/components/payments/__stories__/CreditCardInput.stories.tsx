import type { Meta, StoryObj } from '@storybook/react';
import { action } from '@storybook/addon-actions';
import { Elements } from '@stripe/react-stripe-js';
import { loadStripe } from '@stripe/stripe-js';
import { CreditCardInput } from '../CreditCardInput';

// Note: In a real implementation, you would use your actual Stripe publishable key
// For Storybook, we use a test key or mock
const stripePromise = loadStripe('pk_test_placeholder');

const StripeWrapper = ({ children }: { children: React.ReactNode }) => (
  <Elements stripe={stripePromise}>{children}</Elements>
);

const meta: Meta<typeof CreditCardInput> = {
  title: 'Payments/CreditCardInput',
  component: CreditCardInput,
  tags: ['autodocs'],
  decorators: [
    (Story) => (
      <StripeWrapper>
        <div className="max-w-md">
          <Story />
        </div>
      </StripeWrapper>
    ),
  ],
  parameters: {
    layout: 'padded',
    docs: {
      description: {
        component: `
A credit card input component that wraps Stripe Elements CardElement.

**Note:** This component requires the Stripe Elements context to function properly.
In Storybook, the Stripe Elements may not render correctly without a valid API key.

Features:
- Integrated Stripe Elements for secure card handling
- Visual feedback for completion and errors
- Accessibility support with ARIA attributes
- Security notice display
        `,
      },
    },
  },
  argTypes: {
    disabled: {
      control: 'boolean',
      description: 'Disable the card input',
    },
    error: {
      control: 'text',
      description: 'Error message to display',
    },
    onChange: {
      action: 'changed',
      description: 'Callback when card input changes',
    },
    onReady: {
      action: 'ready',
      description: 'Callback when card element is ready',
    },
    onFocus: {
      action: 'focused',
      description: 'Callback when card element is focused',
    },
    onBlur: {
      action: 'blurred',
      description: 'Callback when card element loses focus',
    },
    className: {
      control: 'text',
      description: 'Additional CSS classes',
    },
  },
};

export default meta;
type Story = StoryObj<typeof CreditCardInput>;

export const Default: Story = {
  args: {
    onChange: action('onChange'),
    onReady: action('onReady'),
    onFocus: action('onFocus'),
    onBlur: action('onBlur'),
  },
};

export const WithError: Story = {
  args: {
    error: 'Your card number is invalid.',
    onChange: action('onChange'),
  },
  parameters: {
    docs: {
      description: {
        story: 'Display an error message below the card input',
      },
    },
  },
};

export const Disabled: Story = {
  args: {
    disabled: true,
  },
  parameters: {
    docs: {
      description: {
        story: 'Disabled state prevents user interaction',
      },
    },
  },
};

export const WithCallbacks: Story = {
  args: {
    onChange: action('card-changed'),
    onReady: action('card-ready'),
    onFocus: action('card-focused'),
    onBlur: action('card-blurred'),
  },
  parameters: {
    docs: {
      description: {
        story: 'All callbacks are triggered during user interaction',
      },
    },
  },
};

export const CustomClassName: Story = {
  args: {
    className: 'bg-neutral-50 p-4 rounded-lg',
    onChange: action('onChange'),
  },
  parameters: {
    docs: {
      description: {
        story: 'Custom styling via className prop',
      },
    },
  },
};

// Interactive example showing typical usage
export const InteractiveExample: Story = {
  render: () => {
    // This would normally use hooks but we'll show the static version
    return (
      <div className="space-y-4">
        <div className="p-4 bg-primary-50 border border-primary-200 rounded-lg">
          <h3 className="text-sm font-medium text-primary-800 mb-2">
            Test Card Numbers
          </h3>
          <ul className="text-xs text-primary-700 space-y-1">
            <li>
              <code>4242 4242 4242 4242</code> - Visa (success)
            </li>
            <li>
              <code>4000 0000 0000 0002</code> - Visa (decline)
            </li>
            <li>
              <code>5555 5555 5555 4444</code> - Mastercard
            </li>
            <li>Use any future date for expiry</li>
            <li>Use any 3 digits for CVC</li>
          </ul>
        </div>
        <CreditCardInput
          onChange={action('onChange')}
          onReady={action('onReady')}
        />
      </div>
    );
  },
  parameters: {
    docs: {
      description: {
        story: 'Interactive example with test card numbers for development',
      },
    },
  },
};
