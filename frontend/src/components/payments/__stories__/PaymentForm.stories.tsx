import type { Meta, StoryObj } from '@storybook/react';
import { action } from '@storybook/addon-actions';
import { PaymentForm } from '../PaymentForm';
import type { PaymentSummaryData } from '../../../types/payment';

const meta: Meta<typeof PaymentForm> = {
  title: 'Payments/PaymentForm',
  component: PaymentForm,
  tags: ['autodocs'],
  decorators: [
    (Story) => (
      <div className="max-w-lg mx-auto p-4">
        <Story />
      </div>
    ),
  ],
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component: `
A complete payment form component integrating Stripe Elements for secure card payments.

**Features:**
- Stripe Elements integration for PCI-compliant card handling
- Billing address collection with validation
- Payment summary display
- Loading and processing states
- Success/failure result screens
- Accessibility support
- Responsive design

**Note:** This component requires a valid Stripe client secret to function properly.
In production, you would obtain this from your backend after creating a PaymentIntent.
        `,
      },
    },
  },
  argTypes: {
    amount: {
      control: 'number',
      description: 'Payment amount in cents',
    },
    currency: {
      control: 'select',
      options: ['usd', 'eur', 'gbp', 'cad', 'aud'],
      description: 'Currency code',
    },
    clientSecret: {
      control: 'text',
      description: 'Stripe PaymentIntent client secret',
    },
    collectBillingAddress: {
      control: 'boolean',
      description: 'Whether to collect billing address',
    },
    allowSavePaymentMethod: {
      control: 'boolean',
      description: 'Show option to save card for future use',
    },
    submitButtonText: {
      control: 'text',
      description: 'Custom submit button text',
    },
    onSuccess: {
      action: 'success',
      description: 'Callback when payment succeeds',
    },
    onError: {
      action: 'error',
      description: 'Callback when payment fails',
    },
    onCancel: {
      action: 'cancel',
      description: 'Callback when user cancels',
    },
    className: {
      control: 'text',
      description: 'Additional CSS classes',
    },
  },
};

export default meta;
type Story = StoryObj<typeof PaymentForm>;

const mockClientSecret = 'pi_test_secret_placeholder';

const basicSummary: PaymentSummaryData = {
  lineItems: [
    {
      id: '1',
      description: 'Pro Subscription - Monthly',
      quantity: 1,
      unitPrice: 2999,
      totalPrice: 2999,
    },
  ],
  subtotal: 2999,
  tax: 240,
  taxRate: 8,
  total: 3239,
  currency: 'usd',
};

export const Default: Story = {
  args: {
    amount: 3239,
    currency: 'usd',
    clientSecret: mockClientSecret,
    onSuccess: action('payment-success'),
    onError: action('payment-error'),
    onCancel: action('payment-cancel'),
  },
};

export const WithSummary: Story = {
  args: {
    amount: 3239,
    currency: 'usd',
    clientSecret: mockClientSecret,
    summary: basicSummary,
    onSuccess: action('payment-success'),
    onError: action('payment-error'),
    onCancel: action('payment-cancel'),
  },
  parameters: {
    docs: {
      description: {
        story: 'Payment form with order summary displayed above the form',
      },
    },
  },
};

export const WithDetailedSummary: Story = {
  args: {
    amount: 6476,
    currency: 'usd',
    clientSecret: mockClientSecret,
    summary: {
      lineItems: [
        {
          id: '1',
          description: 'Pro Subscription - Monthly',
          quantity: 1,
          unitPrice: 2999,
          totalPrice: 2999,
        },
        {
          id: '2',
          description: 'Analytics Add-on',
          quantity: 1,
          unitPrice: 999,
          totalPrice: 999,
        },
        {
          id: '3',
          description: 'Priority Support',
          quantity: 1,
          unitPrice: 1499,
          totalPrice: 1499,
        },
        {
          id: '4',
          description: 'API Access',
          quantity: 1,
          unitPrice: 499,
          totalPrice: 499,
        },
      ],
      subtotal: 5996,
      tax: 480,
      taxRate: 8,
      total: 6476,
      currency: 'usd',
    },
    onSuccess: action('payment-success'),
    onError: action('payment-error'),
    onCancel: action('payment-cancel'),
  },
};

export const WithDiscount: Story = {
  args: {
    amount: 2739,
    currency: 'usd',
    clientSecret: mockClientSecret,
    summary: {
      ...basicSummary,
      discount: 500,
      discountCode: 'SAVE15',
      total: 2739,
    },
    onSuccess: action('payment-success'),
    onError: action('payment-error'),
    onCancel: action('payment-cancel'),
  },
};

export const NoBillingAddress: Story = {
  args: {
    amount: 3239,
    currency: 'usd',
    clientSecret: mockClientSecret,
    collectBillingAddress: false,
    summary: basicSummary,
    onSuccess: action('payment-success'),
    onError: action('payment-error'),
    onCancel: action('payment-cancel'),
  },
  parameters: {
    docs: {
      description: {
        story: 'Simplified form without billing address collection',
      },
    },
  },
};

export const WithSaveCard: Story = {
  args: {
    amount: 3239,
    currency: 'usd',
    clientSecret: mockClientSecret,
    allowSavePaymentMethod: true,
    summary: basicSummary,
    onSuccess: action('payment-success'),
    onError: action('payment-error'),
    onCancel: action('payment-cancel'),
  },
  parameters: {
    docs: {
      description: {
        story: 'Option to save card for future payments',
      },
    },
  },
};

export const PrefilledAddress: Story = {
  args: {
    amount: 3239,
    currency: 'usd',
    clientSecret: mockClientSecret,
    defaultBillingAddress: {
      name: 'John Doe',
      email: 'john@example.com',
      phone: '+1 (555) 123-4567',
      line1: '123 Main Street',
      city: 'San Francisco',
      state: 'CA',
      postalCode: '94102',
      country: 'US',
    },
    summary: basicSummary,
    onSuccess: action('payment-success'),
    onError: action('payment-error'),
    onCancel: action('payment-cancel'),
  },
  parameters: {
    docs: {
      description: {
        story: 'Pre-filled billing address for returning customers',
      },
    },
  },
};

export const CustomButtonText: Story = {
  args: {
    amount: 2999,
    currency: 'usd',
    clientSecret: mockClientSecret,
    submitButtonText: 'Subscribe Now',
    summary: {
      ...basicSummary,
      lineItems: [
        {
          id: '1',
          description: 'Pro Plan - Annual',
          quantity: 1,
          unitPrice: 2999,
          totalPrice: 2999,
        },
      ],
      tax: 0,
      total: 2999,
    },
    onSuccess: action('payment-success'),
    onError: action('payment-error'),
    onCancel: action('payment-cancel'),
  },
};

export const EuroCurrency: Story = {
  args: {
    amount: 4999,
    currency: 'eur',
    clientSecret: mockClientSecret,
    summary: {
      lineItems: [
        {
          id: '1',
          description: 'Business Plan',
          quantity: 1,
          unitPrice: 4582,
          totalPrice: 4582,
        },
      ],
      subtotal: 4582,
      tax: 417,
      taxRate: 21,
      total: 4999,
      currency: 'eur',
    },
    onSuccess: action('payment-success'),
    onError: action('payment-error'),
    onCancel: action('payment-cancel'),
  },
};

export const GBPCurrency: Story = {
  args: {
    amount: 3999,
    currency: 'gbp',
    clientSecret: mockClientSecret,
    summary: {
      lineItems: [
        {
          id: '1',
          description: 'Premium Plan',
          quantity: 1,
          unitPrice: 3332,
          totalPrice: 3332,
        },
      ],
      subtotal: 3332,
      tax: 667,
      taxRate: 20,
      total: 3999,
      currency: 'gbp',
    },
    onSuccess: action('payment-success'),
    onError: action('payment-error'),
    onCancel: action('payment-cancel'),
  },
};

export const LargeAmount: Story = {
  args: {
    amount: 1250000,
    currency: 'usd',
    clientSecret: mockClientSecret,
    summary: {
      lineItems: [
        {
          id: '1',
          description: 'Enterprise Plan - Annual',
          quantity: 1,
          unitPrice: 1157407,
          totalPrice: 1157407,
        },
      ],
      subtotal: 1157407,
      tax: 92593,
      taxRate: 8,
      total: 1250000,
      currency: 'usd',
    },
    onSuccess: action('payment-success'),
    onError: action('payment-error'),
    onCancel: action('payment-cancel'),
  },
};

export const NoCancelOption: Story = {
  args: {
    amount: 3239,
    currency: 'usd',
    clientSecret: mockClientSecret,
    summary: basicSummary,
    onSuccess: action('payment-success'),
    onError: action('payment-error'),
    // No onCancel - cancel button won't appear
  },
  parameters: {
    docs: {
      description: {
        story: 'Without cancel callback, the cancel link is hidden',
      },
    },
  },
};

export const MobileView: Story = {
  args: {
    amount: 3239,
    currency: 'usd',
    clientSecret: mockClientSecret,
    summary: basicSummary,
    onSuccess: action('payment-success'),
    onError: action('payment-error'),
    onCancel: action('payment-cancel'),
  },
  parameters: {
    viewport: {
      defaultViewport: 'mobile1',
    },
    docs: {
      description: {
        story: 'Responsive layout on mobile devices',
      },
    },
  },
};

export const MinimalForm: Story = {
  args: {
    amount: 999,
    currency: 'usd',
    clientSecret: mockClientSecret,
    collectBillingAddress: false,
    allowSavePaymentMethod: false,
    onSuccess: action('payment-success'),
    onError: action('payment-error'),
  },
  parameters: {
    docs: {
      description: {
        story: 'Minimal payment form with just card input',
      },
    },
  },
};

export const FullFeatured: Story = {
  args: {
    amount: 6476,
    currency: 'usd',
    clientSecret: mockClientSecret,
    collectBillingAddress: true,
    allowSavePaymentMethod: true,
    defaultBillingAddress: {
      name: 'Jane Smith',
      email: 'jane@company.com',
    },
    summary: {
      lineItems: [
        {
          id: '1',
          description: 'Team Plan - Monthly',
          quantity: 1,
          unitPrice: 4999,
          totalPrice: 4999,
        },
        {
          id: '2',
          description: 'Additional Seats',
          quantity: 5,
          unitPrice: 200,
          totalPrice: 1000,
        },
      ],
      subtotal: 5999,
      tax: 477,
      taxRate: 8,
      discount: 500,
      discountCode: 'TEAM15',
      total: 5976,
      currency: 'usd',
    },
    onSuccess: action('payment-success'),
    onError: action('payment-error'),
    onCancel: action('payment-cancel'),
  },
  parameters: {
    docs: {
      description: {
        story: 'Full-featured payment form with all options enabled',
      },
    },
  },
};
